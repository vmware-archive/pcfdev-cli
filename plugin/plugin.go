package plugin

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/cloudfoundry/cli/plugin"
	"github.com/pivotal-cf/pcfdev-cli/pivnet"
	"github.com/pivotal-cf/pcfdev-cli/user"
	"github.com/pivotal-cf/pcfdev-cli/vbox"
)

type Plugin struct {
	SSH                 SSH
	UI                  UI
	VBox                VBox
	RequirementsChecker RequirementsChecker
	Client              Client
	Downloader          Downloader

	VMName string
}

//go:generate mockgen -package mocks -destination mocks/ssh.go github.com/pivotal-cf/pcfdev-cli/plugin SSH
type SSH interface {
	RunSSHCommand(command string, port string, timeout time.Duration, stdout io.Writer, stderr io.Writer) error
}

//go:generate mockgen -package mocks -destination mocks/ui.go github.com/pivotal-cf/pcfdev-cli/plugin UI
type UI interface {
	Failed(message string, args ...interface{})
	Say(message string, args ...interface{})
	Confirm(message string, args ...interface{}) bool
	Ask(prompt string, args ...interface{}) (answer string)
}

//go:generate mockgen -package mocks -destination mocks/vbox.go github.com/pivotal-cf/pcfdev-cli/plugin VBox
type VBox interface {
	StartVM(name string) (vm *vbox.VM, err error)
	StopVM(name string) error
	DestroyVMs(name []string) error
	ImportVM(path string, name string) error
	Status(name string) (status string, err error)
	ConflictingVMPresent(name string) (conflict bool, err error)
	GetPCFDevVMs() (names []string, err error)
}

//go:generate mockgen -package mocks -destination mocks/requirements_checker.go github.com/pivotal-cf/pcfdev-cli/plugin RequirementsChecker
type RequirementsChecker interface {
	Check() error
}

//go:generate mockgen -package mocks -destination mocks/downloader.go github.com/pivotal-cf/pcfdev-cli/plugin Downloader
type Downloader interface {
	Download(path string) error
	IsOVACurrent(path string) (bool, error)
}

//go:generate mockgen -package mocks -destination mocks/client.go github.com/pivotal-cf/pcfdev-cli/plugin Client
type Client interface {
	AcceptEULA() error
	IsEULAAccepted() (bool, error)
	GetEULA() (eula string, err error)
}

func (p *Plugin) Run(cliConnection plugin.CliConnection, args []string) {
	if args[0] == "CLI-MESSAGE-UNINSTALL" {
		return
	}

	if len(args) != 2 {
		p.UI.Failed("Usage: %s", p.GetMetadata().Commands[0].UsageDetails.Usage)
		return
	}

	switch args[1] {
	case "download":
		if err := p.downloadVM(); err != nil {
			p.UI.Failed(getErrorText(err))
		}
	case "start":
		if err := p.start(); err != nil {
			p.UI.Failed(getErrorText(err))
		}
	case "status":
		if status, err := p.VBox.Status(p.VMName); err != nil {
			p.UI.Failed(getErrorText(err))
		} else {
			p.UI.Say(status)
		}
	case "stop":
		if err := p.stop(); err != nil {
			p.UI.Failed(getErrorText(err))
		}
	case "destroy":
		if err := p.destroy(); err != nil {
			p.UI.Failed(getErrorText(err))
		}
	default:
		p.UI.Failed("'%s' is not a registered command.\nUsage: %s", args[1], p.GetMetadata().Commands[0].UsageDetails.Usage)
	}
}

func getErrorText(err error) string {
	switch err.(type) {
	case *EULARefusedError:
		return "You must accept the end user license agreement to use PCF Dev."
	case *pivnet.InvalidTokenError:
		return "Invalid Pivotal Network API Token."
	case *pivnet.PivNetUnreachableError:
		return "Failed to reach Pivotal Network. Please try again later."
	case *OldVMError:
		return "An old version of PCF Dev was detected. You must run 'cf dev destroy' to continue."
	case *StartError:
		return "Failed to start PCF Dev VM."
	case *ImportVMError:
		return "Failed to import PCF Dev VM."
	case *ProvisionVMError:
		return "Failed to provision PCF Dev VM."
	case *StopVMError:
		return "Failed to stop PCF Dev VM."
	case *DestroyVMError:
		return "Failed to destroy PCF Dev VM."
	default:
		return fmt.Sprintf("Error: %s", err.Error())
	}
}

func (p *Plugin) downloadVM() error {
	ovaPath, err := p.ovaPath()
	if err != nil {
		return err
	}
	current, err := p.Downloader.IsOVACurrent(ovaPath)
	if err != nil {
		panic(err)
	}
	if current {
		p.UI.Say("Using existing image")
		return nil
	}

	accepted, err := p.Client.IsEULAAccepted()
	if err != nil {
		return err
	}

	if !accepted {
		eula, err := p.Client.GetEULA()
		if err != nil {
			return err
		}

		p.UI.Say(eula)

		if accepted := p.UI.Confirm("Accept (yes/no):"); !accepted {
			return &EULARefusedError{}
		}

		if err := p.Client.AcceptEULA(); err != nil {
			return err
		}
	}

	p.UI.Say("Downloading VM...")

	if err := p.Downloader.Download(ovaPath); err != nil {
		return err
	}
	p.UI.Say("\nVM downloaded")
	return nil
}

func (p *Plugin) start() error {
	if err := p.RequirementsChecker.Check(); err != nil {
		if accepted := p.UI.Confirm("Less than 3 GB of memory detected, continue (y/N): "); !accepted {
			p.UI.Say("Exiting...")
			return nil
		}
	}

	status, err := p.VBox.Status(p.VMName)
	if err != nil {
		return &StartError{err}
	}

	if status == vbox.StatusRunning {
		p.UI.Say("PCF Dev is running")
		return nil
	}

	if status == vbox.StatusNotCreated {
		conflict, err := p.VBox.ConflictingVMPresent(p.VMName)
		if err != nil {
			return &StartError{err}
		}
		if conflict {
			return &OldVMError{}
		}

		if err := p.downloadVM(); err != nil {
			return err
		}

		ovaPath, err := p.ovaPath()
		if err != nil {
			return err
		}

		p.UI.Say("Importing VM...")
		err = p.VBox.ImportVM(ovaPath, p.VMName)
		if err != nil {
			return &ImportVMError{err}
		}
		p.UI.Say("PCF Dev is now imported to Virtualbox")
	}

	p.UI.Say("Starting VM...")
	vm, err := p.VBox.StartVM(p.VMName)
	if err != nil {
		return &StartError{err}
	}
	p.UI.Say("Provisioning VM...")
	err = p.provision(vm)
	if err != nil {
		return &ProvisionVMError{err}
	}

	p.UI.Say("PCF Dev is now running")
	return nil
}

func (p *Plugin) stop() error {
	status, err := p.VBox.Status(p.VMName)
	if err != nil {
		return &StopVMError{err}
	}

	if status == vbox.StatusNotCreated {
		conflict, err := p.VBox.ConflictingVMPresent(p.VMName)
		if err != nil {
			return &StopVMError{err}
		}
		if conflict {
			return &OldVMError{}
		}
		p.UI.Say("PCF Dev VM has not been created")
		return nil
	}

	if status == vbox.StatusStopped {
		p.UI.Say("PCF Dev is stopped")
		return nil
	}

	p.UI.Say("Stopping VM...")
	err = p.VBox.StopVM(p.VMName)
	if err != nil {
		return &StopVMError{err}
	}
	p.UI.Say("PCF Dev is now stopped")
	return nil
}

func (p *Plugin) destroy() error {
	vms, err := p.VBox.GetPCFDevVMs()
	if err != nil {
		return &DestroyVMError{err}
	}

	if len(vms) == 0 {
		p.UI.Say("PCF Dev VM has not been created")
		return nil
	}

	p.UI.Say("Destroying VM...")
	err = p.VBox.DestroyVMs(vms)
	if err != nil {
		return &DestroyVMError{err}
	}
	p.UI.Say("PCF Dev VM has been destroyed")
	return nil
}

func (p *Plugin) provision(vm *vbox.VM) error {
	return p.SSH.RunSSHCommand(fmt.Sprintf("sudo /var/pcfdev/run %s %s '$2a$04$EpJtIJ8w6hfCwbKYBkn3t.GCY18Pk6s7yN66y37fSJlLuDuMkdHtS'", vm.Domain, vm.IP), vm.SSHPort, 2*time.Minute, os.Stdout, os.Stderr)
}

func (p *Plugin) pcfdevDir() (path string, err error) {
	if pcfdevHome := os.Getenv("PCFDEV_HOME"); pcfdevHome != "" {
		return filepath.Join(pcfdevHome, ".pcfdev"), nil
	}

	homeDir, err := user.GetHome()
	if err != nil {
		return "", fmt.Errorf("failed to find home directory: %s", err)
	}

	return filepath.Join(homeDir, ".pcfdev"), nil
}

func (p *Plugin) ovaPath() (path string, err error) {
	pcfdevDir, err := p.pcfdevDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(pcfdevDir, p.VMName+".ova"), nil
}

func (*Plugin) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name: "pcfdev",
		Commands: []plugin.Command{
			plugin.Command{
				Name:  "dev",
				Alias: "pcfdev",
				UsageDetails: plugin.Usage{
					Usage: "cf dev download|start|status|stop|destroy",
				},
			},
		},
	}
}

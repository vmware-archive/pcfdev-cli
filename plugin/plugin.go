package plugin

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/cloudfoundry/cli/plugin"
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
			p.UI.Failed(fmt.Sprintf("Error: %s", err.Error()))
		}
	case "start":
		if err := p.start(); err != nil {
			p.UI.Failed(fmt.Sprintf("Error: %s", err.Error()))
		}
	case "status":
		if status, err := p.VBox.Status(p.VMName); err != nil {
			p.UI.Failed(fmt.Sprintf("Error: %s", err.Error()))
		} else {
			p.UI.Say(status)
		}
	case "stop":
		if err := p.stop(); err != nil {
			p.UI.Failed(fmt.Sprintf("Error: %s", err.Error()))
		}
	case "destroy":
		if err := p.destroy(); err != nil {
			p.UI.Failed(fmt.Sprintf("Error: %s", err.Error()))
		}
	}
}

func (p *Plugin) downloadVM() error {
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

		accepted := p.UI.Confirm("Accept (yes/no):")
		if !accepted {
			return nil
		}

		if err := p.Client.AcceptEULA(); err != nil {
			return err
		}
	}

	p.UI.Say("Downloading VM...")
	ovaPath, err := p.ovaPath()
	if err != nil {
		return err
	}

	if err := p.Downloader.Download(ovaPath); err != nil {
		return err
	}
	p.UI.Say("\nVM downloaded")
	return nil
}

func (p *Plugin) start() error {
	if err := p.RequirementsChecker.Check(); err != nil {
		return fmt.Errorf("could not start PCF Dev: %s", err)
	}

	status, err := p.VBox.Status(p.VMName)
	if err != nil {
		return fmt.Errorf("failed to get VM status: %s", err)
	}

	if status == vbox.StatusRunning {
		p.UI.Say("PCF Dev is running")
		return nil
	}

	if status == vbox.StatusNotCreated {
		conflict, err := p.VBox.ConflictingVMPresent(p.VMName)
		if err != nil {
			return err
		}
		if conflict {
			return errors.New("old version of PCF Dev detected, you must run `cf dev destroy` to continue.")
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
			return fmt.Errorf("failed to import VM: %s", err)
		}
		p.UI.Say("PCF Dev is now imported to Virtualbox")
	}

	p.UI.Say("Starting VM...")
	vm, err := p.VBox.StartVM(p.VMName)
	if err != nil {
		return fmt.Errorf("failed to start VM: %s", err)
	}
	p.UI.Say("Provisioning VM...")
	err = p.provision(vm)
	if err != nil {
		return fmt.Errorf("failed to provision VM: %s", err)
	}

	p.UI.Say("PCF Dev is now running")
	return nil
}

func (p *Plugin) stop() error {
	status, err := p.VBox.Status(p.VMName)
	if err != nil {
		return err
	}

	if status == vbox.StatusNotCreated {
		conflict, err := p.VBox.ConflictingVMPresent(p.VMName)
		if err != nil {
			return err
		}
		if conflict {
			return errors.New("Old version of PCF Dev detected. You must run `cf dev destroy` to continue.")
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
		return fmt.Errorf("failed to stop VM: %s", err)
	}
	p.UI.Say("PCF Dev is now stopped")
	return nil
}

func (p *Plugin) destroy() error {
	vms, err := p.VBox.GetPCFDevVMs()
	if err != nil {
		return fmt.Errorf("failed to query VM: %s", err)
	}

	if len(vms) == 0 {
		p.UI.Say("PCF Dev VM has not been created")
		return nil
	}

	p.UI.Say("Destroying VM...")
	err = p.VBox.DestroyVMs(vms)
	if err != nil {
		return fmt.Errorf("failed to destroy VM: %s", err)
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

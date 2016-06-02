package plugin

import (
	"fmt"
	"io"
	"time"

	"github.com/cloudfoundry/cli/plugin"
	"github.com/pivotal-cf/pcfdev-cli/vm"
)

type Plugin struct {
	SSH                 SSH
	UI                  UI
	VBox                VBox
	Client              Client
	Config              Config
	Downloader          Downloader
	Builder             Builder
	RequirementsChecker RequirementsChecker
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
	GetPCFDevVMs() (names []string, err error)
}

//go:generate mockgen -package mocks -destination mocks/downloader.go github.com/pivotal-cf/pcfdev-cli/plugin Downloader
type Downloader interface {
	Download() error
	IsOVACurrent() (bool, error)
}

//go:generate mockgen -package mocks -destination mocks/client.go github.com/pivotal-cf/pcfdev-cli/plugin Client
type Client interface {
	AcceptEULA() error
	IsEULAAccepted() (bool, error)
	GetEULA() (eula string, err error)
}

//go:generate mockgen -package mocks -destination mocks/config.go github.com/pivotal-cf/pcfdev-cli/plugin Config
type Config interface {
	GetVMName() string
}

//go:generate mockgen -package mocks -destination mocks/builder.go github.com/pivotal-cf/pcfdev-cli/plugin Builder
type Builder interface {
	VM(name string) (vm vm.VM, err error)
}

//go:generate mockgen -package mocks -destination mocks/requirements_checker.go github.com/pivotal-cf/pcfdev-cli/plugin RequirementsChecker
type RequirementsChecker interface {
	Check() error
}

//go:generate mockgen -package mocks -destination mocks/vm.go github.com/pivotal-cf/pcfdev-cli/vm VM

func (p *Plugin) Run(cliConnection plugin.CliConnection, args []string) {
	if args[0] == "CLI-MESSAGE-UNINSTALL" {
		return
	}

	var subcommand string
	if len(args) > 1 {
		subcommand = args[1]
	}

	switch subcommand {
	case "download":
		if err := p.download(); err != nil {
			p.UI.Failed(getErrorText(err))
		}
	case "start":
		if err := p.start(); err != nil {
			p.UI.Failed(getErrorText(err))
		}
	case "status":
		if err := p.status(); err != nil {
			p.UI.Failed(getErrorText(err))
		}
	case "stop":
		if err := p.stop(); err != nil {
			p.UI.Failed(getErrorText(err))
		}
	case "suspend":
		if err := p.suspend(); err != nil {
			p.UI.Failed(getErrorText(err))
		}
	case "resume":
		if err := p.resume(); err != nil {
			p.UI.Failed(getErrorText(err))
		}
	case "destroy":
		if err := p.destroy(); err != nil {
			p.UI.Failed(getErrorText(err))
		}
	default:
		_, err := cliConnection.CliCommand("help", "dev")
		if err != nil {
			p.UI.Failed(getErrorText(err))
		}
	}
}

func getErrorText(err error) string {
	return fmt.Sprintf("Error: %s", err.Error())
}

func (p *Plugin) start() error {
	if err := p.RequirementsChecker.Check(); err != nil {
		if !p.UI.Confirm("Less than 3 GB of memory detected, continue (y/N): ") {
			p.UI.Say("Exiting...")
			return nil
		}
	}

	if err := p.download(); err != nil {
		return err
	}
	vm, err := p.Builder.VM(p.Config.GetVMName())
	if err != nil {
		return err
	}
	return vm.Start()
}

func (p *Plugin) status() error {
	vm, err := p.Builder.VM(p.Config.GetVMName())
	if err != nil {
		return err
	}
	vm.Status()
	return nil
}

func (p *Plugin) stop() error {
	vm, err := p.Builder.VM(p.Config.GetVMName())
	if err != nil {
		return err
	}
	return vm.Stop()
}

func (p *Plugin) suspend() error {
	vm, err := p.Builder.VM(p.Config.GetVMName())
	if err != nil {
		return err
	}

	return vm.Suspend()
}

func (p *Plugin) resume() error {
	if err := p.RequirementsChecker.Check(); err != nil {
		if !p.UI.Confirm("Less than 3 GB of memory detected, continue (y/N): ") {
			p.UI.Say("Exiting...")
			return nil
		}
	}

	vm, err := p.Builder.VM(p.Config.GetVMName())
	if err != nil {
		return err
	}

	return vm.Resume()
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
	for _, vmName := range vms {
		vm, err := p.Builder.VM(vmName)
		if err != nil {
			return &DestroyVMError{err}
		}
		if err := vm.Destroy(); err != nil {
			return &DestroyVMError{err}
		}
	}
	p.UI.Say("PCF Dev VM has been destroyed")
	return nil
}

func (p *Plugin) download() error {
	current, err := p.Downloader.IsOVACurrent()
	if err != nil {
		return err
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

	if err := p.Downloader.Download(); err != nil {
		return err
	}

	p.UI.Say("\nVM downloaded")
	return nil
}

func (*Plugin) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name: "pcfdev",
		Commands: []plugin.Command{
			plugin.Command{
				Name:     "dev",
				Alias:    "pcfdev",
				HelpText: "Control PCF Dev VMs running on your workstation",
				UsageDetails: plugin.Usage{
					Usage: `cf dev SUBCOMMAND

SUBCOMMANDS:
   start                    Start the PCF Dev VM. When creating a VM, http proxy env vars are respected.
   stop                     Shutdown the PCF Dev VM. All data is preserved.
   suspend                  Save the current state of the PCF Dev VM to disk and then stop the VM.
   resume                   Resume PCF Dev VM from suspended state.
   destroy                  Delete the PCF Dev VM. All data is destroyed.
   status                   Query for the status of the PCF Dev VM.
					`,
				},
			},
		},
	}
}

package plugin

import (
	"errors"
	"fmt"

	"github.com/cloudfoundry/cli/flags"
	"github.com/cloudfoundry/cli/plugin"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/vm"
)

type Plugin struct {
	UI         UI
	FS         FS
	VBox       VBox
	Client     Client
	Config     *config.Config
	Downloader Downloader
	Builder    Builder
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
	GetVMName() (name string, err error)
	DestroyPCFDevVMs() (err error)
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

//go:generate mockgen -package mocks -destination mocks/builder.go github.com/pivotal-cf/pcfdev-cli/plugin Builder
type Builder interface {
	VM(name string) (vm vm.VM, err error)
}

//go:generate mockgen -package mocks -destination mocks/fs.go github.com/pivotal-cf/pcfdev-cli/plugin FS
type FS interface {
	Remove(path string) error
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

	flagContext := flags.New()

	switch subcommand {
	case "start":
		flagContext.NewIntFlag("m", "memory", "<memory in MB>")
		flagContext.NewIntFlag("c", "cpus", "<number of cpus>")
		flagContext.NewStringFlag("o", "ova", "<path to custom ova>")
		flagContext.NewStringFlag("s", "services", "<services to start with>")
		flagContext.NewBoolFlag("n", "", "<bool for provisioning>")
	}

	if err := flagContext.Parse(args...); err != nil {
		p.showUsageMessage(cliConnection)
		return
	}

	switch subcommand {
	case "download":
		if err := p.download(); err != nil {
			p.UI.Failed(getErrorText(err))
		}
	case "start":
		if err := p.start(flagContext); err != nil {
			p.UI.Failed(getErrorText(err))
		}
	case "provision":
		if err := p.provision(); err != nil {
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
		p.showUsageMessage(cliConnection)
	}
}

func (p *Plugin) showUsageMessage(cliConnection plugin.CliConnection) {
	_, err := cliConnection.CliCommand("help", "dev")
	if err != nil {
		p.UI.Failed(getErrorText(err))
	}
}

func getErrorText(err error) string {
	return fmt.Sprintf("Error: %s.", err.Error())
}

func (p *Plugin) start(flagContext flags.FlagContext) error {
	var name string

	if flagContext.IsSet("o") {
		name = "pcfdev-custom"
	} else {
		name = p.Config.DefaultVMName
	}

	existingVMName, err := p.VBox.GetVMName()
	if err != nil {
		return err
	}
	if existingVMName != "" {
		if flagContext.IsSet("o") {
			if existingVMName != "pcfdev-custom" {
				return errors.New("you must destroy your existing VM to use a custom OVA")
			}
		} else {
			if existingVMName != p.Config.DefaultVMName && existingVMName != "pcfdev-custom" {
				return &OldVMError{}
			}
		}
	}

	if existingVMName == "pcfdev-custom" {
		name = "pcfdev-custom"
	}

	v, err := p.Builder.VM(name)
	if err != nil {
		return err
	}

	opts := &vm.StartOpts{
		Memory:      uint64(flagContext.Int("m")),
		CPUs:        flagContext.Int("c"),
		OVAPath:     flagContext.String("o"),
		Services:    flagContext.String("s"),
		NoProvision: flagContext.Bool("n"),
	}

	if err := v.VerifyStartOpts(opts); err != nil {
		return err
	}
	if !flagContext.IsSet("o") && existingVMName != "pcfdev-custom" {
		if err := p.download(); err != nil {
			return err
		}
	}

	return v.Start(opts)
}

func (p *Plugin) provision() error {
	vm, err := p.getVM()
	if err != nil {
		return err
	}

	return vm.Provision()
}

func (p *Plugin) status() error {
	vm, err := p.getVM()
	if err != nil {
		return err
	}
	p.UI.Say(vm.Status())
	return nil
}

func (p *Plugin) stop() error {
	vm, err := p.getVM()
	if err != nil {
		return err
	}
	return vm.Stop()
}

func (p *Plugin) suspend() error {
	vm, err := p.getVM()
	if err != nil {
		return err
	}
	return vm.Suspend()
}

func (p *Plugin) resume() error {
	vm, err := p.getVM()
	if err != nil {
		return err
	}
	return vm.Resume()
}

func (p *Plugin) destroy() error {
	if err := p.VBox.DestroyPCFDevVMs(); err != nil {
		p.UI.Failed(fmt.Sprintf("Error destroying PCF Dev VM: %s.", err))
	} else {
		p.UI.Say("PCF Dev VM has been destroyed.")
	}

	if err := p.FS.Remove(p.Config.VMDir); err != nil {
		p.UI.Failed(fmt.Sprintf("Error removing %s: %s.", p.Config.VMDir, err))
	}

	return nil
}

func (p *Plugin) download() error {
	existingVMName, err := p.VBox.GetVMName()
	if err != nil {
		return err
	}
	if existingVMName != "" && existingVMName != p.Config.DefaultVMName {
		return &OldVMError{}
	}

	current, err := p.Downloader.IsOVACurrent()
	if err != nil {
		return err
	}
	if current {
		p.UI.Say("Using existing image.")
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

	p.UI.Say("\nVM downloaded.")
	return nil
}

func (p *Plugin) getVM() (vm vm.VM, err error) {
	name, err := p.VBox.GetVMName()
	if err != nil {
		return nil, err
	}
	if name == "" {
		name = p.Config.DefaultVMName
	}
	if name != p.Config.DefaultVMName && name != "pcfdev-custom" {
		return nil, &OldVMError{}
	}

	return p.Builder.VM(name)
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
   start                       Start the PCF Dev VM. When creating a VM, http proxy env vars are respected.
      [-m memory-in-mb]        Memory to allocate for VM. Default: half of system memory, no more than 4 GB.
      [-c number-of-cores]     Number of processor cores used by VM. Default: number of physical cores.
      [-s service1,service2]   Specify the services started with PCF Dev.
                                  Options: redis, rabbitmq, spring-cloud-services (scs), default, all, none
                                  Default: redis, rabbitmq
   stop                        Shutdown the PCF Dev VM. All data is preserved.
   suspend                     Save the current state of the PCF Dev VM to disk and then stop the VM.
   resume                      Resume PCF Dev VM from suspended state.
   destroy                     Delete the PCF Dev VM. All data is destroyed.
   status                      Query for the status of the PCF Dev VM.
					`,
				},
			},
		},
	}
}

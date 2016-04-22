package plugin

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/cli/plugin"
	"github.com/pivotal-cf/pcfdev-cli/vbox"
)

type Plugin struct {
	PivnetClient Client
	SSH          SSH
	UI           UI
	VBox         VBox
	FS           FS
}

//go:generate mockgen -package mocks -destination mocks/client.go github.com/pivotal-cf/pcfdev-cli/plugin Client
type Client interface {
	DownloadOVA() (io.ReadCloser, error)
}

//go:generate mockgen -package mocks -destination mocks/ssh.go github.com/pivotal-cf/pcfdev-cli/plugin SSH
type SSH interface {
	RunSSHCommand(command string, port string) error
}

//go:generate mockgen -package mocks -destination mocks/ui.go github.com/pivotal-cf/pcfdev-cli/plugin UI
type UI interface {
	Failed(message string, args ...interface{})
	Say(message string, args ...interface{})
}

//go:generate mockgen -package mocks -destination mocks/vbox.go github.com/pivotal-cf/pcfdev-cli/plugin VBox
type VBox interface {
	StartVM(string) (*vbox.VM, error)
	StopVM(string) error
	DestroyVM(string) error
	ImportVM(string, string) error
	Status(string) (string, error)
}

//go:generate mockgen -package mocks -destination mocks/fs.go github.com/pivotal-cf/pcfdev-cli/plugin FS
type FS interface {
	Exists(string) (bool, error)
	Write(string, io.ReadCloser) error
	CreateDir(string) error
}

const vmName = "pcfdev-2016-03-29_1728"

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
			p.UI.Failed(err.Error())
		}
	case "start":
		if err := p.start(); err != nil {
			p.UI.Failed(err.Error())
		}
	case "status":
		if status, err := p.VBox.Status(vmName); err != nil {
			p.UI.Failed(err.Error())
		} else {
			p.UI.Say(status)
		}
	case "stop":
		if err := p.stop(); err != nil {
			p.UI.Failed(err.Error())
		}
	case "destroy":
		if err := p.destroy(); err != nil {
			p.UI.Failed(err.Error())
		}
	}
}

func (p *Plugin) downloadVM() error {
	if _, err := p.getOVAFile(); err != nil {
		return fmt.Errorf("failed to fetch VM: %s", err)
	}

	return nil
}

func (p *Plugin) start() error {
	status, err := p.VBox.Status(vmName)
	if err != nil {
		return fmt.Errorf("failed to get VM status: %s", err)
	}

	if status == vbox.StatusRunning {
		p.UI.Say("PCF Dev is running")
		return nil
	}

	path, err := p.getOVAFile()
	if err != nil {
		return fmt.Errorf("failed to fetch VM: %s", err)
	}

	if status == vbox.StatusNotCreated {
		p.UI.Say("Importing VM...")
		err = p.VBox.ImportVM(path, vmName)
		if err != nil {
			return fmt.Errorf("failed to import VM: %s", err)
		}
		p.UI.Say("PCF Dev is now imported to Virtualbox")
	}

	p.UI.Say("Starting VM...")
	vm, err := p.VBox.StartVM(vmName)
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
	status, err := p.VBox.Status(vmName)
	if err != nil {
		return err
	}

	if status == vbox.StatusNotCreated {
		p.UI.Say("PCF Dev VM has not been created")
		return nil
	}

	if status == vbox.StatusStopped {
		p.UI.Say("PCF Dev is stopped")
		return nil
	}

	p.UI.Say("Stopping VM...")
	err = p.VBox.StopVM(vmName)
	if err != nil {
		return fmt.Errorf("failed to stop VM: %s", err)
	}
	p.UI.Say("PCF Dev is now stopped")
	return nil
}

func (p *Plugin) destroy() error {
	status, err := p.VBox.Status(vmName)
	if err != nil {
		return err
	}

	if status == vbox.StatusNotCreated {
		p.UI.Say("PCF Dev VM has not been created")
		return nil
	}

	p.UI.Say("Destroying VM...")
	err = p.VBox.DestroyVM(vmName)
	if err != nil {
		return fmt.Errorf("failed to destroy VM: %s", err)
	}
	p.UI.Say("PCF Dev VM has been destroyed")
	return nil
}

func (p *Plugin) provision(vm *vbox.VM) error {
	return p.SSH.RunSSHCommand(fmt.Sprintf("sudo /var/pcfdev/run local.pcfdev.io %s", vm.IP), vm.SSHPort)
}

func (p *Plugin) getOVAFile() (string, error) {
	pcfdevDir := filepath.Join(os.Getenv("HOME"), ".pcfdev")
	err := p.FS.CreateDir(pcfdevDir)
	if err != nil {
		return "", err
	}

	path := filepath.Join(pcfdevDir, "pcfdev.ova")
	ovaExists, err := p.FS.Exists(path)
	if err != nil {
		return "", err
	}
	if !ovaExists {
		p.UI.Say("Downloading VM...")
		ova, err := p.PivnetClient.DownloadOVA()
		if err != nil {
			return "", err
		}
		p.FS.Write(path, ova)
		p.UI.Say("Finished downloading VM")
	} else {
		p.UI.Say("VM already downloaded")
	}
	return path, nil
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

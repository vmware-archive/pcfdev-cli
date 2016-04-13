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
	StartVM(string, string) (*vbox.VM, error)
	StopVM(string) error
	IsVMRunning(string) bool
}

//go:generate mockgen -package mocks -destination mocks/fs.go github.com/pivotal-cf/pcfdev-cli/plugin FS
type FS interface {
	Exists(string) (bool, error)
	Write(string, io.ReadCloser) error
	CreateDir(string) error
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
	case "start":
		p.start()
	case "stop":
		p.stop()
	}
}

func (p *Plugin) start() {
	if p.VBox.IsVMRunning("pcfdev-2016-03-29_1728") {
		p.UI.Say("PCFDev is already running")
		return
	}

	ovaPath, err := p.getOvaFile()
	if err != nil {
		p.UI.Failed("failed to fetch OVA: %s", err)
		return
	}

	p.UI.Say("Starting VM...")
	vm, err := p.VBox.StartVM(ovaPath, "pcfdev-2016-03-29_1728")
	if err != nil {
		p.UI.Failed("failed to start VM: %s", err)
		return
	}
	p.UI.Say("Provisioning VM...")
	err = p.provision(vm)
	if err != nil {
		p.UI.Failed("failed to provision VM: %s", err)
		return
	}

	p.UI.Say("PCFDev is now running")
}

func (p *Plugin) stop() {
	p.UI.Say("Stopping VM...")
	err := p.VBox.StopVM("pcfdev-2016-03-29_1728")
	if err != nil {
		p.UI.Failed("failed to stop VM: %s", err)
		return
	}
	p.UI.Say("PCFDev is now stopped")
}

func (p *Plugin) provision(vm *vbox.VM) error {
	return p.SSH.RunSSHCommand(fmt.Sprintf("sudo /var/pcfdev/run local.pcfdev.io %s", vm.IP), vm.SSHPort)
}

func (p *Plugin) getOvaFile() (string, error) {
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
		p.UI.Say("Downloading OVA...")
		ova, err := p.PivnetClient.DownloadOVA()
		if err != nil {
			return "", err
		}
		p.FS.Write(path, ova)
		p.UI.Say("Finished downloading OVA")
	} else {
		p.UI.Say("pcfdev.ova already downloaded")
	}
	return path, nil
}

func (*Plugin) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name: "PCFDev",
		Commands: []plugin.Command{
			plugin.Command{
				Name:  "dev",
				Alias: "pcfdev",
				UsageDetails: plugin.Usage{
					Usage: "cf dev start|stop",
				},
			},
		},
	}
}

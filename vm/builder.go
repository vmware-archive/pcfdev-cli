package vm

import (
	"os"

	"github.com/cloudfoundry/cli/cf/terminal"
	"github.com/pivotal-cf/pcfdev-cli/address"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/network"
	"github.com/pivotal-cf/pcfdev-cli/ping"
	"github.com/pivotal-cf/pcfdev-cli/ssh"
	"github.com/pivotal-cf/pcfdev-cli/system"
	"github.com/pivotal-cf/pcfdev-cli/vbox"
)

//go:generate mockgen -package mocks -destination mocks/driver.go github.com/pivotal-cf/pcfdev-cli/vm Driver
type Driver interface {
	VMExists(vmName string) (exists bool, err error)
	IsVMRunning(vmName string) bool
	GetVMIP(vmName string) (vmIP string, err error)
	GetHostForwardPort(vmName string, ruleName string) (port string, err error)
}

type VBoxBuilder struct {
	Config *config.Config
	Driver Driver
}

func (b *VBoxBuilder) VM(vmName string) (VM, error) {
	termUI := terminal.NewUI(os.Stdin, terminal.NewTeePrinter())
	ssh := &ssh.SSH{}
	system := &system.System{}
	vbox := &vbox.VBox{
		SSH:    ssh,
		Driver: &vbox.VBoxDriver{},
		Picker: &address.Picker{
			Pinger:  &ping.Pinger{},
			Network: &network.Network{},
		},
		Config: b.Config,
		System: system,
	}

	exists, err := b.Driver.VMExists(vmName)
	if err != nil {
		return nil, err
	}
	if !exists {
		return &NotCreated{
			Name:    vmName,
			VBox:    vbox,
			UI:      termUI,
			Builder: b,
		}, nil
	}

	ip, err := b.Driver.GetVMIP(vmName)
	if err != nil {
		return nil, err
	}
	domain, err := address.DomainForIP(ip)
	if err != nil {
		return nil, err
	}
	sshPort, err := b.Driver.GetHostForwardPort(vmName, "ssh")
	if err != nil {
		return nil, err
	}

	if b.Driver.IsVMRunning(vmName) {
		return &Running{
			Name:    vmName,
			IP:      ip,
			SSHPort: sshPort,
			Domain:  domain,

			UI:   termUI,
			VBox: vbox,
		}, nil
	}
	return &Stopped{
		Name:    vmName,
		IP:      ip,
		SSHPort: sshPort,
		Domain:  domain,

		UI:   termUI,
		SSH:  ssh,
		VBox: vbox,
	}, nil
}

package vm

import (
	"fmt"
	"os"

	"github.com/cloudfoundry/cli/cf/terminal"
	"github.com/pivotal-cf/pcfdev-cli/address"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/fs"
	"github.com/pivotal-cf/pcfdev-cli/network"
	"github.com/pivotal-cf/pcfdev-cli/ssh"
	"github.com/pivotal-cf/pcfdev-cli/vbox"
)

//go:generate mockgen -package mocks -destination mocks/driver.go github.com/pivotal-cf/pcfdev-cli/vm Driver
type Driver interface {
	VMExists(vmName string) (exists bool, err error)
	VMState(vmName string) (state string, err error)
	GetVMIP(vmName string) (vmIP string, err error)
	GetMemory(vmName string) (memory uint64, err error)
	GetHostForwardPort(vmName string, ruleName string) (port string, err error)
}

type VBoxBuilder struct {
	Config *config.Config
	Driver Driver
}

func (b *VBoxBuilder) VM(vmName string) (VM, error) {
	termUI := terminal.NewUI(os.Stdin, terminal.NewTeePrinter())
	ssh := &ssh.SSH{}
	vbx := &vbox.VBox{
		SSH:    ssh,
		FS:     &fs.FS{},
		Driver: &vbox.VBoxDriver{},
		Picker: &address.Picker{
			Driver:  &vbox.VBoxDriver{},
			Network: &network.Network{},
		},
		Config: b.Config,
	}

	exists, err := b.Driver.VMExists(vmName)
	if err != nil {
		return nil, err
	}

	if !exists {
		return &NotCreated{
			VBox:    vbx,
			UI:      termUI,
			Builder: b,
			Config:  b.Config,
			VMConfig: &config.VMConfig{
				Name: vmName,
			},
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
	memory, err := b.Driver.GetMemory(vmName)
	if err != nil {
		return nil, err
	}
	sshPort, err := b.Driver.GetHostForwardPort(vmName, "ssh")
	if err != nil {
		return nil, err
	}

	state, err := b.Driver.VMState(vmName)
	if err != nil {
		return nil, err
	}
	if state == vbox.StateRunning {
		return &Running{
			VMConfig: &config.VMConfig{
				Name:    vmName,
				IP:      ip,
				SSHPort: sshPort,
				Domain:  domain,
				Memory:  memory,
			},

			UI:   termUI,
			VBox: vbx,
		}, nil
	}

	if state == vbox.StateSaved || state == vbox.StatePaused {
		return &Suspended{
			VMConfig: &config.VMConfig{
				Name:    vmName,
				IP:      ip,
				SSHPort: sshPort,
				Domain:  domain,
				Memory:  memory,
			},
			Config: b.Config,

			UI:   termUI,
			VBox: vbx,
		}, nil
	}

	if state == vbox.StateStopped || state == vbox.StateAborted {
		return &Stopped{
			VMConfig: &config.VMConfig{
				Name:    vmName,
				IP:      ip,
				SSHPort: sshPort,
				Domain:  domain,
				Memory:  memory,
			},
			Config: b.Config,

			UI:   termUI,
			SSH:  ssh,
			VBox: vbx,
		}, nil
	}

	return nil, fmt.Errorf("failed to handle VM state '%s'", state)
}

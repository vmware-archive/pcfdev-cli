package vm

import (
	"fmt"
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
	VMState(vmName string) (state string, err error)
	GetVMIP(vmName string) (vmIP string, err error)
	GetHostForwardPort(vmName string, ruleName string) (port string, err error)
}

//go:generate mockgen -package mocks -destination mocks/system.go github.com/pivotal-cf/pcfdev-cli/vm System
type System interface {
	FreeMemory() (memory uint64, err error)
}

type VBoxBuilder struct {
	Config *config.Config
	Driver Driver
	System System
}

func (b *VBoxBuilder) VM(vmName string, vmConfig *config.VMConfig) (VM, error) {
	termUI := terminal.NewUI(os.Stdin, terminal.NewTeePrinter())
	ssh := &ssh.SSH{}
	system := &system.System{}
	vbx := &vbox.VBox{
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
		return b.buildNotCreatedVM(vmName, vmConfig, vbx, termUI)
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

	state, err := b.Driver.VMState(vmName)
	if err != nil {
		return nil, err
	}
	if state == vbox.StateRunning {
		return &Running{
			Name:    vmName,
			IP:      ip,
			SSHPort: sshPort,
			Domain:  domain,

			UI:   termUI,
			VBox: vbx,
		}, nil
	}

	if state == vbox.StateSaved {
		return &Suspended{
			Name:    vmName,
			IP:      ip,
			SSHPort: sshPort,
			Domain:  domain,
			UI:      termUI,
			VBox:    vbx,
		}, nil
	}

	if state == vbox.StateStopped {
		return &Stopped{
			Name:    vmName,
			IP:      ip,
			SSHPort: sshPort,
			Domain:  domain,
			UI:      termUI,
			SSH:     ssh,
			VBox:    vbx,
		}, nil
	}

	return nil, fmt.Errorf("failed to handle VM state '%s'", state)
}

func (b *VBoxBuilder) buildNotCreatedVM(vmName string, vmConfig *config.VMConfig, vbx VBox, termUI UI) (VM, error) {
	var desiredMemory uint64
	var err error

	if vmConfig.DesiredMemory != uint64(0) {
		desiredMemory = vmConfig.DesiredMemory
	} else {
		desiredMemory, err = b.computeMemory(vmConfig.DesiredMemory)
		if err != nil {
			return nil, err
		}
	}

	return &NotCreated{
		Name:    vmName,
		VBox:    vbx,
		UI:      termUI,
		Builder: b,
		Config:  &config.VMConfig{DesiredMemory: desiredMemory},
	}, nil
}

func (b *VBoxBuilder) computeMemory(desiredMemory uint64) (uint64, error) {
	var memory uint64
	if desiredMemory != 0 {
		memory = desiredMemory
	} else {
		maxMemory := b.Config.MaxMemory
		minMemory := b.Config.MinMemory
		freeMemory, err := b.System.FreeMemory()
		if err != nil {
			return uint64(0), err
		}
		if freeMemory <= minMemory {
			memory = minMemory
		} else if freeMemory >= maxMemory {
			memory = maxMemory
		} else {
			memory = freeMemory
		}
	}
	return memory, nil
}

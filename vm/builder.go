package vm

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudfoundry/cli/cf/terminal"
	"github.com/pivotal-cf/pcfdev-cli/address"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/fs"
	"github.com/pivotal-cf/pcfdev-cli/network"
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
	FS     FS
	SSH    SSH
}

func (b *VBoxBuilder) VM(vmName string) (VM, error) {
	termUI := terminal.NewUI(os.Stdin, terminal.NewTeePrinter())
	vbx := &vbox.VBox{
		SSH:    b.SSH,
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
		dirExists, err := b.FS.Exists(filepath.Join(b.Config.VMDir, vmName))
		if err != nil {
			return nil, err
		}

		if dirExists {
			return &Invalid{
				Err: errors.New("VM files need to be purged"),
				UI:  termUI,
			}, nil
		}

		return &NotCreated{
			VBox:    vbx,
			UI:      termUI,
			Builder: b,
			Config:  b.Config,
			FS:      b.FS,
			VMConfig: &config.VMConfig{
				Name: vmName,
			},
		}, nil
	}

	ip, err := b.Driver.GetVMIP(vmName)
	if err != nil {
		return &Invalid{
			Err: err,
			UI:  termUI,
		}, nil
	}
	domain, err := address.DomainForIP(ip)
	if err != nil {
		return &Invalid{
			Err: err,
			UI:  termUI,
		}, nil
	}
	memory, err := b.Driver.GetMemory(vmName)
	if err != nil {
		return nil, err
	}
	sshPort, err := b.Driver.GetHostForwardPort(vmName, "ssh")
	if err != nil {
		return &Invalid{
			Err: err,
			UI:  termUI,
		}, nil
	}

	state, err := b.Driver.VMState(vmName)
	if err != nil {
		return nil, err
	}
	if state == vbox.StateRunning {
		if output, err := b.healthcheck(ip, sshPort); strings.TrimSpace(output) != "ok" || err != nil {
			return &Unprovisioned{
				VMConfig: &config.VMConfig{
					Name:    vmName,
					IP:      ip,
					SSHPort: sshPort,
					Domain:  domain,
					Memory:  memory,
				},
				Config: &config.Config{
					VMDir: b.Config.VMDir,
				},

				UI:   termUI,
				VBox: vbx,
				FS:   b.FS,
				SSH:  b.SSH,
			}, nil
		} else {
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

			SSH:  b.SSH,
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

			FS:      b.FS,
			UI:      termUI,
			SSH:     b.SSH,
			VBox:    vbx,
			Builder: b,
		}, nil
	}

	return nil, fmt.Errorf("failed to handle VM state '%s'", state)
}

func (b *VBoxBuilder) healthcheck(ip string, sshPort string) (string, error) {
	healthCheckCommand := "sudo /var/pcfdev/health-check"

	forwardPortOutputChan := make(chan string, 1)
	forwardPortErrChan := make(chan error, 1)
	sshOutputChan := make(chan string, 1)
	sshErrChan := make(chan error, 1)

	go func() {
		output, err := b.SSH.GetSSHOutput(healthCheckCommand, "127.0.0.1", sshPort, 20*time.Second)
		forwardPortOutputChan <- output
		forwardPortErrChan <- err
	}()
	go func() {
		output, err := b.SSH.GetSSHOutput(healthCheckCommand, ip, "22", 20*time.Second)
		sshOutputChan <- output
		sshErrChan <- err
	}()

	select {
	case out := <-sshOutputChan:
		return out, <-sshErrChan
	case out := <-forwardPortOutputChan:
		return out, <-forwardPortErrChan
	}
}

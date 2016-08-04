package vm

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudfoundry/cli/cf/terminal"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/vbox"
)

type VBoxBuilder struct {
	Config *config.Config
	VBox   VBox
	FS     FS
	SSH    SSH
}

func (b *VBoxBuilder) VM(vmName string) (VM, error) {
	termUI := terminal.NewUI(os.Stdin, terminal.NewTeePrinter())

	exists, err := b.VBox.VMExists(vmName)
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
			VBox:    b.VBox,
			UI:      termUI,
			Builder: b,
			Config:  b.Config,
			FS:      b.FS,
			VMConfig: &config.VMConfig{
				Name: vmName,
			},
		}, nil
	}

	vmConfig, err := b.VBox.VMConfig(vmName)
	if err != nil {
		return &Invalid{
			Err: err,
			UI:  termUI,
		}, nil
	}

	state, err := b.VBox.VMState(vmName)
	if err != nil {
		return &Invalid{
			Err: err,
			UI:  termUI,
		}, nil
	}

	if state == vbox.StateRunning {
		if output, err := b.healthcheck(vmConfig.IP, vmConfig.SSHPort); strings.TrimSpace(output) != "ok" || err != nil {
			return &Unprovisioned{
				VMConfig: vmConfig,
				Config: &config.Config{
					VMDir: b.Config.VMDir,
				},
				UI:   termUI,
				VBox: b.VBox,
				FS:   b.FS,
				SSH:  b.SSH,
			}, nil
		} else {
			return &Running{
				VMConfig: vmConfig,
				UI:       termUI,
				VBox:     b.VBox,
				SSH:      b.SSH,
				Builder:  b,
			}, nil
		}
	}

	if state == vbox.StateSaved || state == vbox.StatePaused {
		return &Suspended{
			VMConfig: vmConfig,
			Config:   b.Config,
			SSH:      b.SSH,
			UI:       termUI,
			VBox:     b.VBox,
		}, nil
	}

	if state == vbox.StateStopped || state == vbox.StateAborted {
		return &Stopped{
			VMConfig: vmConfig,
			Config:   b.Config,

			FS:      b.FS,
			UI:      termUI,
			SSH:     b.SSH,
			VBox:    b.VBox,
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

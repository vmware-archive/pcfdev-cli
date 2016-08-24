package vm

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudfoundry/cli/cf/terminal"
	"github.com/cloudfoundry/cli/cf/trace"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/fs"
	"github.com/pivotal-cf/pcfdev-cli/vbox"
)

type VBoxBuilder struct {
	Config *config.Config
	VBox   VBox
	FS     FS
	SSH    SSH
}

func (b *VBoxBuilder) VM(vmName string) (VM, error) {
	termUI := terminal.NewUI(
		os.Stdin,
		os.Stdout,
		terminal.NewTeePrinter(os.Stdout),
		trace.NewWriterPrinter(os.Stdout, true),
	)

	status, err := b.VBox.VMStatus(vmName)
	if err != nil {
		return nil, err
	}

	vmConfig, err := b.getVMConfig(vmName, status)
	if err != nil {
		return &Invalid{
			Err: err,
			UI:  termUI,
		}, nil
	}

	switch status {
	case vbox.StatusNotCreated:
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
			VBox:     b.VBox,
			UI:       termUI,
			Builder:  b,
			Config:   b.Config,
			FS:       b.FS,
			VMConfig: vmConfig,
		}, nil
	case vbox.StatusRunning:
		if output, err := b.healthcheck(vmConfig.IP, vmConfig.SSHPort); strings.TrimSpace(output) != "ok" || err != nil {
			return &Unprovisioned{
				VMConfig: vmConfig,
				Config:   b.Config,
				UI:       termUI,
				VBox:     b.VBox,
				FS:       b.FS,
				SSH:      b.SSH,
				LogFetcher: &ConcreteLogFetcher{
					VMConfig: vmConfig,
					UI:       termUI,
					FS:       b.FS,
					SSH:      b.SSH,
					Driver: &vbox.VBoxDriver{
						FS: &fs.FS{},
					},
				},
			}, nil
		} else {
			return &Running{
				VMConfig: vmConfig,
				FS:       b.FS,
				UI:       termUI,
				VBox:     b.VBox,
				SSH:      b.SSH,
				Builder:  b,
				LogFetcher: &ConcreteLogFetcher{
					VMConfig: vmConfig,
					UI:       termUI,
					FS:       b.FS,
					SSH:      b.SSH,
					Driver: &vbox.VBoxDriver{
						FS: &fs.FS{},
					},
				},
			}, nil
		}
	case vbox.StatusStopped:
		return &Stopped{
			VMConfig: vmConfig,
			Config:   b.Config,

			FS:      b.FS,
			UI:      termUI,
			SSH:     b.SSH,
			VBox:    b.VBox,
			Builder: b,
		}, nil
	case vbox.StatusPaused:
		return &Paused{
			VMConfig: vmConfig,
			SSH:      b.SSH,
			UI:       termUI,
			VBox:     b.VBox,
		}, nil
	case vbox.StatusSaved:
		return &Saved{
			VMConfig: vmConfig,
			SSH:      b.SSH,
			UI:       termUI,
			VBox:     b.VBox,
			Config:   b.Config,
		}, nil
	default:
		return &Invalid{
			Err: errors.New("vm in unknown state"),
			UI:  termUI,
		}, nil
	}
}

func (b *VBoxBuilder) getVMConfig(vmName string, status string) (*config.VMConfig, error) {
	if status == vbox.StatusNotCreated {
		return &config.VMConfig{
			Name: vmName,
		}, nil
	}
	return b.VBox.VMConfig(vmName)
}

func (b *VBoxBuilder) healthcheck(ip string, sshPort string) (string, error) {
	healthCheckCommand := "sudo /var/pcfdev/health-check"
	outputChan := make(chan string, 1)
	errChan := make(chan error, 1)

	go func() {
		output, err := b.SSH.GetSSHOutput(healthCheckCommand, "127.0.0.1", sshPort, 20*time.Second)
		outputChan <- output
		errChan <- err
	}()
	go func() {
		output, err := b.SSH.GetSSHOutput(healthCheckCommand, ip, "22", 20*time.Second)
		outputChan <- output
		errChan <- err
	}()

	return <-outputChan, <-errChan
}

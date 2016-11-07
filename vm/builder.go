package vm

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"fmt"

	"github.com/cloudfoundry/cli/cf/terminal"
	"github.com/cloudfoundry/cli/cf/trace"
	"github.com/pivotal-cf/pcfdev-cli/cert"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/debug"
	"github.com/pivotal-cf/pcfdev-cli/fs"
	"github.com/pivotal-cf/pcfdev-cli/network"
	"github.com/pivotal-cf/pcfdev-cli/runner"
	"github.com/pivotal-cf/pcfdev-cli/ssh"
	"github.com/pivotal-cf/pcfdev-cli/ui"
	"github.com/pivotal-cf/pcfdev-cli/vbox"
	"github.com/pivotal-cf/pcfdev-cli/vm/client"
	"github.com/pivotal-cf/pcfdev-cli/vboxdriver"
)

type VBoxBuilder struct {
	Config *config.Config
	VBox   VBox
	FS     FS
	SSH    SSH
	Client Client
}

func (b *VBoxBuilder) VM(vmName string) (VM, error) {
	termUI := terminal.NewUI(
		os.Stdin,
		os.Stdout,
		terminal.NewTeePrinter(os.Stdout),
		trace.NewLogger(os.Stdout, false, "", ""),
	)

	status, err := b.VBox.VMStatus(vmName)
	if err != nil {
		return nil, err
	}

	vmConfig, err := b.getVMConfig(vmName, status)
	if err != nil {
		return &Invalid{
			Err: err,
		}, nil
	}

	unprovisionedVm := &Unprovisioned{
		VMConfig:  vmConfig,
		Config:    b.Config,
		UI:        termUI,
		VBox:      b.VBox,
		FS:        b.FS,
		SSHClient: b.SSH,
		HelpText: &ui.HelpText{
			UI: termUI,
		},
		Client: b.Client,
		LogFetcher: &debug.LogFetcher{
			VMConfig: vmConfig,
			Config:   b.Config,
			FS:       b.FS,
			SSH:      b.SSH,
			Driver: &vboxdriver.VBoxDriver{
				FS:        &fs.FS{},
				CmdRunner: &runner.CmdRunner{},
			},
		},
	}
	runningVm := &Running{
		Config:    b.Config,
		VMConfig:  vmConfig,
		FS:        b.FS,
		UI:        termUI,
		VBox:      b.VBox,
		SSHClient: b.SSH,
		Builder:   b,
		CmdRunner: &runner.CmdRunner{},
		HelpText: &ui.HelpText{
			UI: termUI,
		},
		CertStore: &cert.CertStore{
			FS: b.FS,
			SystemStore: &cert.ConcreteSystemStore{
				FS:        b.FS,
				CmdRunner: &runner.CmdRunner{},
			},
		},
		LogFetcher: &debug.LogFetcher{
			VMConfig: vmConfig,
			Config:   b.Config,
			FS:       b.FS,
			SSH:      b.SSH,
			Driver: &vboxdriver.VBoxDriver{
				FS:        &fs.FS{},
				CmdRunner: &runner.CmdRunner{},
			},
		},
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
			}, nil
		}

		return &NotCreated{
			VBox:     b.VBox,
			UI:       termUI,
			Builder:  b,
			Config:   b.Config,
			FS:       b.FS,
			VMConfig: vmConfig,
			Network:  &network.Network{},
		}, nil
	case vbox.StatusRunning:
		output, err := b.Client.Status(fmt.Sprintf("http://%s:%d", vmConfig.Domain, client.APIPort))
		if output == "Unprovisioned" || err != nil {
			return unprovisionedVm, nil
		} else if output == "Running" {
			return runningVm, nil
		} else {
			return &Invalid{
				Err: errors.New("vm in unknown state"),
			}, nil
		}

	case vbox.StatusStopped:
		return &Stopped{
			VMConfig: vmConfig,
			Config:   b.Config,

			FS:        b.FS,
			UI:        termUI,
			SSHClient: b.SSH,
			VBox:      b.VBox,
			Builder:   b,
		}, nil
	case vbox.StatusPaused:
		return &Paused{
			VMConfig:  vmConfig,
			SSHClient: b.SSH,
			UI:        termUI,
			VBox:      b.VBox,
			Config:    b.Config,
			FS:        b.FS,
		}, nil
	case vbox.StatusSaved:
		return &Saved{
			VMConfig:  vmConfig,
			SSHClient: b.SSH,
			UI:        termUI,
			VBox:      b.VBox,
			Config:    b.Config,
			FS:        b.FS,
		}, nil
	default:
		return &Invalid{
			Err: errors.New("vm in unknown state"),
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
	privateKeyBytes, err := b.FS.Read(b.Config.PrivateKeyPath)
	if err != nil {
		return "", err
	}

	return b.SSH.GetSSHOutput(
		healthCheckCommand,
		[]ssh.SSHAddress{
			{
				IP:   "127.0.0.1",
				Port: sshPort,
			},
			{
				IP:   ip,
				Port: "22",
			},
		},
		privateKeyBytes,
		20*time.Second,
	)
}

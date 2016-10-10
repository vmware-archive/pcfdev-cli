package vm

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pivotal-cf/pcfdev-cli/config"
)

type Unprovisioned struct {
	FS         FS
	SSHClient  SSH
	UI         UI
	VBox       VBox
	LogFetcher LogFetcher
	Config     *config.Config
	VMConfig   *config.VMConfig
	HelpText   HelpText
}

func (u *Unprovisioned) Stop() error {
	u.UI.Say("Stopping VM...")
	u.VBox.StopVM(u.VMConfig)
	u.UI.Say("PCF Dev is now stopped.")
	return nil
}

func (u *Unprovisioned) VerifyStartOpts(opts *StartOpts) error {
	return u.err()
}

func (u *Unprovisioned) Start(opts *StartOpts) error {
	return u.err()
}

func (u *Unprovisioned) Status() string {
	return u.err().Error()
}

func (u *Unprovisioned) Provision(opts *StartOpts) error {
	privateKeyBytes, err := u.FS.Read(u.Config.PrivateKeyPath)
	if err != nil {
		return err
	}

	if err := u.SSHClient.RunSSHCommand("if [ -e /var/pcfdev/provision-options.json ]; then exit 0; else exit 1; fi", "127.0.0.1", u.VMConfig.SSHPort, privateKeyBytes, 30*time.Second, os.Stdout, os.Stderr); err != nil {
		return &ProvisionVMError{errors.New("missing provision configuration")}
	}

	data, err := u.SSHClient.GetSSHOutput("cat /var/pcfdev/provision-options.json", "127.0.0.1", u.VMConfig.SSHPort, privateKeyBytes, 30*time.Second)
	if err != nil {
		return &ProvisionVMError{err}
	}

	provisionConfig := &config.ProvisionConfig{}
	if err := json.Unmarshal([]byte(data), provisionConfig); err != nil {
		return &ProvisionVMError{err}
	}

	u.UI.Say("Provisioning VM...")
	provisionCommand := fmt.Sprintf(`sudo -H /var/pcfdev/provision "%s" "%s" "%s" "%s"`, provisionConfig.Domain, provisionConfig.IP, provisionConfig.Services, strings.Join(provisionConfig.Registries, ","))
	if err := u.SSHClient.RunSSHCommand(provisionCommand, "127.0.0.1", u.VMConfig.SSHPort, privateKeyBytes, 5*time.Minute, os.Stdout, os.Stderr); err != nil {
		return &ProvisionVMError{err}
	}

	u.HelpText.Print(u.VMConfig.Domain, opts.Target)

	return nil
}

func (u *Unprovisioned) Suspend() error {
	return u.err()
}

func (u *Unprovisioned) Resume() error {
	return u.err()
}

func (u *Unprovisioned) Trust(startOps *StartOpts) error {
	return u.err()
}

func (u *Unprovisioned) Target(autoTarget bool) error {
	return u.err()
}

func (u *Unprovisioned) SSH() error {
	privateKeyBytes, err := u.FS.Read(u.Config.PrivateKeyPath)
	if err != nil {
		return err
	}

	return u.SSHClient.StartSSHSession("127.0.0.1", u.VMConfig.SSHPort, privateKeyBytes, 5*time.Minute, os.Stdin, os.Stdout, os.Stderr)
}

func (u *Unprovisioned) err() error {
	return errors.New("PCF Dev is in an invalid state. Please run 'cf dev destroy' or 'cf dev stop'")
}

func (u *Unprovisioned) GetDebugLogs() error {
	if err := u.LogFetcher.FetchLogs(); err != nil {
		return &FetchLogsError{err}
	}

	return nil
}

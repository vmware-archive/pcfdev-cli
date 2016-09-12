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
	SSH        SSH
	UI         UI
	VBox       VBox
	LogFetcher LogFetcher
	Config     *config.Config
	VMConfig   *config.VMConfig
}

func (u *Unprovisioned) Stop() error {
	u.UI.Say("Stopping VM...")
	u.VBox.StopVM(u.VMConfig)
	u.UI.Say("PCF Dev is now stopped.")
	return nil
}

func (u *Unprovisioned) VerifyStartOpts(opts *StartOpts) error {
	return errors.New(u.err())
}

func (u *Unprovisioned) Start(opts *StartOpts) error {
	u.UI.Failed(u.message())
	return nil
}

func (u *Unprovisioned) Status() string {
	return u.message()
}

func (u *Unprovisioned) Provision() error {
	if err := u.SSH.RunSSHCommand("if [ -e /var/pcfdev/provision-options.json ]; then exit 0; else exit 1; fi", "127.0.0.1", u.VMConfig.SSHPort, 30*time.Second, os.Stdout, os.Stderr); err != nil {
		return &ProvisionVMError{errors.New("missing provision configuration")}
	}

	data, err := u.SSH.GetSSHOutput("cat /var/pcfdev/provision-options.json", "127.0.0.1", u.VMConfig.SSHPort, 30*time.Second)
	if err != nil {
		return &ProvisionVMError{err}
	}

	provisionConfig := &config.ProvisionConfig{}
	if err := json.Unmarshal([]byte(data), provisionConfig); err != nil {
		return &ProvisionVMError{err}
	}

	u.UI.Say("Provisioning VM...")
	provisionCommand := fmt.Sprintf(`sudo -H /var/pcfdev/provision "%s" "%s" "%s" "%s"`, provisionConfig.Domain, provisionConfig.IP, provisionConfig.Services, strings.Join(provisionConfig.Registries, ","))
	if err := u.SSH.RunSSHCommand(provisionCommand, "127.0.0.1", u.VMConfig.SSHPort, 5*time.Minute, os.Stdout, os.Stderr); err != nil {
		return &ProvisionVMError{err}
	}

	return nil
}

func (u *Unprovisioned) Suspend() error {
	u.UI.Failed(u.message())
	return nil
}

func (u *Unprovisioned) Resume() error {
	u.UI.Failed(u.message())
	return nil
}

func (u *Unprovisioned) message() string {
	return u.err() + "."
}

func (u *Unprovisioned) Trust() error {
	u.UI.Failed(u.message())
	return nil
}

func (u *Unprovisioned) err() string {
	return "PCF Dev is in an invalid state. Please run 'cf dev destroy' or 'cf dev stop' before attempting to start again"
}

func (u *Unprovisioned) GetDebugLogs() error {
	if err := u.LogFetcher.FetchLogs(); err != nil {
		return &FetchLogsError{err}
	}

	return nil
}

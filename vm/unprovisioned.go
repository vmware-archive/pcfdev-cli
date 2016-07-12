package vm

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pivotal-cf/pcfdev-cli/config"
)

type Unprovisioned struct {
	FS       FS
	SSH      SSH
	UI       UI
	VBox     VBox
	Config   *config.Config
	VMConfig *config.VMConfig
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
	if exists, err := u.FS.Exists(filepath.Join(u.Config.VMDir, "provision-options")); !exists || err != nil {
		return &ProvisionVMError{errors.New("missing provision configuration")}
	}

	data, err := u.FS.Read(filepath.Join(u.Config.VMDir, "provision-options"))
	if err != nil {
		return &ProvisionVMError{err}
	}

	provisionConfig := &config.ProvisionConfig{}
	if err := json.Unmarshal(data, provisionConfig); err != nil {
		return &ProvisionVMError{err}
	}

	u.UI.Say("Provisioning VM...")
	provisionCommand := fmt.Sprintf("sudo -H /var/pcfdev/run %s %s %s", provisionConfig.Domain, provisionConfig.IP, provisionConfig.Services)
	if err := u.SSH.RunSSHCommand(provisionCommand, u.VMConfig.SSHPort, 5*time.Minute, os.Stdout, os.Stderr); err != nil {
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

func (u *Unprovisioned) err() string {
	return "PCF Dev is in an invalid state. Please run 'cf dev destroy' or 'cf dev stop' before attempting to start again"
}

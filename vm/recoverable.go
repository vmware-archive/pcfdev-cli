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

type Recoverable struct {
	FS       FS
	SSH      SSH
	UI       UI
	VBox     VBox
	Config   *config.Config
	VMConfig *config.VMConfig
}

func (r *Recoverable) Stop() error {
	r.UI.Say("Stopping VM...")
	r.VBox.StopVM(r.VMConfig)
	r.UI.Say("PCF Dev is now stopped.")
	return nil
}

func (r *Recoverable) VerifyStartOpts(opts *StartOpts) error {
	return errors.New(r.err())
}

func (r *Recoverable) Start(opts *StartOpts) error {
	r.UI.Failed(r.message())
	return nil
}

func (r *Recoverable) Status() string {
	return r.message()
}

func (r *Recoverable) Provision() error {
	if exists, err := r.FS.Exists(filepath.Join(r.Config.VMDir, "provision-options")); !exists || err != nil {
		return &ProvisionVMError{errors.New("missing provision configuration")}
	}

	data, err := r.FS.Read(filepath.Join(r.Config.VMDir, "provision-options"))
	if err != nil {
		return &ProvisionVMError{err}
	}

	provisionConfig := &config.ProvisionConfig{}
	if err := json.Unmarshal(data, provisionConfig); err != nil {
		return &ProvisionVMError{err}
	}

	r.UI.Say("Provisioning VM...")
	provisionCommand := fmt.Sprintf("sudo -H /var/pcfdev/run %s %s %s", provisionConfig.Domain, provisionConfig.IP, provisionConfig.Services)
	if err := r.SSH.RunSSHCommand(provisionCommand, r.VMConfig.SSHPort, 5*time.Minute, os.Stdout, os.Stderr); err != nil {
		return &ProvisionVMError{err}
	}

	return nil
}

func (r *Recoverable) Suspend() error {
	r.UI.Failed(r.message())
	return nil
}

func (r *Recoverable) Resume() error {
	r.UI.Failed(r.message())
	return nil
}

func (r *Recoverable) message() string {
	return r.err() + "."
}

func (r *Recoverable) err() string {
	return "PCF Dev is in an invalid state. Please run 'cf dev destroy' or 'cf dev stop' before attempting to start again"
}

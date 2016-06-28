package vm

import (
	"errors"
	"fmt"

	"github.com/pivotal-cf/pcfdev-cli/config"
)

type Running struct {
	VMConfig *config.VMConfig

	VBox VBox
	UI   UI
}

func (r *Running) Stop() error {
	r.UI.Say("Stopping VM...")
	err := r.VBox.StopVM(r.VMConfig)
	if err != nil {
		return &StopVMError{err}
	}
	r.UI.Say("PCF Dev is now stopped")
	return nil
}

func (r *Running) VerifyStartOpts(opts *StartOpts) error {
	if opts.Memory != uint64(0) {
		return errors.New("memory cannot be changed once the vm has been created")
	}
	if opts.CPUs != 0 {
		return errors.New("cores cannot be changed once the vm has been created")
	}
	return nil
}

func (r *Running) Start(opts *StartOpts) error {
	r.UI.Say("PCF Dev is running")
	return nil
}

func (r *Running) Status() string {
	return fmt.Sprintf("Running\nLogin: cf login -a https://api.%s --skip-ssl-validation\nAdmin user => Email: admin / Password: admin\nRegular user => Email: user / Password: pass", r.VMConfig.Domain)
}

func (r *Running) Suspend() error {
	r.UI.Say("Suspending VM...")
	if err := r.VBox.SuspendVM(r.VMConfig); err != nil {
		return &SuspendVMError{err}
	}

	r.UI.Say("PCF Dev is now suspended")
	return nil
}

func (r *Running) Resume() error {
	r.UI.Say("PCF Dev is running")

	return nil
}

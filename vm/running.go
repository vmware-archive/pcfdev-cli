package vm

import "github.com/pivotal-cf/pcfdev-cli/config"

type Running struct {
	Name    string
	Domain  string
	IP      string
	SSHPort string
	Config  *config.VMConfig

	VBox VBox
	UI   UI
}

func (r *Running) Stop() error {
	r.UI.Say("Stopping VM...")
	err := r.VBox.StopVM(r.Name)
	if err != nil {
		return &StopVMError{err}
	}
	r.UI.Say("PCF Dev is now stopped")
	return nil
}

func (r *Running) Start() error {
	r.UI.Say("PCF Dev is running")
	return nil
}

func (r *Running) Status() {
	r.UI.Say("Running")
}

func (r *Running) Destroy() error {
	if err := r.VBox.PowerOffVM(r.Name); err != nil {
		return &DestroyVMError{err}
	}
	if err := r.VBox.DestroyVM(r.Name); err != nil {
		return &DestroyVMError{err}
	}
	return nil
}

func (r *Running) Suspend() error {
	r.UI.Say("Suspending VM...")
	if err := r.VBox.SuspendVM(r.Name); err != nil {
		return &SuspendVMError{err}
	}

	r.UI.Say("PCF Dev is now suspended")
	return nil
}

func (r *Running) Resume() error {
	r.UI.Say("PCF Dev is running")

	return nil
}

func (r *Running) GetConfig() *config.VMConfig {
	return r.Config
}

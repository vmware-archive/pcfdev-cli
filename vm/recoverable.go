package vm

import (
	"errors"

	"github.com/pivotal-cf/pcfdev-cli/config"
)

type Recoverable struct {
	UI       UI
	VBox     VBox
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

package vm

import "errors"

type Invalid struct {
	Err error
}

func (i *Invalid) Stop() error {
	return i.err()
}

func (i *Invalid) VerifyStartOpts(opts *StartOpts) error {
	return i.err()
}

func (i *Invalid) Start(opts *StartOpts) error {
	return i.err()
}

func (i *Invalid) Provision(opts *StartOpts) error {
	return i.err()
}

func (i *Invalid) Status() string {
	return i.message()
}

func (i *Invalid) Suspend() error {
	return i.err()
}

func (i *Invalid) Resume() error {
	return i.err()
}

func (i *Invalid) GetDebugLogs() error {
	return i.err()
}

func (i *Invalid) Trust(startOps *StartOpts) error {
	return i.err()
}

func (i *Invalid) Target(autoTarget bool) error {
	return i.err()
}

func (i *Invalid) SSH(opts *SSHOpts) error {
	return i.err()
}

func (i *Invalid) message() string {
	return "PCF Dev is in an invalid state. Please run 'cf dev destroy'"
}

func (i *Invalid) err() error {
	return errors.New(i.Err.Error() + ".\n" + i.message())
}

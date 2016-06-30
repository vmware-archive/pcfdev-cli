package vm

import "errors"

type Invalid struct {
	UI UI
}

func (i *Invalid) Stop() error {
	i.UI.Failed(i.message())
	return nil
}

func (i *Invalid) VerifyStartOpts(opts *StartOpts) error {
	return errors.New(i.err())
}

func (i *Invalid) Start(opts *StartOpts) error {
	i.UI.Failed(i.message())
	return nil
}

func (i *Invalid) Status() string {
	return i.message()
}

func (i *Invalid) Suspend() error {
	i.UI.Failed(i.message())
	return nil
}

func (i *Invalid) Resume() error {
	i.UI.Failed(i.message())
	return nil
}

func (i *Invalid) message() string {
	return i.err() + "."
}

func (i *Invalid) err() string {
	return "PCF Dev is in an invalid state. Please run 'cf dev destroy'"
}

package cmd

import (
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/vm"
)

type ConcreteAutoTrustCmd struct {
	VMBuilder VMBuilder
	VBox      VBox
	Config    *config.Config
}

func (t *ConcreteAutoTrustCmd) Run() error {
	currentVM, err := t.getVM()
	if err != nil {
		return err
	}
	return currentVM.Trust(&vm.StartOpts{})
}

func (t *ConcreteAutoTrustCmd) getVM() (vm vm.VM, err error) {
	name, err := t.VBox.GetVMName()
	if err != nil {
		return nil, err
	}
	if name == "" {
		name = t.Config.DefaultVMName
	}
	if name != t.Config.DefaultVMName && name != "pcfdev-custom" {
		return nil, &OldVMError{}
	}

	return t.VMBuilder.VM(name)
}

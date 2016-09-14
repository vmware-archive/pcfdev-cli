package cmd

import (
	"github.com/cloudfoundry/cli/cf/flags"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/vm"
)

const TRUST_ARGS = 0

type TrustCmd struct {
	VMBuilder VMBuilder
	VBox      VBox
	Config    *config.Config
}

func (t *TrustCmd) Parse(args []string) error {
	return parse(flags.New(), args, TRUST_ARGS)
}

func (t *TrustCmd) Run() error {
	vm, err := t.getVM()
	if err != nil {
		return err
	}
	return vm.Trust()
}

func (t *TrustCmd) getVM() (vm vm.VM, err error) {
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

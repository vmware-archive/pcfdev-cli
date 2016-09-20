package cmd

import (
	"github.com/cloudfoundry/cli/cf/flags"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/vm"
)

const TARGET_ARGS = 0

type TargetCmd struct {
	VMBuilder  VMBuilder
	VBox       VBox
	Config     *config.Config
	AutoTarget bool
}

func (t *TargetCmd) Parse(args []string) error {
	return parse(flags.New(), args, TARGET_ARGS)
}

func (t *TargetCmd) Run() error {
	vm, err := t.getVM()
	if err != nil {
		return err
	}
	return vm.Target(t.AutoTarget)
}

func (t *TargetCmd) getVM() (vm vm.VM, err error) {
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

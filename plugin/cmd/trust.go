package cmd

import (
	"github.com/cloudfoundry/cli/cf/flags"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/vm"
)

const TRUST_ARGS = 0

type TrustCmd struct {
	Opts        *vm.StartOpts
	VMBuilder   VMBuilder
	VBox        VBox
	Config      *config.Config
	flagContext flags.FlagContext
}

func (t *TrustCmd) Parse(args []string) error {
	t.flagContext = flags.New()
	t.flagContext.NewBoolFlag("p", "", "<trust>")
	if err := parse(t.flagContext, args, TRUST_ARGS); err != nil {
		return err
	}

	t.Opts = &vm.StartOpts{
		PrintCA: t.flagContext.Bool("p"),
	}

	return nil
}

func (t *TrustCmd) Run() error {
	vm, err := t.getVM()
	if err != nil {
		return err
	}
	return vm.Trust(t.Opts)
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

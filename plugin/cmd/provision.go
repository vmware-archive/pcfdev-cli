package cmd

import (
	"github.com/cloudfoundry/cli/flags"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/vm"
)

const PROVISION_ARGS = 0

type ProvisionCmd struct {
	VBox      VBox
	VMBuilder VMBuilder
	Config    *config.Config
}

func (p *ProvisionCmd) Parse(args []string) error {
	return parse(flags.New(), args, PROVISION_ARGS)
}

func (p *ProvisionCmd) Run() error {
	vm, err := p.getVM()
	if err != nil {
		return err
	}

	return vm.Provision()
}

func (p *ProvisionCmd) getVM() (vm vm.VM, err error) {
	name, err := p.VBox.GetVMName()
	if err != nil {
		return nil, err
	}
	if name == "" {
		name = p.Config.DefaultVMName
	}
	if name != p.Config.DefaultVMName && name != "pcfdev-custom" {
		return nil, &OldVMError{}
	}

	return p.VMBuilder.VM(name)
}

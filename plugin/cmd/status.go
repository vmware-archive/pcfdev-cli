package cmd

import (
	"github.com/cloudfoundry/cli/cf/flags"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/vm"
)

const STATUS_ARGS = 0

type StatusCmd struct {
	VBox      VBox
	VMBuilder VMBuilder
	Config    *config.Config
	UI        UI
}

func (s *StatusCmd) Parse(args []string) error {
	return parse(flags.New(), args, STATUS_ARGS)
}

func (s *StatusCmd) Run() error {
	vm, err := s.getVM()
	if err != nil {
		return err
	}
	s.UI.Say(vm.Status())
	return nil
}

func (s *StatusCmd) getVM() (vm vm.VM, err error) {
	name, err := s.VBox.GetVMName()
	if err != nil {
		return nil, err
	}
	if name == "" {
		name = s.Config.DefaultVMName
	}
	if name != s.Config.DefaultVMName && name != "pcfdev-custom" {
		return nil, &OldVMError{}
	}

	return s.VMBuilder.VM(name)
}

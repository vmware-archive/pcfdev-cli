package cmd

import (
	"github.com/cloudfoundry/cli/flags"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/vm"
)

const STOP_ARGS = 0

type StopCmd struct {
	VBox      VBox
	VMBuilder VMBuilder
	Config    *config.Config
}

func (s *StopCmd) Parse(args []string) error {
	return parse(flags.New(), args, STOP_ARGS)
}

func (s *StopCmd) Run() error {
	vm, err := s.getVM()
	if err != nil {
		return err
	}
	return vm.Stop()
}

func (s *StopCmd) getVM() (vm vm.VM, err error) {
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

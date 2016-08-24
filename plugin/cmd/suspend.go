package cmd

import (
	"github.com/cloudfoundry/cli/cf/flags"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/vm"
)

const SUSPEND_ARGS = 0

type SuspendCmd struct {
	VBox      VBox
	VMBuilder VMBuilder
	Config    *config.Config
}

func (s *SuspendCmd) Parse(args []string) error {
	return parse(flags.New(), args, SUSPEND_ARGS)
}

func (s *SuspendCmd) Run() error {
	vm, err := s.getVM()
	if err != nil {
		return err
	}
	return vm.Suspend()
}

func (s *SuspendCmd) getVM() (vm vm.VM, err error) {
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

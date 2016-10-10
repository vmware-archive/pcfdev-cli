package cmd

import (
	"github.com/cloudfoundry/cli/cf/flags"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/vm"
)

const SSH_ARGS = 0

type SSHCmd struct {
	VMBuilder VMBuilder
	VBox      VBox
	Config    *config.Config
}

func (s *SSHCmd) Parse(args []string) error {
	return parse(flags.New(), args, SSH_ARGS)
}

func (s *SSHCmd) Run() error {
	vm, err := s.getVM()
	if err != nil {
		return err
	}
	return vm.SSH()
}

func (s *SSHCmd) getVM() (vm vm.VM, err error) {
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

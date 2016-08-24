package cmd

import (
	"github.com/cloudfoundry/cli/cf/flags"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/vm"
)

const RESUME_ARGS = 0

type ResumeCmd struct {
	VBox      VBox
	VMBuilder VMBuilder
	Config    *config.Config
}

func (r *ResumeCmd) Parse(args []string) error {
	return parse(flags.New(), args, RESUME_ARGS)
}

func (r *ResumeCmd) Run() error {
	vm, err := r.getVM()
	if err != nil {
		return err
	}
	return vm.Resume()
}

func (r *ResumeCmd) getVM() (vm vm.VM, err error) {
	name, err := r.VBox.GetVMName()
	if err != nil {
		return nil, err
	}
	if name == "" {
		name = r.Config.DefaultVMName
	}
	if name != r.Config.DefaultVMName && name != "pcfdev-custom" {
		return nil, &OldVMError{}
	}

	return r.VMBuilder.VM(name)
}

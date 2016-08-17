package cmd

import (
	"io"
	"time"

	"github.com/cloudfoundry/cli/flags"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/vm"
)

//go:generate mockgen -package mocks -destination mocks/ssh.go github.com/pivotal-cf/pcfdev-cli/vm SSH
type SSH interface {
	GenerateAddress() (host string, port string, err error)
	WaitForSSH(ip string, port string, timeout time.Duration) error
	RunSSHCommand(command string, ip string, port string, timeout time.Duration, stdout io.Writer, stderr io.Writer) error
	GetSSHOutput(command string, ip string, port string, timeout time.Duration) (combinedOutput string, err error)
}

const DEBUG_ARGS = 0

type DebugCmd struct {
	VBox      VBox
	VMBuilder VMBuilder
	Config    *config.Config
}

func (d *DebugCmd) Parse(args []string) error {
	return parse(flags.New(), args, DEBUG_ARGS)
}

func (d *DebugCmd) Run() error {
	vm, err := d.getVM()
	if err != nil {
		return err
	}
	return vm.GetDebugLogs()
}

func (d *DebugCmd) getVM() (vm vm.VM, err error) {
	name, err := d.VBox.GetVMName()
	if err != nil {
		return nil, err
	}
	if name == "" {
		name = d.Config.DefaultVMName
	}
	if name != d.Config.DefaultVMName && name != "pcfdev-custom" {
		return nil, &OldVMError{}
	}

	return d.VMBuilder.VM(name)
}

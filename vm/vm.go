package vm

import (
	"io"
	"time"

	"github.com/pivotal-cf/pcfdev-cli/vbox"
)

//go:generate mockgen -package mocks -destination mocks/vbox.go github.com/pivotal-cf/pcfdev-cli/vm VBox
type VBox interface {
	StartVM(name string) (vm *vbox.VM, err error)
	StopVM(name string) error
	ImportVM(name string) error
	ConflictingVMPresent(name string) (conflict bool, err error)
}

//go:generate mockgen -package mocks -destination mocks/ui.go github.com/pivotal-cf/pcfdev-cli/vm UI
type UI interface {
	Failed(message string, args ...interface{})
	Say(message string, args ...interface{})
	Confirm(message string, args ...interface{}) bool
	Ask(prompt string, args ...interface{}) (answer string)
}

//go:generate mockgen -package mocks -destination mocks/ssh.go github.com/pivotal-cf/pcfdev-cli/vm SSH
type SSH interface {
	RunSSHCommand(command string, port string, timeout time.Duration, stdout io.Writer, stderr io.Writer) error
}

//go:generate mockgen -package mocks -destination mocks/vm.go github.com/pivotal-cf/pcfdev-cli/vm VM
type VM interface {
	Start() error
	Stop() error
	Status()
}

//go:generate mockgen -package mocks -destination mocks/builder.go github.com/pivotal-cf/pcfdev-cli/vm Builder
type Builder interface {
	VM(name string) (vm VM, err error)
}

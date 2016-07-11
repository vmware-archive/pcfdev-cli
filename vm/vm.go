package vm

import (
	"io"
	"time"

	"github.com/pivotal-cf/pcfdev-cli/config"
)

//go:generate mockgen -package mocks -destination mocks/vbox.go github.com/pivotal-cf/pcfdev-cli/vm VBox
type VBox interface {
	StartVM(vmConfig *config.VMConfig) error
	StopVM(vmConfig *config.VMConfig) error
	ResumeVM(vmConfig *config.VMConfig) error
	SuspendVM(vmConfig *config.VMConfig) error
	PowerOffVM(vmConfig *config.VMConfig) error
	ImportVM(vmConfig *config.VMConfig) error
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
	GenerateAddress() (host string, port string, err error)
	WaitForSSH(ip string, port string, timeout time.Duration) error
	RunSSHCommand(command string, port string, timeout time.Duration, stdout io.Writer, stderr io.Writer) error
	GetSSHOutput(command string, ip string, port string, timeout time.Duration) (combinedOutput string, err error)
}

//go:generate mockgen -package mocks -destination mocks/vm.go github.com/pivotal-cf/pcfdev-cli/vm VM
type VM interface {
	Start(*StartOpts) error
	Provision() error
	Stop() error
	Status() string
	Suspend() error
	Resume() error

	VerifyStartOpts(*StartOpts) error
}

//go:generate mockgen -package mocks -destination mocks/builder.go github.com/pivotal-cf/pcfdev-cli/vm Builder
type Builder interface {
	VM(name string) (vm VM, err error)
}

//go:generate mockgen -package mocks -destination mocks/fs.go github.com/pivotal-cf/pcfdev-cli/vm FS
type FS interface {
	Remove(path string) error
	Exists(path string) (exists bool, err error)
	Write(path string, contents io.Reader) error
	Read(path string) (contents []byte, err error)
}

type StartOpts struct {
	Memory      uint64
	CPUs        int
	OVAPath     string
	Services    string
	NoProvision bool
}

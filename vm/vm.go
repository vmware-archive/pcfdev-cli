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
	ResumeSavedVM(vmConfig *config.VMConfig) error
	ResumePausedVM(vmConfig *config.VMConfig) error
	SuspendVM(vmConfig *config.VMConfig) error
	PowerOffVM(vmConfig *config.VMConfig) error
	ImportVM(vmConfig *config.VMConfig) error
	VMStatus(vmName string) (state string, err error)
	VMConfig(vmName string) (vmConfig *config.VMConfig, err error)
}

//go:generate mockgen -package mocks -destination mocks/ui.go github.com/pivotal-cf/pcfdev-cli/vm UI
type UI interface {
	Failed(message string, args ...interface{})
	Say(message string, args ...interface{})
	Confirm(message string) bool
	Ask(prompt string) (answer string)
}

//go:generate mockgen -package mocks -destination mocks/ssh.go github.com/pivotal-cf/pcfdev-cli/vm SSH
type SSH interface {
	GenerateAddress() (host string, port string, err error)
	WaitForSSH(ip string, port string, timeout time.Duration) error
	RunSSHCommand(command string, ip string, port string, timeout time.Duration, stdout io.Writer, stderr io.Writer) error
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
	GetDebugLogs() error
	Trust() error

	VerifyStartOpts(*StartOpts) error
}

//go:generate mockgen -package mocks -destination mocks/builder.go github.com/pivotal-cf/pcfdev-cli/vm Builder
type Builder interface {
	VM(name string) (vm VM, err error)
}

//go:generate mockgen -package mocks -destination mocks/cert_store.go github.com/pivotal-cf/pcfdev-cli/vm CertStore
type CertStore interface {
	Store(cert string) error
}

//go:generate mockgen -package mocks -destination mocks/fs.go github.com/pivotal-cf/pcfdev-cli/vm FS
type FS interface {
	Remove(path string) error
	Exists(path string) (exists bool, err error)
	Write(path string, contents io.Reader) error
	Read(path string) (contents []byte, err error)
	Compress(name string, path string, contentPaths []string) error
	TempDir() (tempDir string, err error)
}

//go:generate mockgen -package mocks -destination mocks/log_fetcher.go github.com/pivotal-cf/pcfdev-cli/vm LogFetcher
type LogFetcher interface {
	FetchLogs() error
}

//go:generate mockgen -package mocks -destination mocks/driver.go github.com/pivotal-cf/pcfdev-cli/vm Driver
type Driver interface {
	VBoxManage(arg ...string) (output []byte, err error)
}

type StartOpts struct {
	Memory      uint64
	CPUs        int
	OVAPath     string
	Services    string
	NoProvision bool
	Registries  string
}

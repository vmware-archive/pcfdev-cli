package vm

import (
	"io"
	"time"

	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/ssh"
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
	StartSSHSession(addresses []ssh.SSHAddress, privateKey []byte, timeout time.Duration, stdin io.Reader, stdout io.Writer, stderr io.Writer) error
	WaitForSSH(addresses []ssh.SSHAddress, privateKey []byte, timeout time.Duration) error
	RunSSHCommand(command string, addresses []ssh.SSHAddress, privateKey []byte, timeout time.Duration, stdout io.Writer, stderr io.Writer) error
	GetSSHOutput(command string, addresses []ssh.SSHAddress, privateKey []byte, timeout time.Duration) (combinedOutput string, err error)
}

//go:generate mockgen -package mocks -destination mocks/vm.go github.com/pivotal-cf/pcfdev-cli/vm VM
type VM interface {
	Start(*StartOpts) error
	Provision(*StartOpts) error
	Stop() error
	Status() string
	Suspend() error
	Resume() error
	GetDebugLogs() error
	Trust(*StartOpts) error
	Target(autoTarget bool) error
	SSH() error

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

//go:generate mockgen -package mocks -destination mocks/client.go github.com/pivotal-cf/pcfdev-cli/vm Client
type Client interface {
	Status(host string, privateKey []byte) (string, error)
	ReplaceSecrets(host, password string, privateKey []byte) error
}

//go:generate mockgen -package mocks -destination mocks/fs.go github.com/pivotal-cf/pcfdev-cli/vm FS
type FS interface {
	Remove(path string) error
	Exists(path string) (exists bool, err error)
	Write(path string, contents io.Reader, append bool) error
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

//go:generate mockgen -package mocks -destination mocks/cmd_runner.go github.com/pivotal-cf/pcfdev-cli/vm CmdRunner
type CmdRunner interface {
	Run(command string, args ...string) (output []byte, err error)
}

//go:generate mockgen -package mocks -destination mocks/help_text.go github.com/pivotal-cf/pcfdev-cli/vm HelpText
type HelpText interface {
	Print(domain string, autoTarget bool)
}

//go:generate mockgen -package mocks -destination mocks/network.go github.com/pivotal-cf/pcfdev-cli/vm Network
type Network interface {
	HasIPCollision(ip string) (bool, error)
}

type StartOpts struct {
	CPUs           int
	Memory         uint64
	NoProvision    bool
	OVAPath        string
	Registries     string
	Services       string
	Trust          bool
	PrintCA        bool
	Target         bool
	IP             string
	Domain         string
	MasterPassword string
}

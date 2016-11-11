package cmd

import (
	"errors"
	"io"

	"github.com/cloudfoundry/cli/cf/flags"
	"github.com/pivotal-cf/pcfdev-cli/cert"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/downloader"
	"github.com/pivotal-cf/pcfdev-cli/runner"
	"github.com/pivotal-cf/pcfdev-cli/vboxdriver"
	"github.com/pivotal-cf/pcfdev-cli/vm"
)

//go:generate mockgen -package mocks -destination mocks/ui.go github.com/pivotal-cf/pcfdev-cli/plugin/cmd UI
type UI interface {
	AskForPassword(string) string
	Say(message string, args ...interface{})
	Confirm(message string) bool
}

//go:generate mockgen -package mocks -destination mocks/vbox.go github.com/pivotal-cf/pcfdev-cli/plugin/cmd VBox
type VBox interface {
	GetVMName() (name string, err error)
	VMConfig(vmName string) (vmConfig *config.VMConfig, err error)
	DestroyPCFDevVMs() (err error)
	Version() (version *vboxdriver.VBoxDriverVersion, err error)
}

//go:generate mockgen -package mocks -destination mocks/fs.go github.com/pivotal-cf/pcfdev-cli/plugin/cmd FS
type FS interface {
	Write(path string, contents io.Reader, append bool) error
	Copy(source string, destination string) error
	Exists(path string) (exists bool, err error)
	MD5(path string) (md5 string, err error)
	Read(path string) (contents []byte, err error)
	Remove(path string) error
	TempDir() (string, error)
}

//go:generate mockgen -package mocks -destination mocks/vm_builder.go github.com/pivotal-cf/pcfdev-cli/plugin/cmd VMBuilder
type VMBuilder interface {
	VM(name string) (vm vm.VM, err error)
}

//go:generate mockgen -package mocks -destination mocks/downloader_factory.go github.com/pivotal-cf/pcfdev-cli/plugin/cmd DownloaderFactory
type DownloaderFactory interface {
	Create() (downloader downloader.Downloader, err error)
}

//go:generate mockgen -package mocks -destination mocks/downloader.go github.com/pivotal-cf/pcfdev-cli/downloader Downloader

//go:generate mockgen -package mocks -destination mocks/cmd.go github.com/pivotal-cf/pcfdev-cli/plugin/cmd Cmd
type Cmd interface {
	Parse([]string) error
	Run() error
}

//go:generate mockgen -package mocks -destination mocks/auto_cmd.go github.com/pivotal-cf/pcfdev-cli/plugin/cmd AutoCmd
type AutoCmd interface {
	Run() error
}

//go:generate mockgen -package mocks -destination mocks/cert_store.go github.com/pivotal-cf/pcfdev-cli/plugin/cmd CertStore
type CertStore interface {
	Unstore() error
}

func parse(flagContext flags.FlagContext, args []string, expectedLength int) error {
	if err := flagContext.Parse(args...); err != nil {
		return err
	}
	if len(flagContext.Args()) != expectedLength {
		return errors.New("wrong number of arguments")
	}
	return nil
}

type Builder struct {
	Client            Client
	Config            *config.Config
	DownloaderFactory DownloaderFactory
	EULAUI            EULAUI
	FS                FS
	UI                UI
	VBox              VBox
	VMBuilder         VMBuilder
}

func (b *Builder) Cmd(subcommand string) (Cmd, error) {
	switch subcommand {
	case "destroy":
		return &DestroyCmd{
			VBox:   b.VBox,
			UI:     b.UI,
			FS:     b.FS,
			Config: b.Config,
			UntrustCmd: &UntrustCmd{
				CertStore: &cert.CertStore{
					SystemStore: &cert.ConcreteSystemStore{
						FS:        b.FS,
						CmdRunner: &runner.CmdRunner{},
					},
				},
			},
		}, nil
	case "download":
		return &DownloadCmd{
			VBox:              b.VBox,
			UI:                b.UI,
			EULAUI:            b.EULAUI,
			Client:            b.Client,
			DownloaderFactory: b.DownloaderFactory,
			FS:                b.FS,
			Config:            b.Config,
		}, nil
	case "import":
		return &ImportCmd{
			DownloaderFactory: b.DownloaderFactory,
			UI:                b.UI,
			Config:            b.Config,
			FS:                b.FS,
		}, nil
	case "resume":
		return &ResumeCmd{
			VBox:      b.VBox,
			VMBuilder: b.VMBuilder,
			Config:    b.Config,
		}, nil
	case "start":
		return &StartCmd{
			VBox:      b.VBox,
			VMBuilder: b.VMBuilder,
			Config:    b.Config,
			DownloadCmd: &DownloadCmd{
				VBox:              b.VBox,
				UI:                b.UI,
				EULAUI:            b.EULAUI,
				Client:            b.Client,
				DownloaderFactory: b.DownloaderFactory,
				FS:                b.FS,
				Config:            b.Config,
			},
			AutoTrustCmd: &AutoTrustCmd{
				VBox:      b.VBox,
				VMBuilder: b.VMBuilder,
				Config:    b.Config,
			},
			TargetCmd: &TargetCmd{
				VBox:       b.VBox,
				VMBuilder:  b.VMBuilder,
				Config:     b.Config,
				AutoTarget: true,
			},
		}, nil
	case "status":
		return &StatusCmd{
			VBox:      b.VBox,
			VMBuilder: b.VMBuilder,
			Config:    b.Config,
			UI:        b.UI,
		}, nil
	case "stop":
		return &StopCmd{
			VBox:      b.VBox,
			VMBuilder: b.VMBuilder,
			Config:    b.Config,
		}, nil
	case "suspend":
		return &SuspendCmd{
			VBox:      b.VBox,
			VMBuilder: b.VMBuilder,
			Config:    b.Config,
		}, nil
	case "version", "--version":
		return &VersionCmd{
			UI:     b.UI,
			Config: b.Config,
		}, nil
	case "debug":
		return &DebugCmd{
			VMBuilder: b.VMBuilder,
			VBox:      b.VBox,
			Config:    b.Config,
		}, nil
	case "trust":
		return &TrustCmd{
			VBox:      b.VBox,
			VMBuilder: b.VMBuilder,
			Config:    b.Config,
		}, nil
	case "untrust":
		return &UntrustCmd{
			CertStore: &cert.CertStore{
				SystemStore: &cert.ConcreteSystemStore{
					FS:        b.FS,
					CmdRunner: &runner.CmdRunner{},
				},
			},
		}, nil
	case "target":
		return &TargetCmd{
			VBox:       b.VBox,
			VMBuilder:  b.VMBuilder,
			Config:     b.Config,
			AutoTarget: false,
		}, nil
	case "ssh":
		return &SSHCmd{
			VBox:      b.VBox,
			VMBuilder: b.VMBuilder,
			Config:    b.Config,
		}, nil
	default:
		return nil, errors.New("")
	}
}

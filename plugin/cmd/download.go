package cmd

import (
	"github.com/cloudfoundry/cli/cf/flags"
	"github.com/pivotal-cf/pcfdev-cli/config"
)

const DOWNLOAD_ARGS = 0

//go:generate mockgen -package mocks -destination mocks/client.go github.com/pivotal-cf/pcfdev-cli/plugin/cmd Client
type Client interface {
	AcceptEULA() error
	IsEULAAccepted() (bool, error)
	GetEULA() (eula string, err error)
}

//go:generate mockgen -package mocks -destination mocks/eula_ui.go github.com/pivotal-cf/pcfdev-cli/plugin/cmd EULAUI
type EULAUI interface {
	ConfirmText(string) bool
	Init() error
	Close() error
}

type DownloadCmd struct {
	VBox       VBox
	UI         UI
	EULAUI     EULAUI
	Client     Client
	Downloader Downloader
	FS         FS
	Config     *config.Config
}

func (d *DownloadCmd) Parse(args []string) error {
	return parse(flags.New(), args, DOWNLOAD_ARGS)
}

func (d *DownloadCmd) Run() error {
	existingVMName, err := d.VBox.GetVMName()
	if err != nil {
		return err
	}
	if existingVMName != "" && existingVMName != d.Config.DefaultVMName {
		return &OldVMError{}
	}

	current, err := d.Downloader.IsOVACurrent()
	if err != nil {
		return err
	}
	if current {
		d.UI.Say("Using existing image.")
		return nil
	}

	accepted, err := d.Client.IsEULAAccepted()
	if err != nil {
		return err
	}

	if !accepted {
		if err := d.confirmEULA(); err != nil {
			return err
		}
		if err := d.Client.AcceptEULA(); err != nil {
			return err
		}
	}

	d.UI.Say("Downloading VM...")

	if err := d.Downloader.Download(); err != nil {
		return err
	}

	d.UI.Say("\nVM downloaded.")
	return nil
}

func (d *DownloadCmd) confirmEULA() error {
	eula, err := d.Client.GetEULA()
	if err != nil {
		return err
	}

	if err := d.EULAUI.Init(); err != nil {
		return err
	}
	if accepted := d.EULAUI.ConfirmText(eula); !accepted {
		if err := d.EULAUI.Close(); err != nil {
			return err
		}
		return &EULARefusedError{}
	}
	return d.EULAUI.Close()
}

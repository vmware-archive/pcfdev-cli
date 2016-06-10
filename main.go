package main

import (
	"os"

	"github.com/pivotal-cf/pcfdev-cli/address"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/downloader"
	"github.com/pivotal-cf/pcfdev-cli/fs"
	"github.com/pivotal-cf/pcfdev-cli/network"
	"github.com/pivotal-cf/pcfdev-cli/ping"
	"github.com/pivotal-cf/pcfdev-cli/pivnet"
	"github.com/pivotal-cf/pcfdev-cli/plugin"
	"github.com/pivotal-cf/pcfdev-cli/ssh"
	"github.com/pivotal-cf/pcfdev-cli/system"
	"github.com/pivotal-cf/pcfdev-cli/vbox"
	"github.com/pivotal-cf/pcfdev-cli/vm"

	"github.com/cloudfoundry/cli/cf/terminal"
	cfplugin "github.com/cloudfoundry/cli/plugin"
)

var (
	releaseId     string
	productFileId string
	md5           string
	vmName        string
)

func main() {
	fileSystem := &fs.FS{}
	termUI := terminal.NewUI(os.Stdin, terminal.NewTeePrinter())
	system := &system.System{
		FS: fileSystem,
	}
	config, err := config.New(vmName, system)
	if err != nil {
		termUI.Failed("Error: %s", err)
	}
	token := &pivnet.Token{
		Config: config,
		FS:     fileSystem,
		UI:     termUI,
	}
	client := &pivnet.Client{
		Host:          "https://network.pivotal.io",
		ReleaseId:     releaseId,
		ProductFileId: productFileId,
		Token:         token,
	}

	cfplugin.Start(&plugin.Plugin{
		Client: client,
		Downloader: &downloader.Downloader{
			PivnetClient: client,
			FS:           fileSystem,
			ExpectedMD5:  md5,
			Config:       config,
			Token:        token,
		},
		UI:     &plugin.NonTranslatingUI{termUI},
		Config: config,
		SSH:    &ssh.SSH{},
		Builder: &vm.VBoxBuilder{
			Driver: &vbox.VBoxDriver{},
			Config: config,
		},
		VBox: &vbox.VBox{
			SSH:    &ssh.SSH{},
			Driver: &vbox.VBoxDriver{},
			Picker: &address.Picker{
				Pinger:  &ping.Pinger{},
				Network: &network.Network{},
			},
			Config: config,
		},
	})
}

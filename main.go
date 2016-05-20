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
	"github.com/pivotal-cf/pcfdev-cli/requirements"
	"github.com/pivotal-cf/pcfdev-cli/ssh"
	"github.com/pivotal-cf/pcfdev-cli/system"
	"github.com/pivotal-cf/pcfdev-cli/vbox"

	"github.com/cloudfoundry/cli/cf/terminal"
	cfplugin "github.com/cloudfoundry/cli/plugin"
)

var productFileDownloadURI string
var (
	releaseId     = "1622"
	productFileId = "4693"
	md5           = "7eed7a7314435a3cffc2c943e80606ad"
	vmName        = "pcfdev-0.83.0"
)

func main() {
	ui := terminal.NewUI(os.Stdin, terminal.NewTeePrinter())
	config := &config.Config{
		UI:        ui,
		MinMemory: 3072,
		MaxMemory: 4096,
	}
	client := &pivnet.Client{
		Config:        config,
		Host:          "https://network.pivotal.io",
		ReleaseId:     releaseId,
		ProductFileId: productFileId,
	}
	system := &system.System{}
	cfplugin.Start(&plugin.Plugin{
		Client: client,
		Downloader: &downloader.Downloader{
			PivnetClient: client,
			FS:           &fs.FS{},
			ExpectedMD5:  md5,
		},
		UI:  ui,
		SSH: &ssh.SSH{},
		VBox: &vbox.VBox{
			SSH:    &ssh.SSH{},
			Driver: &vbox.VBoxDriver{},
			Picker: &address.Picker{
				Pinger:  &ping.Pinger{},
				Network: &network.Network{},
			},
			Config: config,
			System: system,
		},
		RequirementsChecker: &requirements.Checker{
			System: system,
			Config: config,
		},
		VMName: vmName,
	})
}

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
	currentUser "github.com/pivotal-cf/pcfdev-cli/user"
	"github.com/pivotal-cf/pcfdev-cli/vbox"

	"github.com/cloudfoundry/cli/cf/terminal"
	cfplugin "github.com/cloudfoundry/cli/plugin"
)

var productFileDownloadURI string
var (
	releaseId     = "1622"
	productFileId = "4546"
	md5           = "424a588f1d359905632a9221efad6097"
	vmName        = "pcfdev-0.71.0"
)

func main() {
	ui := terminal.NewUI(os.Stdin, terminal.NewTeePrinter())
	cfplugin.Start(&plugin.Plugin{
		Downloader: &downloader.Downloader{
			PivnetClient: &pivnet.Client{
				Config: &config.Config{
					UI: ui,
				},
				Host:          "https://network.pivotal.io",
				ReleaseId:     releaseId,
				ProductFileId: productFileId,
			},
			FS:          &fs.FS{},
			ExpectedMD5: md5,
		},
		UI:  ui,
		SSH: &ssh.SSH{},
		VBox: &vbox.VBox{
			SSH:    &ssh.SSH{},
			Driver: &vbox.VBoxDriver{},
			Picker: &address.Picker{
				Pinger: &ping.Pinger{
					User: &currentUser.User{},
				},
				Network: &network.Network{},
			},
		},
		RequirementsChecker: &requirements.Checker{
			MemoryChecker: &requirements.Memory{
				MinimumFreeMemory: 3072,
			},
		},
		VMName: vmName,
	})
}

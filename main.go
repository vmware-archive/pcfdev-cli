package main

import (
	"os"

	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/fs"
	"github.com/pivotal-cf/pcfdev-cli/pivnet"
	"github.com/pivotal-cf/pcfdev-cli/plugin"
	"github.com/pivotal-cf/pcfdev-cli/ssh"
	"github.com/pivotal-cf/pcfdev-cli/vbox"

	"github.com/cloudfoundry/cli/cf/terminal"
	cfplugin "github.com/cloudfoundry/cli/plugin"
)

var productFileDownloadURI string
var (
	releaseId     = "1622"
	productFileId = "4466"
	md5           = "346f42ae096185b39403017f0c45ee37"
	vmName        = "pcfdev-0.68.0"
)

func main() {
	ui := terminal.NewUI(os.Stdin, terminal.NewTeePrinter())
	cfplugin.Start(&plugin.Plugin{
		UI:  ui,
		SSH: &ssh.SSH{},
		PivnetClient: &pivnet.Client{
			Host:          "https://network.pivotal.io",
			ReleaseId:     releaseId,
			ProductFileId: productFileId,
		},
		VBox: &vbox.VBox{
			SSH:    &ssh.SSH{},
			Driver: &vbox.VBoxDriver{},
		},
		FS: &fs.FS{},
		Config: &config.Config{
			UI: ui,
		},
		ExpectedMD5: md5,
		VMName:      vmName,
	})
}

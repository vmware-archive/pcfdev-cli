package main

import (
	"os"

	"github.com/pivotal-cf/pcfdev-cli/fs"
	"github.com/pivotal-cf/pcfdev-cli/pivnet"
	"github.com/pivotal-cf/pcfdev-cli/plugin"
	"github.com/pivotal-cf/pcfdev-cli/ssh"
	"github.com/pivotal-cf/pcfdev-cli/vbox"

	"github.com/cloudfoundry/cli/cf/terminal"
	cfplugin "github.com/cloudfoundry/cli/plugin"
)

func main() {
	cfplugin.Start(&plugin.Plugin{
		UI:  terminal.NewUI(os.Stdin, terminal.NewTeePrinter()),
		SSH: &ssh.SSH{},
		PivnetClient: &pivnet.Client{
			Host:  "https://network.pivotal.io",
			Token: os.Getenv("PIVNET_TOKEN"),
		},
		VBox: &vbox.VBox{
			SSH:    &ssh.SSH{},
			Driver: &vbox.VBoxDriver{},
		},
		FS: &fs.FS{},
	})
}

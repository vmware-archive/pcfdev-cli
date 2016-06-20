package main

import (
	"os"
	"os/exec"
	"strings"

	"github.com/pivotal-cf/pcfdev-cli/address"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/downloader"
	"github.com/pivotal-cf/pcfdev-cli/fs"
	"github.com/pivotal-cf/pcfdev-cli/network"
	"github.com/pivotal-cf/pcfdev-cli/pivnet"
	"github.com/pivotal-cf/pcfdev-cli/plugin"
	"github.com/pivotal-cf/pcfdev-cli/ssh"
	"github.com/pivotal-cf/pcfdev-cli/system"
	"github.com/pivotal-cf/pcfdev-cli/vbox"
	"github.com/pivotal-cf/pcfdev-cli/vm"

	"github.com/cloudfoundry/cli/cf/terminal"
	cfplugin "github.com/cloudfoundry/cli/plugin"
	"github.com/kardianos/osext"
)

var (
	releaseId     string
	productFileId string
	md5           string
	vmName        string
)

func main() {
	ui := terminal.NewUI(os.Stdin, terminal.NewTeePrinter())

	confirmInstalled(ui)

	fileSystem := &fs.FS{}
	system := &system.System{
		FS: fileSystem,
	}
	config, err := config.New(vmName, system)
	if err != nil {
		ui.Failed("Error: %s", err)
	}
	token := &pivnet.Token{
		Config: config,
		FS:     fileSystem,
		UI:     ui,
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
		UI:     &plugin.NonTranslatingUI{ui},
		Config: config,
		SSH:    &ssh.SSH{},
		FS:     fileSystem,
		Builder: &vm.VBoxBuilder{
			Driver: &vbox.VBoxDriver{},
			Config: config,
			FS:     fileSystem,
		},
		VBox: &vbox.VBox{
			SSH:    &ssh.SSH{},
			FS:     fileSystem,
			Driver: &vbox.VBoxDriver{},
			Picker: &address.Picker{
				Network: &network.Network{},
				Driver:  &vbox.VBoxDriver{},
			},
			Config: config,
		},
	})
}

func confirmInstalled(ui terminal.UI) {
	var firstArg string
	if len(os.Args) > 1 {
		firstArg = os.Args[1]
	}

	switch firstArg {
	case "":
		plugin, err := osext.Executable()
		if err != nil {
			ui.Say("Failed to determine plugin path: %s", err)
			os.Exit(1)
		}

		operation := "upgraded"
		if err := exec.Command("cf", "uninstall-plugin", "pcfdev").Run(); err != nil {
			operation = "installed"
		}

		if output, err := exec.Command("cf", "install-plugin", plugin, "-f").CombinedOutput(); err != nil {
			ui.Say(strings.TrimSpace(string(output)))
			os.Exit(1)
		}

		ui.Say("Plugin successfully %s, run: cf dev help", operation)
		os.Exit(0)
	case "help", "-h", "--help":
		ui.Say("Usage: %s", os.Args[0])
		ui.Say("Running this binary directly will automatically install the PCF Dev cf CLI plugin.")
		ui.Say("You must have the latest version of the cf CLI and Virtualbox 5.0+ to use PCF Dev.")
		ui.Say("After installing, run: cf dev help")
		os.Exit(0)
	}
}

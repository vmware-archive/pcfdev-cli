package main

import (
	"os"
	"os/exec"
	"strconv"
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
	buildVersion    string
	buildSHA        string
	ovaBuildVersion string
	releaseId       string
	productFileId   string
	md5             string
	vmName          string
)

func main() {
	ui := terminal.NewUI(os.Stdin, terminal.NewTeePrinter())

	confirmInstalled(ui)

	fileSystem := &fs.FS{}
	driver := &vbox.VBoxDriver{FS: fileSystem}
	system := &system.System{
		FS: fileSystem,
	}
	config, err := config.New(vmName, md5, system)
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
			Config:       config,
			Token:        token,
		},
		UI:     &plugin.NonTranslatingUI{ui},
		Config: config,
		FS:     fileSystem,
		Builder: &vm.VBoxBuilder{
			Config: config,
			Driver: driver,
			FS:     fileSystem,
			SSH:    &ssh.SSH{},
		},
		VBox: &vbox.VBox{
			SSH:    &ssh.SSH{},
			FS:     fileSystem,
			Driver: driver,
			Picker: &address.Picker{
				Network: &network.Network{},
				Driver:  driver,
			},
			Config: config,
		},
		Version: &plugin.Version{
			BuildVersion:    buildVersion,
			BuildSHA:        buildSHA,
			OVABuildVersion: ovaBuildVersion,
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

		installOpts := []string{"install-plugin", plugin}
		if needsConfirm := checkCLIVersion(ui); needsConfirm {
			installOpts = append(installOpts, "-f")
		}
		if output, err := exec.Command("cf", installOpts...).CombinedOutput(); err != nil {
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

func checkCLIVersion(ui terminal.UI) (installNeedsConfirm bool) {
	cfVersion, err := exec.Command("cf", "--version").Output()
	versionParts := strings.SplitN(strings.TrimPrefix(string(cfVersion), "cf version "), ".", 3)
	if err != nil || len(versionParts) < 3 {
		ui.Say("Failed to determine cf CLI version.")
		os.Exit(1)
	}
	majorVersion, errMajor := strconv.Atoi(versionParts[0])
	minorVersion, errMinor := strconv.Atoi(versionParts[1])
	if errMajor != nil || errMinor != nil || majorVersion < 6 || (majorVersion == 6 && minorVersion < 7) {
		ui.Say("Your cf CLI version is too old. Please install the latest cf CLI.")
		os.Exit(1)
	}
	if majorVersion == 6 && minorVersion < 13 {
		return false
	}
	return true
}

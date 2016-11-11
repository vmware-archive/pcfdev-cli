package main

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/cloudfoundry/cli/cf/trace"
	"github.com/pivotal-cf/pcfdev-cli/address"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/downloader"
	"github.com/pivotal-cf/pcfdev-cli/exit"
	"github.com/pivotal-cf/pcfdev-cli/fs"
	"github.com/pivotal-cf/pcfdev-cli/network"
	"github.com/pivotal-cf/pcfdev-cli/pivnet"
	"github.com/pivotal-cf/pcfdev-cli/plugin"
	"github.com/pivotal-cf/pcfdev-cli/plugin/cmd"
	"github.com/pivotal-cf/pcfdev-cli/runner"
	"github.com/pivotal-cf/pcfdev-cli/ssh"
	"github.com/pivotal-cf/pcfdev-cli/system"
	"github.com/pivotal-cf/pcfdev-cli/ui"
	"github.com/pivotal-cf/pcfdev-cli/vbox"
	"github.com/pivotal-cf/pcfdev-cli/vm"

	"github.com/cloudfoundry/cli/cf/terminal"
	cfplugin "github.com/cloudfoundry/cli/plugin"
	"github.com/kardianos/osext"
	"github.com/pivotal-cf/pcfdev-cli/vboxdriver"
	vmClient "github.com/pivotal-cf/pcfdev-cli/vm/client"
	"net/http"
)

var (
	buildVersion       string
	buildSHA           string
	ovaBuildVersion    string
	releaseId          string
	productFileId      string
	md5                string
	vmName             string
	insecurePrivateKey string
)

func main() {
	cfui := terminal.NewUI(
		os.Stdin,
		os.Stdout,
		terminal.NewTeePrinter(os.Stdout),
		trace.NewLogger(os.Stdout, false, "", ""),
	)

	confirmInstalled(cfui)

	fileSystem := &fs.FS{}
	driver := &vboxdriver.VBoxDriver{
		FS:        fileSystem,
		CmdRunner: &runner.CmdRunner{},
	}
	conf, err := config.New(
		vmName,
		md5,
		[]byte(insecurePrivateKey),
		&system.System{
			FS: fileSystem,
		},
		&config.Version{
			BuildVersion:    buildVersion,
			BuildSHA:        buildSHA,
			OVABuildVersion: ovaBuildVersion,
		})
	if err != nil {
		cfui.Failed("Error: %s", err)
		os.Exit(1)
	}
	token := &pivnet.Token{
		Config: conf,
		FS:     fileSystem,
		UI:     cfui,
	}
	client := &pivnet.Client{
		Host:          "https://network.pivotal.io",
		ReleaseId:     releaseId,
		ProductFileId: productFileId,
		Token:         token,
	}
	token.Client = client
	sshClient := &ssh.SSH{
		Terminal: &ssh.TerminalWrapper{},
		WindowResizer: &ssh.ConcreteWindowResizer{
			DoneChannel: make(chan bool),
		},
	}
	vbx := &vbox.VBox{
		SSH:    sshClient,
		FS:     fileSystem,
		Driver: driver,
		Picker: &address.Picker{
			Network: &network.Network{},
			Driver:  driver,
		},
		Config: conf,
	}
	httpClientIgnoringEnvironmentProxies := &http.Client{
		Transport: &http.Transport{
			Proxy: nil,
		},
	}
	cfplugin.Start(&plugin.Plugin{
		UI:     &plugin.NonTranslatingUI{cfui},
		Config: conf,
		Exit:   &exit.Exit{},
		CmdBuilder: &cmd.Builder{
			Client: client,
			Config: conf,
			DownloaderFactory: &downloader.DownloaderFactory{
				PivnetClient:         client,
				FS:                   fileSystem,
				Token:                token,
				Config:               conf,
				DownloadAttempts:     10,
				DownloadAttemptDelay: time.Second,
			},
			EULAUI: &ui.UI{},
			FS:     fileSystem,
			UI:     cfui,
			VBox:   vbx,
			VMBuilder: &vm.VBoxBuilder{
				VBox:   vbx,
				Config: conf,
				FS:     fileSystem,
				SSH:    sshClient,
				Client: &vmClient.Client{
					Timeout:    time.Second * 20,
					HttpClient: httpClientIgnoringEnvironmentProxies,
					SSHClient:  sshClient,
				},
			},
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

		ui.Say("Plugin successfully %s. Current version: %s. For more info run: cf dev help", operation, buildVersion)
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

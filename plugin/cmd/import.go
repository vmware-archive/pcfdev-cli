package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/cloudfoundry/cli/flags"
	"github.com/pivotal-cf/pcfdev-cli/config"
)

const IMPORT_ARGS=1

type ImportCmd struct {
	OVAPath    string
	Downloader Downloader
	UI         UI
	Config     *config.Config
	FS         FS
}

func (i *ImportCmd) Parse(args []string) error {
	if err := parse(flags.New(), args, IMPORT_ARGS); err != nil {
		return err
	}
	i.OVAPath = args[0]
	return nil
}

func (i *ImportCmd) Run() error {
	md5, err := i.FS.MD5(i.OVAPath)
	if err != nil {
		return err
	}
	if md5 != i.Config.ExpectedMD5 {
		return fmt.Errorf("specified OVA version does not match the expected OVA version (%s) for this version of the cf CLI plugin", i.Config.Version.OVABuildVersion)
	}
	ovaIsCurrent, err := i.Downloader.IsOVACurrent()
	if err != nil {
		return err
	}
	if ovaIsCurrent {
		i.UI.Say("PCF Dev OVA is already installed.")
		return nil
	}
	if err := i.FS.Copy(i.OVAPath, filepath.Join(i.Config.OVADir, i.Config.DefaultVMName+".ova")); err != nil {
		return err
	}
	i.UI.Say(fmt.Sprintf("OVA version %s imported successfully.", i.Config.Version.OVABuildVersion))
	return nil
}

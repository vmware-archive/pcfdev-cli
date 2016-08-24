package cmd

import (
	"fmt"

	"github.com/cloudfoundry/cli/cf/flags"
	"github.com/pivotal-cf/pcfdev-cli/config"
)

const VERSION_ARGS = 0

type VersionCmd struct {
	Config *config.Config
	UI     UI
}

func (v *VersionCmd) Parse(args []string) error {
	return parse(flags.New(), args, VERSION_ARGS)
}

func (v *VersionCmd) Run() error {
	v.UI.Say(fmt.Sprintf("PCF Dev version %s (CLI: %s, OVA: %s)",
		v.Config.Version.BuildVersion,
		v.Config.Version.BuildSHA,
		v.Config.Version.OVABuildVersion))
	return nil
}

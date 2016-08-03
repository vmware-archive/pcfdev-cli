package cmd

import (
	"fmt"

	"github.com/cloudfoundry/cli/flags"
	"github.com/pivotal-cf/pcfdev-cli/config"
)

const DESTROY_ARGS = 0

type DestroyCmd struct {
	VBox   VBox
	UI     UI
	FS     FS
	Config *config.Config
}

func (d *DestroyCmd) Parse(args []string) error {
	return parse(flags.New(), args, DESTROY_ARGS)
}

func (d *DestroyCmd) Run() error {
	var vmErr error
	if err := d.VBox.DestroyPCFDevVMs(); err != nil {
		vmErr = fmt.Errorf("error destroying PCF Dev VM: %s", err)
	} else {
		d.UI.Say("PCF Dev VM has been destroyed.")
	}

	if err := d.FS.Remove(d.Config.VMDir); err != nil {
		if vmErr != nil {
			return fmt.Errorf("%s\nerror removing %s: %s", vmErr, d.Config.VMDir, err)
		}
		return fmt.Errorf("error removing %s: %s", d.Config.VMDir, err)
	}
	return vmErr
}

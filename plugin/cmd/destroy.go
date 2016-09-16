package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/cloudfoundry/cli/cf/flags"
	"github.com/pivotal-cf/pcfdev-cli/config"
)

const DESTROY_ARGS = 0

type DestroyCmd struct {
	VBox       VBox
	UI         UI
	FS         FS
	UntrustCmd Cmd
	Config     *config.Config
}

func (d *DestroyCmd) Parse(args []string) error {
	return parse(flags.New(), args, DESTROY_ARGS)
}

func (d *DestroyCmd) Run() error {
	var errs []string

	if err := d.UntrustCmd.Run(); err != nil {
		errs = append(errs, fmt.Sprintf("error removing certificates from trust store: %s", err))
	}

	if err := d.VBox.DestroyPCFDevVMs(); err != nil {
		errs = append(errs, fmt.Sprintf("error destroying PCF Dev VM: %s", err))
	} else {
		d.UI.Say("PCF Dev VM has been destroyed.")
	}

	if err := d.FS.Remove(d.Config.VMDir); err != nil {
		errs = append(errs, fmt.Sprintf("error removing %s: %s", d.Config.VMDir, err))
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}

	return nil
}

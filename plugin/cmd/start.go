package cmd

import (
	"errors"

	"github.com/cloudfoundry/cli/flags"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/vm"
)

const START_ARGS = 0

type StartCmd struct {
	Opts        *vm.StartOpts
	VBox        VBox
	VMBuilder   VMBuilder
	Config      *config.Config
	DownloadCmd Cmd
}

func (s *StartCmd) Parse(args []string) error {
	flagContext := flags.New()
	flagContext.NewIntFlag("m", "memory", "<memory in MB>")
	flagContext.NewIntFlag("c", "cpus", "<number of cpus>")
	flagContext.NewStringFlag("o", "ova", "<path to custom ova>")
	flagContext.NewStringFlag("s", "services", "<services to start with>")
	flagContext.NewBoolFlag("n", "", "<skip provisioning>")
	if err := parse(flagContext, args, START_ARGS); err != nil {
		return err
	}

	s.Opts = &vm.StartOpts{
		Memory:      uint64(flagContext.Int("m")),
		CPUs:        flagContext.Int("c"),
		OVAPath:     flagContext.String("o"),
		Services:    flagContext.String("s"),
		NoProvision: flagContext.Bool("n"),
	}
	return nil
}

func (s *StartCmd) Run() error {
	var name string

	if s.Opts.OVAPath != "" {
		name = "pcfdev-custom"
	} else {
		name = s.Config.DefaultVMName
	}

	existingVMName, err := s.VBox.GetVMName()
	if err != nil {
		return err
	}
	if existingVMName != "" {
		if s.Opts.OVAPath != "" {
			if existingVMName != "pcfdev-custom" {
				return errors.New("you must destroy your existing VM to use a custom OVA")
			}
		} else {
			if existingVMName != s.Config.DefaultVMName && existingVMName != "pcfdev-custom" {
				return &OldVMError{}
			}
		}
	}

	if existingVMName == "pcfdev-custom" {
		name = "pcfdev-custom"
	}

	v, err := s.VMBuilder.VM(name)
	if err != nil {
		return err
	}

	if err := v.VerifyStartOpts(s.Opts); err != nil {
		return err
	}
	if s.Opts.OVAPath == "" && existingVMName != "pcfdev-custom" {
		if err := s.DownloadCmd.Run(); err != nil {
			return err
		}
	}

	return v.Start(s.Opts)
}

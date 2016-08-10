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
	flagContext.NewBoolFlag("n", "", "<skip provisioning>")
	flagContext.NewIntFlag("c", "", "<number of cpus>")
	flagContext.NewIntFlag("m", "", "<memory in MB>")
	flagContext.NewStringFlag("o", "", "<path to custom ova>")
	flagContext.NewStringFlag("p", "", "<private docker registries>")
	flagContext.NewStringFlag("s", "", "<services to start with>")
	if err := parse(flagContext, args, START_ARGS); err != nil {
		return err
	}

	s.Opts = &vm.StartOpts{
		CPUs:        flagContext.Int("c"),
		Memory:      uint64(flagContext.Int("m")),
		NoProvision: flagContext.Bool("n"),
		OVAPath:     flagContext.String("o"),
		Registries:  flagContext.String("p"),
		Services:    flagContext.String("s"),
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

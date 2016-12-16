package cmd

import (
	"errors"

	"github.com/cloudfoundry/cli/cf/flags"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/vm"
	"os"
)

const START_ARGS = 0

type StartCmd struct {
	Opts           *vm.StartOpts
	VBox           VBox
	VMBuilder      VMBuilder
	Config         *config.Config
	AutoTrustCmd   AutoCmd
	DownloadCmd    Cmd
	TargetCmd      Cmd
	UI             UI
	flagContext    flags.FlagContext
	existingVMName string
}

func (s *StartCmd) Parse(args []string) error {
	s.flagContext = flags.New()
	s.flagContext.NewBoolFlag("k", "", "<trust>")
	s.flagContext.NewBoolFlag("t", "", "<target>")
	s.flagContext.NewBoolFlag("n", "", "<skip provisioning>")
	s.flagContext.NewBoolFlag("p", "", "<provision>")
	s.flagContext.NewIntFlag("c", "", "<number of cpus>")
	s.flagContext.NewIntFlag("m", "", "<memory in MB>")
	s.flagContext.NewStringFlag("o", "", "<path to custom ova>")
	s.flagContext.NewStringFlag("r", "", "<docker registries>")
	s.flagContext.NewStringFlag("s", "", "<services to start with>")
	s.flagContext.NewStringFlag("d", "", "<domain>")
	s.flagContext.NewStringFlag("i", "", "<IP>")
	s.flagContext.NewBoolFlag("x", "", "<master password>")
	if err := parse(s.flagContext, args, START_ARGS); err != nil {
		return err
	}

	var password string
	if s.flagContext.Bool("x") {
		var err error
		password, err = s.getPCFDevPassword()
		if err != nil {
			return err
		}
	}

	s.Opts = &vm.StartOpts{
		CPUs:           s.flagContext.Int("c"),
		Memory:         uint64(s.flagContext.Int("m")),
		NoProvision:    s.flagContext.Bool("n"),
		OVAPath:        s.flagContext.String("o"),
		Registries:     s.flagContext.String("r"),
		Services:       s.flagContext.String("s"),
		Target:         s.flagContext.Bool("t"),
		Domain:         s.flagContext.String("d"),
		IP:             s.flagContext.String("i"),
		MasterPassword: password,
	}
	return nil
}

func (s *StartCmd) Run() error {
	err := s.isIncompatibleVBox()
	if err != nil {
		return err
	}

	if s.existingVMName, err = s.VBox.GetVMName(); err != nil {
		return err
	}

	if err = s.shouldNotStartVm(); err != nil {
		return err
	}

	v, err := s.VMBuilder.VM(s.vmName())
	if err != nil {
		return err
	}

	if s.flagContext.Bool("p") {
		return v.Provision(&vm.StartOpts{})
	} else {
		if err := v.VerifyStartOpts(s.Opts); err != nil {
			return err
		}
		if !s.isCustomOva() {
			if err := s.DownloadCmd.Run(); err != nil {
				return err
			}
		}

		if err := v.Start(s.Opts); err != nil {
			return err
		}

		if s.flagContext.Bool("k") {
			if err := s.AutoTrustCmd.Run(); err != nil {
				return err
			}
		}

		if s.flagContext.Bool("t") {
			return s.TargetCmd.Run()
		}

		return nil
	}
}

func (s *StartCmd) isIncompatibleVBox() error {
	version, err := s.VBox.Version()
	if err != nil {
		return err
	}

	if version.Major < 5 {
		return &OldDriverError{}
	}

	return nil
}

func (s *StartCmd) shouldNotStartVm() error {
	if s.vmIsOld() {
		return &OldVMError{}
	}

	if s.switchingToCustomOva() {
		return errors.New("you must destroy your existing VM to use a custom OVA")
	}

	return nil
}

func (s *StartCmd) vmIsOld() bool {
	return !s.isCustomOva() && s.existingVMName != s.Config.DefaultVMName && s.existingVMName != ""
}

func (s *StartCmd) switchingToCustomOva() bool {
	if s.existingVMName != "" && s.Opts.OVAPath != "" && s.existingVMName != "pcfdev-custom" {
		return true
	} else {
		return false
	}
}

func (s *StartCmd) isCustomOva() bool {
	return (s.Opts.OVAPath != "" || s.existingVMName == "pcfdev-custom")
}

func (s *StartCmd) vmName() string {
	if s.isCustomOva() {
		return "pcfdev-custom"
	} else {
		return s.Config.DefaultVMName
	}
}

func (s *StartCmd) getPCFDevPassword() (string, error) {
	if os.Getenv("PCFDEV_PASSWORD") != "" {
		return os.Getenv("PCFDEV_PASSWORD"), nil
	}

	password := s.UI.AskForPassword("Choose master password")
	passwordConfirmation := s.UI.AskForPassword("Confirm master password")

	if password == "" && passwordConfirmation == "" {
		return "", errors.New("password cannot be empty")
	}

	if password != passwordConfirmation {
		return "", errors.New("passwords do not match")
	}

	return password, nil
}

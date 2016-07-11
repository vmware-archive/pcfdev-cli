package vm

import (
	"errors"
	"fmt"
	"time"

	"github.com/pivotal-cf/pcfdev-cli/config"
)

type Suspended struct {
	Config   *config.Config
	VMConfig *config.VMConfig

	VBox VBox
	UI   UI
	SSH  SSH
}

func (s *Suspended) Stop() error {
	s.UI.Say("Your VM is currently suspended. You must resume your VM with `cf dev resume` to shut it down.")
	return nil
}

func (s *Suspended) VerifyStartOpts(opts *StartOpts) error {
	if opts.Memory != uint64(0) {
		return errors.New("memory cannot be changed once the vm has been created")
	}
	if opts.CPUs != 0 {
		return errors.New("cores cannot be changed once the vm has been created")
	}
	if opts.Services != "" {
		return errors.New("services cannot be changed once the vm has been created")
	}
	return s.checkMemory()
}

func (s *Suspended) Start(opts *StartOpts) error {
	return s.Resume()
}

func (s *Suspended) Provision() error {
	return nil
}


func (s *Suspended) Status() string {
	return "Suspended"
}

func (s *Suspended) Suspend() error {
	s.UI.Say("Your VM is suspended.")
	return nil
}

func (s *Suspended) Resume() error {
	if err := s.checkMemory(); err != nil {
		return err
	}

	s.UI.Say("Resuming VM...")
	if err := s.VBox.ResumeVM(s.VMConfig); err != nil {
		return &ResumeVMError{err}
	}

	if err := s.SSH.WaitForSSH(s.VMConfig.IP, "22", 5*time.Minute); err != nil {
		return &ResumeVMError{err}
	}

	s.UI.Say("PCF Dev is now running.")

	return nil
}

func (s *Suspended) checkMemory() error {
	if s.VMConfig.Memory > s.Config.FreeMemory {
		if !s.UI.Confirm(fmt.Sprintf("Less than %d MB of free memory detected, continue (y/N): ", s.VMConfig.Memory)) {
			return errors.New("user declined to continue, exiting")
		}
	}
	return nil
}

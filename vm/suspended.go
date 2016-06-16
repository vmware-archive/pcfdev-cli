package vm

import (
	"errors"
	"fmt"

	"github.com/pivotal-cf/pcfdev-cli/config"
)

type Suspended struct {
	Name    string
	Domain  string
	IP      string
	Memory  uint64
	SSHPort string
	Config  *config.Config

	VBox VBox
	UI   UI
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
	return s.checkMemory()
}

func (s *Suspended) Start(opts *StartOpts) error {
	return s.Resume()
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
	if err := s.VBox.ResumeVM(s.Name); err != nil {
		return &ResumeVMError{err}
	}

	return nil
}

func (s *Suspended) checkMemory() error {
	if s.Memory > s.Config.FreeMemory {
		if !s.UI.Confirm(fmt.Sprintf("Less than %d MB of free memory detected, continue (y/N): ", s.Memory)) {
			return errors.New("user declined to continue, exiting")
		}
	}
	return nil
}

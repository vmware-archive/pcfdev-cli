package vm

import (
	"errors"
	"fmt"

	"github.com/pivotal-cf/pcfdev-cli/config"
)

type Saved struct {
	Config      *config.Config
	SuspendedVM *Suspended
	UI          UI
}

func (s *Saved) VerifyStartOpts(opts *StartOpts) error {
	if err := s.SuspendedVM.VerifyStartOpts(opts); err != nil {
		return err
	}
	if err := s.checkMemory(); err != nil {
		return err
	}
	return nil
}

func (s *Saved) Start(opts *StartOpts) error {
	return s.SuspendedVM.Start(opts)
}

func (s *Saved) Provision() error {
	return s.SuspendedVM.Provision()
}

func (s *Saved) Stop() error {
	return s.SuspendedVM.Stop()
}

func (s *Saved) Status() string {
	return "Suspended"
}

func (s *Saved) Suspend() error {
	s.UI.Say("Your VM is suspended.")
	return nil
}

func (s *Saved) Resume() error {
	if err := s.checkMemory(); err != nil {
		return err
	}
	return s.SuspendedVM.Resume()
}

func (s *Saved) checkMemory() error {
	if s.SuspendedVM.VMConfig.Memory > s.Config.FreeMemory {
		if !s.UI.Confirm(fmt.Sprintf("Less than %d MB of free memory detected, continue (y/N): ", s.SuspendedVM.VMConfig.Memory)) {
			return errors.New("user declined to continue, exiting")
		}
	}
	return nil
}

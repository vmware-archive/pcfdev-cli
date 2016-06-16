package vm

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/pivotal-cf/pcfdev-cli/config"
)

type Stopped struct {
	Config   *config.Config
	VMConfig *config.VMConfig

	VBox VBox
	SSH  SSH
	UI   UI
}

func (s *Stopped) Stop() error {
	s.UI.Say("PCF Dev is stopped")
	return nil
}

func (s *Stopped) VerifyStartOpts(opts *StartOpts) error {
	if opts.Memory != uint64(0) {
		return errors.New("memory cannot be changed once the vm has been created")
	}
	if opts.CPUs != 0 {
		return errors.New("cores cannot be changed once the vm has been created")
	}
	if s.VMConfig.Memory > s.Config.FreeMemory {
		if !s.UI.Confirm(fmt.Sprintf("Less than %d MB of free memory detected, continue (y/N): ", s.VMConfig.Memory)) {
			return errors.New("user declined to continue, exiting")
		}
	}
	return nil
}

func (s *Stopped) Start(opts *StartOpts) error {
	s.UI.Say("Starting VM...")
	if err := s.VBox.StartVM(s.VMConfig); err != nil {
		return &StartVMError{err}
	}

	s.UI.Say("Provisioning VM...")
	provisionCommand := fmt.Sprintf("sudo /var/pcfdev/run %s %s '$2a$04$EpJtIJ8w6hfCwbKYBkn3t.GCY18Pk6s7yN66y37fSJlLuDuMkdHtS'", s.VMConfig.Domain, s.VMConfig.IP)
	if err := s.SSH.RunSSHCommand(provisionCommand, s.VMConfig.SSHPort, 2*time.Minute, os.Stdout, os.Stderr); err != nil {
		return &ProvisionVMError{err}
	}

	return nil
}

func (s *Stopped) Status() string {
	return "Stopped"
}

func (s *Stopped) Suspend() error {
	s.UI.Say("Your VM is currently stopped and cannot be suspended.")
	return nil
}

func (s *Stopped) Resume() error {
	s.UI.Say("Your VM is currently stopped. Only a suspended VM can be resumed.")
	return nil
}

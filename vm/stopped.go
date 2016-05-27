package vm

import (
	"fmt"
	"os"
	"time"
)

type Stopped struct {
	Name    string
	Domain  string
	IP      string
	SSHPort string

	VBox VBox
	SSH  SSH
	UI   UI
}

func (s *Stopped) Stop() error {
	s.UI.Say("PCF Dev is stopped")
	return nil
}

func (s *Stopped) Start() error {
	s.UI.Say("Starting VM...")
	_, err := s.VBox.StartVM(s.Name)
	if err != nil {
		return &StartVMError{err}
	}
	s.UI.Say("Provisioning VM...")
	err = s.SSH.RunSSHCommand(fmt.Sprintf("sudo /var/pcfdev/run %s %s '$2a$04$EpJtIJ8w6hfCwbKYBkn3t.GCY18Pk6s7yN66y37fSJlLuDuMkdHtS'", s.Domain, s.IP), s.SSHPort, 2*time.Minute, os.Stdout, os.Stderr)
	if err != nil {
		return &ProvisionVMError{err}
	}

	s.UI.Say("PCF Dev is now running")
	return nil
}

func (s *Stopped) Status() {
	s.UI.Say("Stopped")
}

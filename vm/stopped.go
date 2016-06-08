package vm

import (
	"fmt"
	"os"
	"time"

	"github.com/pivotal-cf/pcfdev-cli/config"
)

type Stopped struct {
	Name    string
	Domain  string
	IP      string
	SSHPort string

	VBox   VBox
	SSH    SSH
	UI     UI
	Config *config.VMConfig
}

func (s *Stopped) Stop() error {
	s.UI.Say("PCF Dev is stopped")
	return nil
}

func (s *Stopped) Start() error {
	s.UI.Say("Starting VM...")
	if err := s.VBox.StartVM(s.Name, s.IP, s.SSHPort, s.Domain); err != nil {
		return &StartVMError{err}
	}

	s.UI.Say("Provisioning VM...")
	provisionCommand := fmt.Sprintf("sudo /var/pcfdev/run %s %s '$2a$04$EpJtIJ8w6hfCwbKYBkn3t.GCY18Pk6s7yN66y37fSJlLuDuMkdHtS'", s.Domain, s.IP)
	if err := s.SSH.RunSSHCommand(provisionCommand, s.SSHPort, 2*time.Minute, os.Stdout, os.Stderr); err != nil {
		return &ProvisionVMError{err}
	}

	return nil
}

func (s *Stopped) Status() string {
	return "Stopped"
}

func (s *Stopped) Destroy() error {
	return s.VBox.DestroyVM(s.Name)
}

func (s *Stopped) Suspend() error {
	s.UI.Say("Your VM is currently stopped and cannot be suspended.")
	return nil
}

func (s *Stopped) Resume() error {
	s.UI.Say("Your VM is currently stopped. Only a suspended VM can be resumed.")
	return nil
}

func (s *Stopped) GetConfig() *config.VMConfig {
	return s.Config
}

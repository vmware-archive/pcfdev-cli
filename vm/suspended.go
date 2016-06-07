package vm

import "github.com/pivotal-cf/pcfdev-cli/config"

type Suspended struct {
	Name    string
	Domain  string
	IP      string
	SSHPort string
	Config  *config.VMConfig

	VBox VBox
	UI   UI
}

func (s *Suspended) Stop() error {
	s.UI.Say("Your VM is currently suspended. You must resume your VM with `cf dev resume` to shut it down.")
	return nil
}

func (s *Suspended) Start() error {
	return s.Resume()
}

func (s *Suspended) Status() {
	s.UI.Say("Suspended")
}

func (s *Suspended) Destroy() error {
	return s.VBox.DestroyVM(s.Name)
}

func (s *Suspended) Suspend() error {
	s.UI.Say("Your VM is suspended.")
	return nil
}

func (s *Suspended) Resume() error {
	s.UI.Say("Resuming VM...")
	if err := s.VBox.ResumeVM(s.Name); err != nil {
		return &ResumeVMError{err}
	}

	return nil
}

func (s *Suspended) GetConfig() *config.VMConfig {
	return s.Config
}

package vm

type Suspended struct {
	Name    string
	Domain  string
	IP      string
	SSHPort string

	VBox                VBox
	UI                  UI
	RequirementsChecker RequirementsChecker
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
	if err := s.RequirementsChecker.Check(); err != nil {
		if !s.UI.Confirm("Less than 3 GB of memory detected, continue (y/N): ") {
			s.UI.Say("Exiting...")
			return nil
		}
	}

	s.UI.Say("Resuming VM...")
	if err := s.VBox.ResumeVM(s.Name); err != nil {
		return &ResumeVMError{err}
	}

	s.UI.Say("PCF Dev is now running")
	return nil
}

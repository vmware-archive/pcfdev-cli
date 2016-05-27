package vm

type Running struct {
	Name    string
	Domain  string
	IP      string
	SSHPort string

	VBox VBox
	UI   UI
}

func (r *Running) Stop() error {
	r.UI.Say("Stopping VM...")
	err := r.VBox.StopVM(r.Name)
	if err != nil {
		return &StopVMError{err}
	}
	r.UI.Say("PCF Dev is now stopped")
	return nil
}

func (r *Running) Start() error {
	r.UI.Say("PCF Dev is running")
	return nil
}

func (r *Running) Status() {
	r.UI.Say("Running")
}

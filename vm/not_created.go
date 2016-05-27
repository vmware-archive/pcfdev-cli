package vm

type NotCreated struct {
	Name string

	VBox    VBox
	UI      UI
	Builder Builder
}

func (n *NotCreated) Stop() error {
	conflict, err := n.VBox.ConflictingVMPresent(n.Name)
	if err != nil {
		return &StopVMError{err}
	}
	if conflict {
		return &OldVMError{}
	}

	n.UI.Say("PCF Dev VM has not been created")
	return nil
}

func (n *NotCreated) Start() error {
	conflict, err := n.VBox.ConflictingVMPresent(n.Name)
	if err != nil {
		return &StartVMError{err}
	}
	if conflict {
		return &OldVMError{}
	}

	n.UI.Say("Importing VM...")
	if err := n.VBox.ImportVM(n.Name); err != nil {
		return &StartVMError{err}
	}
	n.UI.Say("PCF Dev is now imported to Virtualbox")

	stoppedVM, err := n.Builder.VM(n.Name)
	if err != nil {
		return &StartVMError{err}
	}
	if err := stoppedVM.Start(); err != nil {
		return &StartVMError{err}
	}
	return nil
}

func (n *NotCreated) Status() {
	n.UI.Say("Not Created")
}

func (n *NotCreated) Destroy() error {
	return nil
}

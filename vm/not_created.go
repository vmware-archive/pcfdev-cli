package vm

import "github.com/pivotal-cf/pcfdev-cli/config"

type NotCreated struct {
	Name string

	VBox    VBox
	UI      UI
	Builder Builder
	Config  *config.VMConfig
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
	if err := n.VBox.ImportVM(n.Name, n.Config); err != nil {
		return &ImportVMError{err}
	}

	stoppedVM, err := n.Builder.VM(n.Name, n.Config)
	if err != nil {
		return &StartVMError{err}
	}
	if err := stoppedVM.Start(); err != nil {
		return &StartVMError{err}
	}
	return nil
}

func (n *NotCreated) Status() string {
	return "Not Created"
}

func (n *NotCreated) Destroy() error {
	return nil
}

func (n *NotCreated) Suspend() error {
	n.UI.Say("No VM running, cannot suspend.")
	return nil
}

func (n *NotCreated) Resume() error {
	n.UI.Say("No VM suspended, cannot resume.")
	return nil
}

func (n *NotCreated) GetConfig() *config.VMConfig {
	return n.Config
}

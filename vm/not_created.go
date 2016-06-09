package vm

import (
	"errors"
	"fmt"

	"github.com/pivotal-cf/pcfdev-cli/config"
)

type NotCreated struct {
	Name string

	VBox    VBox
	UI      UI
	Builder Builder
	Config  *config.Config
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

func (n *NotCreated) VerifyStartOpts(opts *StartOpts) error {
	var memory uint64
	if opts.Memory != uint64(0) {
		if opts.Memory < n.Config.MinMemory {
			return fmt.Errorf("PCF Dev requires at least %d MB of memory to run", n.Config.MinMemory)
		}
		memory = opts.Memory

	} else {
		memory = n.Config.DefaultMemory
	}
	if memory > n.Config.FreeMemory {
		if !n.UI.Confirm(fmt.Sprintf("Less than %d MB of free memory detected, continue (y/N): ", memory)) {
			return errors.New("user declined to continue, exiting")
		}
	}
	return nil
}

func (n *NotCreated) Start(opts *StartOpts) error {
	conflict, err := n.VBox.ConflictingVMPresent(n.Name)
	if err != nil {
		return &StartVMError{err}
	}
	if conflict {
		return &OldVMError{}
	}

	var memory uint64
	n.UI.Say("Importing VM...")
	if opts.Memory != uint64(0) {
		memory = opts.Memory
	} else {
		memory = n.Config.DefaultMemory
	}
	if err := n.VBox.ImportVM(n.Name, &config.VMConfig{
		Memory: memory,
	}); err != nil {
		return &ImportVMError{err}
	}

	stoppedVM, err := n.Builder.VM(n.Name)
	if err != nil {
		return &StartVMError{err}
	}
	if err := stoppedVM.Start(&StartOpts{}); err != nil {
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

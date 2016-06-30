package vm

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pivotal-cf/pcfdev-cli/config"
)

type NotCreated struct {
	VBox     VBox
	UI       UI
	Builder  Builder
	Config   *config.Config
	VMConfig *config.VMConfig
	FS       FS
}

func (n *NotCreated) Stop() error {
	n.UI.Say("PCF Dev VM has not been created.")
	return nil
}

func (n *NotCreated) VerifyStartOpts(opts *StartOpts) error {
	if opts.OVAPath == "" {
		if err := n.verifyMemory(opts); err != nil {
			return err
		}
	} else {
		exists, err := n.FS.Exists(opts.OVAPath)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("no file found at %s", opts.OVAPath)
		}
	}
	if opts.CPUs < 0 {
		return errors.New("cannot start with less than one core")
	}

	if len(opts.Services) != 0 {
		var disallowedServices []string

		for _, service := range strings.Split(opts.Services, ",") {
			switch service {
			case "all", "none", "redis", "rabbitmq", "mysql":
			default:
				disallowedServices = append(disallowedServices, service)
			}
		}

		if len(disallowedServices) > 0 {
			return fmt.Errorf("invalid services specified: %s", strings.Join(disallowedServices, ", "))
		}
	}

	return nil
}

func (n *NotCreated) verifyMemory(opts *StartOpts) error {
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
	var memory uint64
	if opts.Memory != uint64(0) {
		memory = opts.Memory
	} else {
		memory = n.Config.DefaultMemory
	}

	var cpus int
	if opts.CPUs != 0 {
		cpus = opts.CPUs
	} else {
		cpus = n.Config.DefaultCPUs
	}

	var ovaPath string
	if opts.OVAPath != "" {
		ovaPath = opts.OVAPath
	} else {
		ovaPath = filepath.Join(n.Config.OVADir, n.VMConfig.Name+".ova")
	}

	n.UI.Say(fmt.Sprintf("Allocating %d MB out of %d MB total system memory (%d MB free).", memory, n.Config.TotalMemory, n.Config.FreeMemory))
	n.UI.Say("Importing VM...")
	if err := n.VBox.ImportVM(&config.VMConfig{
		Name:    n.VMConfig.Name,
		Memory:  memory,
		CPUs:    cpus,
		OVAPath: ovaPath,
	}); err != nil {
		return &ImportVMError{err}
	}

	stoppedVM, err := n.Builder.VM(n.VMConfig.Name)
	if err != nil {
		return &StartVMError{err}
	}
	if err := stoppedVM.Start(&StartOpts{Services: opts.Services}); err != nil {
		return err
	}
	return nil
}

func (n *NotCreated) Status() string {
	return "Not Created"
}

func (n *NotCreated) Suspend() error {
	n.UI.Say("No VM running, cannot suspend.")
	return nil
}

func (n *NotCreated) Resume() error {
	n.UI.Say("No VM suspended, cannot resume.")
	return nil
}

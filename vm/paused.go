package vm

import (
	"errors"
	"path/filepath"
	"time"

	"github.com/pivotal-cf/pcfdev-cli/config"
)

type Paused struct {
	VMConfig *config.VMConfig
	Config   *config.Config

	UI        UI
	VBox      VBox
	SSHClient SSH
	FS        FS
}

func (p *Paused) Stop() error {
	p.UI.Say("Your VM is currently suspended. You must resume your VM with `cf dev resume` to shut it down.")
	return nil
}

func (p *Paused) VerifyStartOpts(opts *StartOpts) error {
	if opts.Memory != uint64(0) {
		return errors.New("memory cannot be changed once the vm has been created")
	}
	if opts.CPUs != 0 {
		return errors.New("cores cannot be changed once the vm has been created")
	}
	if opts.Services != "" {
		return errors.New("services cannot be changed once the vm has been created")
	}
	if opts.Domain != "" {
		return errors.New("the -d flag cannot be used if the VM has already been created")
	}
	if opts.IP != "" {
		return errors.New("the -i flag cannot be used if the VM has already been created")
	}
	return nil
}

func (p *Paused) Start(opts *StartOpts) error {
	return p.Resume()
}

func (p *Paused) Provision(opts *StartOpts) error {
	return nil
}

func (p *Paused) Status() string {
	return "Suspended - system memory for the VM is still allocated. Resume and suspend to suspend pcfdev VM to the disk."
}

func (p *Paused) Suspend() error {
	p.UI.Say("Your VM is suspended and system memory for the VM is still allocated. Resume and suspend to suspend pcfdev VM to the disk.")
	return nil
}

func (p *Paused) Resume() error {
	p.UI.Say("Resuming VM...")
	if err := p.VBox.ResumePausedVM(p.VMConfig); err != nil {
		return &ResumeVMError{err}
	}

	privateKeyBytes, err := p.FS.Read(filepath.Join(p.Config.VMDir, "key.pem"))
	if err != nil {
		return &ResumeVMError{err}
	}

	if err := p.SSHClient.WaitForSSH(p.VMConfig.IP, "22", string(privateKeyBytes), 5*time.Minute); err != nil {
		return &ResumeVMError{err}
	}

	p.UI.Say("PCF Dev is now running.")

	return nil
}

func (p *Paused) GetDebugLogs() error {
	p.UI.Say("Your VM is suspended. Resume to retrieve debug logs.")
	return nil
}

func (p *Paused) Trust(startOps *StartOpts) error {
	p.UI.Say("Your VM is suspended. Resume to trust VM certificates.")
	return nil
}

func (p *Paused) Target(autoTarget bool) error {
	p.UI.Say("Your VM is suspended. Resume to target PCF Dev.")
	return nil
}

func (p *Paused) SSH() error {
	p.UI.Say("Your VM is suspended. Resume to SSH to PCF Dev.")
	return nil
}

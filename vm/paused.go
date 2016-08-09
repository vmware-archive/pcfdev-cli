package vm

type Paused struct {
	UI          UI
	SuspendedVM *Suspended
}

func (p *Paused) Stop() error {
	return p.SuspendedVM.Stop()
}

func (p *Paused) VerifyStartOpts(opts *StartOpts) error {
	return p.SuspendedVM.VerifyStartOpts(opts)
}

func (p *Paused) Start(opts *StartOpts) error {
	return p.SuspendedVM.Start(opts)
}

func (p *Paused) Provision() error {
	return p.SuspendedVM.Provision()
}

func (p *Paused) Status() string {
	return "Suspended - system memory for the VM is still allocated. Resume and suspend to suspend pcfdev VM to the disk."
}

func (p *Paused) Suspend() error {
	p.UI.Say("Your VM is suspended and system memory for the VM is still allocated. Resume and suspend to suspend pcfdev VM to the disk.")
	return nil
}

func (p *Paused) Resume() error {
	return p.SuspendedVM.Resume()
}

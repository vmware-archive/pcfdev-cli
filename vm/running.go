package vm

import (
	"errors"
	"fmt"
	"time"

	"github.com/docker/docker/pkg/term"

	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/ssh"
)

type Running struct {
	Config   *config.Config
	VMConfig *config.VMConfig

	VBox       VBox
	FS         FS
	UI         UI
	SSHClient  SSH
	Builder    Builder
	LogFetcher LogFetcher
	CertStore  CertStore
	CmdRunner  CmdRunner
	HelpText   HelpText
}

func (r *Running) Stop() error {
	r.UI.Say("Stopping VM...")
	err := r.VBox.StopVM(r.VMConfig)
	if err != nil {
		return &StopVMError{err}
	}
	r.UI.Say("PCF Dev is now stopped.")
	return nil
}

func (r *Running) Provision(opts *StartOpts) error {
	privateKeyBytes, err := r.FS.Read(r.Config.PrivateKeyPath)
	if err != nil {
		return err
	}

	addresses := []ssh.SSHAddress{
		{
			IP:   "127.0.0.1",
			Port: r.VMConfig.SSHPort,
		},
		{
			IP:   r.VMConfig.IP,
			Port: "22",
		},
	}

	if _, err := r.SSHClient.GetSSHOutput("sudo rm -f /run/pcfdev-healthcheck", addresses, privateKeyBytes, 30*time.Second); err != nil {
		return err
	}
	unprovisionedVM, err := r.Builder.VM(r.VMConfig.Name)
	if err != nil {
		return err
	}
	return unprovisionedVM.Provision(opts)
}

func (r *Running) VerifyStartOpts(opts *StartOpts) error {
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

func (r *Running) Start(opts *StartOpts) error {
	r.UI.Say("PCF Dev is running.")
	return nil
}

func (r *Running) Status() string {
	return fmt.Sprintf("Running\nCLI Login: cf login -a https://api.%s --skip-ssl-validation\nApps Manager URL: https://apps.%s\nAdmin user => Email: admin / Password: admin\nRegular user => Email: user / Password: pass", r.VMConfig.Domain, r.VMConfig.Domain)
}

func (r *Running) Suspend() error {
	r.UI.Say("Suspending VM...")
	if err := r.VBox.SuspendVM(r.VMConfig); err != nil {
		return &SuspendVMError{err}
	}

	r.UI.Say("PCF Dev is now suspended.")
	return nil
}

func (r *Running) Resume() error {
	r.UI.Say("PCF Dev is running.")

	return nil
}

func (r *Running) Trust(startOpts *StartOpts) error {
	privateKeyBytes, err := r.FS.Read(r.Config.PrivateKeyPath)
	if err != nil {
		return &TrustError{err}
	}

	addresses := []ssh.SSHAddress{
		{
			IP:   "127.0.0.1",
			Port: r.VMConfig.SSHPort,
		},
		{
			IP:   r.VMConfig.IP,
			Port: "22",
		},
	}

	output, err := r.SSHClient.GetSSHOutput("cat /var/pcfdev/openssl/ca_cert.pem", addresses, privateKeyBytes, 5*time.Minute)
	if err != nil {
		return &TrustError{err}
	}

	if startOpts.PrintCA {
		r.UI.Say(output)
		return nil
	}

	if err := r.CertStore.Store(output); err != nil {
		return &TrustError{err}
	}

	r.UI.Say(fmt.Sprintf("***Warning: a self-signed certificate for *.%s has been inserted into your OS certificate store. To remove this certificate, run: cf dev untrust***", r.VMConfig.Domain))

	return nil
}

func (r *Running) Target(autoTarget bool) error {
	if _, err := r.CmdRunner.Run(
		"cf",
		"login",
		"-a", fmt.Sprintf("api.%s", r.VMConfig.Domain),
		"--skip-ssl-validation",
		"-u", "user",
		"-p", "pass",
		"-o", "pcfdev-org",
		"-s", "pcfdev-space",
	); err != nil {
		return &TargetError{err}
	}

	if !autoTarget {
		r.UI.Say(fmt.Sprintf("Successfully logged in to api.%s as user.", r.VMConfig.Domain))
	}

	return nil
}

func (r *Running) GetDebugLogs() error {
	if err := r.LogFetcher.FetchLogs(); err != nil {
		return &FetchLogsError{err}
	}

	r.UI.Say("Debug logs written to pcfdev-debug.tgz. While some scrubbing has taken place, please remove any remaining sensitive information from these logs before sharing.")
	return nil
}

func (r *Running) SSH() error {
	privateKeyBytes, err := r.FS.Read(r.Config.PrivateKeyPath)
	if err != nil {
		return err
	}

	addresses := []ssh.SSHAddress{
		{IP: "127.0.0.1", Port: r.VMConfig.SSHPort},
		{IP: r.VMConfig.IP, Port: "22"},
	}

	stdin, stdout, stderr := term.StdStreams()
	return r.SSHClient.StartSSHSession(addresses, privateKeyBytes, 5*time.Minute, stdin, stdout, stderr)
}

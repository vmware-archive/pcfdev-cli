package vm

import (
	"errors"
	"fmt"
	"time"

	"github.com/pivotal-cf/pcfdev-cli/config"
)

type Running struct {
	VMConfig *config.VMConfig

	VBox       VBox
	FS         FS
	UI         UI
	SSH        SSH
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
	if _, err := r.SSH.GetSSHOutput("sudo rm -f /run/pcfdev-healthcheck", r.VMConfig.IP, "22", 30*time.Second); err != nil {
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
	return nil
}

func (r *Running) Start(opts *StartOpts) error {
	r.UI.Say("PCF Dev is running.")
	return nil
}

func (r *Running) Status() string {
	return fmt.Sprintf("Running\nCLI Login: cf login -a https://api.%s --skip-ssl-validation\nApps Manager URL: https://%s\nAdmin user => Email: admin / Password: admin\nRegular user => Email: user / Password: pass", r.VMConfig.Domain, r.VMConfig.Domain)
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
	output, err := r.SSH.GetSSHOutput("cat /var/pcfdev/openssl/ca_cert.pem", "127.0.0.1", r.VMConfig.SSHPort, 5*time.Minute)
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

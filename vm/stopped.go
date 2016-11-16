package vm

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/helpers"
	"github.com/pivotal-cf/pcfdev-cli/ssh"
)

type Stopped struct {
	Config   *config.Config
	VMConfig *config.VMConfig

	FS        FS
	VBox      VBox
	SSHClient SSH
	UI        UI
	Builder   Builder
}

func (s *Stopped) Stop() error {
	s.UI.Say("PCF Dev is stopped.")
	return nil
}

func (s *Stopped) VerifyStartOpts(opts *StartOpts) error {
	if opts.Memory != uint64(0) {
		return errors.New("memory cannot be changed once the vm has been created")
	}
	if opts.CPUs != 0 {
		return errors.New("cores cannot be changed once the vm has been created")
	}
	if opts.Services != "" {
		return errors.New("services cannot be changed once the vm has been created")
	}
	if opts.Registries != "" {
		return errors.New("private registries cannot be changed once the vm has been created")
	}
	if opts.Domain != "" {
		return errors.New("the -d flag cannot be used if the VM has already been created")
	}
	if opts.IP != "" {
		return errors.New("the -i flag cannot be used if the VM has already been created")
	}
	if s.VMConfig.Memory > s.Config.FreeMemory {
		if !s.UI.Confirm(fmt.Sprintf("Less than %d MB of free memory detected, continue (y/N): ", s.VMConfig.Memory)) {
			return errors.New("user declined to continue, exiting")
		}
	}
	return nil
}

func (s *Stopped) Start(opts *StartOpts) error {
	s.UI.Say("Starting VM...")
	if err := s.VBox.StartVM(s.VMConfig); err != nil {
		return &StartVMError{err}
	}

	services := []string{}
	if len(opts.Services) == 0 {
		services = append(services, "rabbitmq", "redis")
	} else {
		for _, service := range strings.Split(opts.Services, ",") {
			switch service {
			case "all":
				services = append(services, "rabbitmq", "redis", "spring-cloud-services")
			case "default":
				services = append(services, "rabbitmq", "redis")
			case "rabbitmq":
				services = append(services, "rabbitmq")
			case "redis":
				services = append(services, "redis")
			case "spring-cloud-services", "scs":
				services = append(services, "rabbitmq", "spring-cloud-services")
			}
		}
		services = helpers.RemoveDuplicates(services)
		sort.Strings(services)
	}

	registries := []string{}
	if opts.Registries != "" {
		registries = strings.Split(opts.Registries, ",")
	}

	provisionConfig := &config.ProvisionConfig{
		Domain:     s.VMConfig.Domain,
		IP:         s.VMConfig.IP,
		Provider:   s.VMConfig.Provider,
		Services:   strings.Join(services, ","),
		Registries: registries,
	}

	if opts.IP != "" {
		provisionConfig.IP = opts.IP
	}

	if opts.Domain != "" {
		provisionConfig.Domain = opts.Domain
	}

	data, err := json.Marshal(provisionConfig)
	if err != nil {
		return &StartVMError{err}
	}

	unprovisionedVM, err := s.Builder.VM(s.VMConfig.Name)
	if err != nil {
		return &StartVMError{err}
	}

	privateKeyBytes, err := s.FS.Read(s.Config.PrivateKeyPath)
	if err != nil {
		return &StartVMError{err}
	}

	addresses := []ssh.SSHAddress{
		{
			IP:   "127.0.0.1",
			Port: s.VMConfig.SSHPort,
		},
		{
			IP:   s.VMConfig.IP,
			Port: "22",
		},
	}

	if err := s.SSHClient.RunSSHCommand("echo '"+string(data)+"' | sudo tee /var/pcfdev/provision-options.json >/dev/null", addresses, privateKeyBytes, 5*time.Minute, os.Stdout, os.Stderr); err != nil {
		return &StartVMError{err}
	}

	if opts.NoProvision {
		s.UI.Say("VM will not be provisioned because '-n' (no-provision) flag was specified.")
		return nil
	}

	return unprovisionedVM.Provision(opts)
}

func (s *Stopped) Provision(opts *StartOpts) error {
	return nil
}

func (s *Stopped) Status() string {
	return "Stopped"
}

func (s *Stopped) Suspend() error {
	s.UI.Say("Your VM is currently stopped and cannot be suspended.")
	return nil
}

func (s *Stopped) Resume() error {
	s.UI.Say("Your VM is currently stopped. Only a suspended VM can be resumed.")
	return nil
}

func (s *Stopped) GetDebugLogs() error {
	s.UI.Say("Your VM is currently stopped. Start VM to retrieve debug logs.")
	return nil
}

func (s *Stopped) Trust(startOps *StartOpts) error {
	s.UI.Say("Your VM is currently stopped. Start VM to trust VM certificates.")
	return nil
}

func (s *Stopped) Target(autoTarget bool) error {
	s.UI.Say("Your VM is currently stopped. Start VM to target PCF Dev.")
	return nil
}

func (s *Stopped) SSH(opts *SSHOpts) error {
	s.UI.Say("Your VM is currently stopped. Start VM to SSH to PCF Dev.")
	return nil
}

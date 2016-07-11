package vm

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/helpers"
)

type Stopped struct {
	Config   *config.Config
	VMConfig *config.VMConfig

	FS   FS
	VBox VBox
	SSH  SSH
	UI   UI
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

	if err := s.FS.Remove(filepath.Join(s.Config.VMDir, "provision-options")); err != nil {
		return &StartVMError{err}
	}

	provisionConfig := &config.ProvisionConfig{
		Domain:   s.VMConfig.Domain,
		IP:       s.VMConfig.IP,
		Services: strings.Join(services, ","),
	}

	data, err := json.Marshal(provisionConfig)
	if err != nil {
		return &StartVMError{err}
	}

	if err := s.FS.Write(filepath.Join(s.Config.VMDir, "provision-options"), bytes.NewReader(data)); err != nil {
		return &StartVMError{err}
	}

	if opts.NoProvision {
		s.UI.Say("VM will not be provisioned because '-n' (no-provision) flag was specified.")
		return nil
	}

	return s.Provision()
}

func (s *Stopped) Provision() error {
	if exists, err := s.FS.Exists(filepath.Join(s.Config.VMDir, "provision-options")); !exists || err != nil {
		return &ProvisionVMError{errors.New("missing provision configuration")}
	}

	data, err := s.FS.Read(filepath.Join(s.Config.VMDir, "provision-options"))
	if err != nil {
		return &ProvisionVMError{err}
	}

	provisionConfig := &config.ProvisionConfig{}
	if err := json.Unmarshal(data, provisionConfig); err != nil {
		return &ProvisionVMError{err}
	}

	s.UI.Say("Provisioning VM...")
	provisionCommand := fmt.Sprintf("sudo -H /var/pcfdev/run %s %s %s", provisionConfig.Domain, provisionConfig.IP, provisionConfig.Services)
	if err := s.SSH.RunSSHCommand(provisionCommand, s.VMConfig.SSHPort, 5*time.Minute, os.Stdout, os.Stderr); err != nil {
		return &ProvisionVMError{err}
	}

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

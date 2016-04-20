package vbox

import "fmt"

//go:generate mockgen -package mocks -destination mocks/driver.go github.com/pivotal-cf/pcfdev-cli/vbox Driver
type Driver interface {
	VBoxManage(arg ...string) ([]byte, error)
	StopVM(string) error
	StartVM(string) error
	DestroyVM(string) error
	CreateHostOnlyInterface(string) (string, error)
	DestroyHostOnlyInterface(string) (string, error)
	GetVBoxNetName(string) (string, error)
	AttachNetworkInterface(string, string) error
	ForwardPort(string, string, string, string) error
	VMExists(string) (bool, error)
	GetHostForwardPort(string, string) (string, error)
	IsVMRunning(string) bool
}

//go:generate mockgen -package mocks -destination mocks/ssh.go github.com/pivotal-cf/pcfdev-cli/vbox SSH
type SSH interface {
	FreePort() (string, error)
	RunSSHCommand(string, string) error
}

type VBox struct {
	Driver Driver
	SSH    SSH
}

type VM struct {
	SSHPort string
	Name    string
	IP      string
}

func (v *VBox) StartVM(name string) (*VM, error) {
	var sshPort string
	ip := "192.168.11.11"
	sshPort, err := v.Driver.GetHostForwardPort(name, "ssh")
	if err != nil {
		return nil, fmt.Errorf("failed to get host port for ssh forwarding: %s", err)
	}
	vm := &VM{
		SSHPort: sshPort,
		Name:    name,
		IP:      ip,
	}

	err = v.Driver.StartVM(name)
	if err != nil {
		return nil, fmt.Errorf("failed to start vm: %s", err)
	}
	err = v.SSH.RunSSHCommand(fmt.Sprintf("echo -e \"auto eth1\niface eth1 inet static\naddress %s\nnetmask 255.255.255.0\" | sudo tee -a /etc/network/interfaces", ip), sshPort)
	if err != nil {
		return nil, fmt.Errorf("failed to set static ip: %s", err)
	}
	err = v.Driver.StopVM(name)
	if err != nil {
		return nil, fmt.Errorf("failed to stop vm: %s", err)
	}
	err = v.Driver.StartVM(vm.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to start vm: %s", err)
	}

	return vm, nil
}

func (v *VBox) ImportVM(path string, name string) error {
	var sshPort string
	sshPort, err := v.SSH.FreePort()
	if err != nil {
		return fmt.Errorf("failed to aquire random port: %s", err)
	}
	_, err = v.Driver.VBoxManage("import", path)
	if err != nil {
		return fmt.Errorf("failed to import ova: %s", err)
	}
	vboxnet, err := v.Driver.CreateHostOnlyInterface("192.168.11.1")
	if err != nil {
		return fmt.Errorf("failed to create host only interface: %s", err)
	}
	err = v.Driver.AttachNetworkInterface(vboxnet, name)
	if err != nil {
		return fmt.Errorf("failed to attach interface: %s", err)
	}
	err = v.Driver.ForwardPort(name, "ssh", "22", sshPort)
	if err != nil {
		return fmt.Errorf("failed to forward ssh port: %s", err)
	}
	return nil
}

func (v *VBox) DestroyVM(name string) error {
	return v.Driver.DestroyVM(name)
}

func (v *VBox) StopVM(name string) error {
	return v.Driver.StopVM(name)
}

func (v *VBox) IsVMRunning(name string) bool {
	return v.Driver.IsVMRunning(name)
}

func (v *VBox) IsVMImported(name string) (bool, error) {
	exists, err := v.Driver.VMExists(name)
	if err != nil {
		return false, fmt.Errorf("failed to query for VM: %s", err)
	}
	return exists, nil
}

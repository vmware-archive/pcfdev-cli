package vbox

import "fmt"

//go:generate mockgen -package mocks -destination mocks/driver.go github.com/pivotal-cf/pcfdev-cli/vbox Driver
type Driver interface {
	VBoxManage(arg ...string) ([]byte, error)
	StopVM(string) error
	StartVM(string) error
	DestroyVM(string) error
	CreateHostOnlyInterface(string) (string, error)
	AttachNetworkInterface(string, string) error
	ForwardPort(string, string, string, string) error
	VMExists(string) (bool, error)
	GetHostForwardPort(string, string) (string, error)
	IsVMRunning(string) bool
}

//go:generate mockgen -package mocks -destination mocks/ssh.go github.com/pivotal-cf/pcfdev-cli/vbox SSH
type SSH interface {
	GenerateAddress() (host string, port string, err error)
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

const (
	StatusRunning    = "Running"
	StatusStopped    = "Stopped"
	StatusNotCreated = "Not created"
)

func (v *VBox) StartVM(name string) (*VM, error) {
	var sshPort string
	ip := "192.168.11.11"
	sshPort, err := v.Driver.GetHostForwardPort(name, "ssh")
	if err != nil {
		return nil, err
	}
	vm := &VM{
		SSHPort: sshPort,
		Name:    name,
		IP:      ip,
	}

	err = v.Driver.StartVM(name)
	if err != nil {
		return nil, err
	}
	err = v.SSH.RunSSHCommand(fmt.Sprintf("echo -e \"auto eth1\niface eth1 inet static\naddress %s\nnetmask 255.255.255.0\" | sudo tee -a /etc/network/interfaces", ip), sshPort)
	if err != nil {
		return nil, err
	}
	err = v.Driver.StopVM(name)
	if err != nil {
		return nil, err
	}
	err = v.Driver.StartVM(vm.Name)
	if err != nil {
		return nil, err
	}

	return vm, nil
}

func (v *VBox) ImportVM(path string, name string) error {
	var sshPort string
	_, sshPort, err := v.SSH.GenerateAddress()
	if err != nil {
		return err
	}
	_, err = v.Driver.VBoxManage("import", path)
	if err != nil {
		return err
	}
	vboxnet, err := v.Driver.CreateHostOnlyInterface("192.168.11.1")
	if err != nil {
		return err
	}
	err = v.Driver.AttachNetworkInterface(vboxnet, name)
	if err != nil {
		return err
	}

	err = v.Driver.ForwardPort(name, "ssh", "22", sshPort)
	if err != nil {
		return err
	}
	return nil
}

func (v *VBox) DestroyVM(name string) error {
	status, err := v.Status(name)
	if err != nil {
		return err
	}

	if status == StatusRunning {
		err = v.StopVM(name)
		if err != nil {
			return err
		}
	}

	return v.Driver.DestroyVM(name)
}

func (v *VBox) StopVM(name string) error {
	return v.Driver.StopVM(name)
}

func (v *VBox) Status(name string) (string, error) {
	exists, err := v.Driver.VMExists(name)
	if err != nil {
		return "", err
	}

	if !exists {
		return StatusNotCreated, nil
	}

	if v.Driver.IsVMRunning(name) {
		return StatusRunning, nil
	}

	return StatusStopped, nil
}

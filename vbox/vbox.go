package vbox

import "fmt"

//go:generate mockgen -package mocks -destination mocks/driver.go github.com/pivotal-cf/pcfdev-cli/vbox Driver
type Driver interface {
	VBoxManage(arg ...string) (output []byte, err error)
	StartVM(vmName string) error
	VMExists(vmName string) (exists bool, err error)
	IsVMRunning(vmName string) bool
	StopVM(vmName string) error
	DestroyVM(vmName string) error
	CreateHostOnlyInterface(ip string) (interfaceName string, err error)
	AttachNetworkInterface(interfaceName string, vmName string) error
	ForwardPort(vmName string, ruleName string, hostPort string, guestPort string) error
	GetHostForwardPort(vmName string, ruleName string) (port string, err error)
}

//go:generate mockgen -package mocks -destination mocks/ssh.go github.com/pivotal-cf/pcfdev-cli/vbox SSH
type SSH interface {
	GenerateAddress() (host string, port string, err error)
	RunSSHCommand(command string, port string) error
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

func (v *VBox) StartVM(vmName string) (vm *VM, err error) {
	ip := "192.168.11.11"
	sshPort, err := v.Driver.GetHostForwardPort(vmName, "ssh")
	if err != nil {
		return nil, err
	}
	vm = &VM{
		SSHPort: sshPort,
		Name:    vmName,
		IP:      ip,
	}

	err = v.Driver.StartVM(vmName)
	if err != nil {
		return nil, err
	}
	err = v.SSH.RunSSHCommand(fmt.Sprintf("echo -e \"auto eth1\niface eth1 inet static\naddress %s\nnetmask 255.255.255.0\" | sudo tee -a /etc/network/interfaces", ip), sshPort)
	if err != nil {
		return nil, err
	}
	err = v.Driver.StopVM(vmName)
	if err != nil {
		return nil, err
	}
	err = v.Driver.StartVM(vm.Name)
	if err != nil {
		return nil, err
	}

	return vm, nil
}

func (v *VBox) ImportVM(path string, vmName string) error {
	_, sshPort, err := v.SSH.GenerateAddress()
	if err != nil {
		return err
	}
	_, err = v.Driver.VBoxManage("import", path)
	if err != nil {
		return err
	}
	interfaceName, err := v.Driver.CreateHostOnlyInterface("192.168.11.1")
	if err != nil {
		return err
	}
	err = v.Driver.AttachNetworkInterface(interfaceName, vmName)
	if err != nil {
		return err
	}

	err = v.Driver.ForwardPort(vmName, "ssh", sshPort, "22")
	if err != nil {
		return err
	}
	return nil
}

func (v *VBox) DestroyVM(vmName string) error {
	status, err := v.Status(vmName)
	if err != nil {
		return err
	}

	if status == StatusRunning {
		err = v.StopVM(vmName)
		if err != nil {
			return err
		}
	}

	return v.Driver.DestroyVM(vmName)
}

func (v *VBox) StopVM(vmName string) error {
	return v.Driver.StopVM(vmName)
}

func (v *VBox) Status(vmName string) (string, error) {
	exists, err := v.Driver.VMExists(vmName)
	if err != nil {
		return "", err
	}

	if !exists {
		return StatusNotCreated, nil
	}

	if v.Driver.IsVMRunning(vmName) {
		return StatusRunning, nil
	}

	return StatusStopped, nil
}

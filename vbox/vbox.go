package vbox

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"time"

	"github.com/pivotal-cf/pcfdev-cli/address"
	"github.com/pivotal-cf/pcfdev-cli/network"
)

//go:generate mockgen -package mocks -destination mocks/driver.go github.com/pivotal-cf/pcfdev-cli/vbox Driver
type Driver interface {
	VBoxManage(arg ...string) (output []byte, err error)
	StartVM(vmName string) error
	VMExists(vmName string) (exists bool, err error)
	IsVMRunning(vmName string) bool
	PowerOffVM(vmName string) error
	StopVM(vmName string) error
	DestroyVM(vmName string) error
	VMs() (vms []string, err error)
	RunningVMs() (vms []string, err error)
	CreateHostOnlyInterface(ip string) (interfaceName string, err error)
	AttachNetworkInterface(interfaceName string, vmName string) error
	ForwardPort(vmName string, ruleName string, hostPort string, guestPort string) error
	GetHostForwardPort(vmName string, ruleName string) (port string, err error)
	GetHostOnlyInterfaces() (interfaces []*network.Interface, err error)
	GetVMIP(vmName string) (vmIP string, err error)
	SetMemory(vmName string, memory uint64) error
}

//go:generate mockgen -package mocks -destination mocks/ssh.go github.com/pivotal-cf/pcfdev-cli/vbox SSH
type SSH interface {
	GenerateAddress() (host string, port string, err error)
	RunSSHCommand(command string, port string, timeout time.Duration, stdout io.Writer, stderr io.Writer) error
}

//go:generate mockgen -package mocks -destination mocks/picker.go github.com/pivotal-cf/pcfdev-cli/vbox NetworkPicker
type NetworkPicker interface {
	SelectAvailableNetworkInterface(candidates []*network.Interface) (selectedInterface *network.Interface, exists bool, err error)
}

//go:generate mockgen -package mocks -destination mocks/address.go github.com/pivotal-cf/pcfdev-cli/vbox Address
type Address interface {
	DomainForIP(vmIP string) (domain string, err error)
	SubnetForIP(vmIP string) (subnetIP string, err error)
}

//go:generate mockgen -package mocks -destination mocks/system.go github.com/pivotal-cf/pcfdev-cli/vbox System
type System interface {
	FreeMemory() (memory uint64, err error)
}

//go:generate mockgen -package mocks -destination mocks/config.go github.com/pivotal-cf/pcfdev-cli/vbox Config
type Config interface {
	GetMaxMemory() (memory uint64)
	GetMinMemory() (memory uint64)
	GetDesiredMemory() (memory uint64, err error)
}

type VBox struct {
	Driver Driver
	SSH    SSH
	Picker NetworkPicker
	System System
	Config Config
}

type VM struct {
	Domain  string
	IP      string
	Name    string
	SSHPort string
}

const (
	StatusRunning    = "Running"
	StatusStopped    = "Stopped"
	StatusNotCreated = "Not created"
)

func (v *VBox) StartVM(vmName string) (vm *VM, err error) {
	ip, err := v.Driver.GetVMIP(vmName)
	if err != nil {
		return nil, err
	}
	sshPort, err := v.Driver.GetHostForwardPort(vmName, "ssh")
	if err != nil {
		return nil, err
	}
	domain, err := address.DomainForIP(ip)
	if err != nil {
		return nil, err
	}

	vm = &VM{
		SSHPort: sshPort,
		Name:    vmName,
		IP:      ip,
		Domain:  domain,
	}

	err = v.Driver.StartVM(vmName)
	if err != nil {
		return nil, err
	}
	err = v.SSH.RunSSHCommand(fmt.Sprintf("echo -e \"auto eth1\niface eth1 inet static\naddress %s\nnetmask 255.255.255.0\" | sudo tee -a /etc/network/interfaces", ip), sshPort, 2*time.Minute, ioutil.Discard, ioutil.Discard)
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

	vboxInterfaces, err := v.Driver.GetHostOnlyInterfaces()
	if err != nil {
		return err
	}

	selectedInterface, exists, err := v.Picker.SelectAvailableNetworkInterface(vboxInterfaces)
	if err != nil {
		return err
	}
	if !exists {
		selectedInterface.Name, err = v.Driver.CreateHostOnlyInterface(selectedInterface.IP)
		if err != nil {
			return err
		}
	}

	err = v.Driver.AttachNetworkInterface(selectedInterface.Name, vmName)
	if err != nil {
		return err
	}

	err = v.Driver.ForwardPort(vmName, "ssh", sshPort, "22")
	if err != nil {
		return err
	}

	memory, err := v.computeMemory()
	if err != nil {
		return err
	}
	if err := v.Driver.SetMemory(vmName, memory); err != nil {
		return err
	}

	return nil
}

func (v *VBox) computeMemory() (uint64, error) {
	var memory uint64
	desiredMemory, err := v.Config.GetDesiredMemory()
	if err != nil {
		return uint64(0), err
	}
	if desiredMemory != 0 {
		memory = desiredMemory
	} else {
		maxMemory := v.Config.GetMaxMemory()
		minMemory := v.Config.GetMinMemory()
		freeMemory, err := v.System.FreeMemory()
		if err != nil {
			return uint64(0), err
		}
		if freeMemory <= minMemory {
			memory = minMemory
		} else if freeMemory >= maxMemory {
			memory = maxMemory
		} else {
			memory = freeMemory
		}
	}
	return memory, nil
}

func (v *VBox) DestroyVMs(vmNames []string) error {
	for _, vmName := range vmNames {
		status, err := v.Status(vmName)
		if err != nil {
			return err
		}

		if status == StatusRunning {
			err = v.Driver.PowerOffVM(vmName)
			if err != nil {
				return err
			}
		}

		err = v.Driver.DestroyVM(vmName)
		if err != nil {
			return err
		}
	}

	return nil
}

func (v *VBox) ConflictingVMPresent(vmName string) (conflict bool, err error) {
	vms, err := v.Driver.RunningVMs()
	if err != nil {
		return false, err
	}

	for _, vm := range vms {
		if strings.HasPrefix(vm, "pcfdev-") && vm != vmName {
			return true, nil
		}
	}
	return false, nil
}

func (v *VBox) StopVM(vmName string) error {
	return v.Driver.StopVM(vmName)
}

func (v *VBox) Status(vmName string) (status string, err error) {
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

func (v *VBox) GetPCFDevVMs() ([]string, error) {
	vms, err := v.Driver.VMs()
	if err != nil {
		return []string{}, err
	}

	pcfdevVMs := []string{}

	for _, vm := range vms {
		if strings.HasPrefix(vm, "pcfdev-") {
			pcfdevVMs = append(pcfdevVMs, vm)
		}
	}

	return pcfdevVMs, nil
}

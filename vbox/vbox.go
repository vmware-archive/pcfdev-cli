package vbox

import (
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	"github.com/pivotal-cf/pcfdev-cli/address"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/network"
)

//go:generate mockgen -package mocks -destination mocks/driver.go github.com/pivotal-cf/pcfdev-cli/vbox Driver
type Driver interface {
	VBoxManage(arg ...string) (output []byte, err error)
	StartVM(vmName string) error
	VMExists(vmName string) (exists bool, err error)
	PowerOffVM(vmName string) error
	StopVM(vmName string) error
	SuspendVM(vmName string) error
	ResumeVM(vmName string) error
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
	GetVirtualSystemNumbersOfHardDiskImages(ovaPath string) (virtualSystemNumbers []string, err error)
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

type VBox struct {
	Driver Driver
	SSH    SSH
	Picker NetworkPicker
	System System
	Config *config.Config
}

const (
	StatusRunning    = "Running"
	StatusSuspended  = "Suspended"
	StatusStopped    = "Stopped"
	StatusNotCreated = "Not created"
)

func (v *VBox) StartVM(vmName string, ip string, sshPort string, domain string) error {
	if err := v.Driver.StartVM(vmName); err != nil {
		return err
	}

	if err := v.configureNetwork(ip, sshPort); err != nil {
		return err
	}
	if err := v.configureEnvironment(ip, sshPort); err != nil {
		return err
	}

	if err := v.Driver.StopVM(vmName); err != nil {
		return err
	}

	return v.Driver.StartVM(vmName)
}

func (v *VBox) configureNetwork(ip string, sshPort string) error {
	return v.SSH.RunSSHCommand(fmt.Sprintf("echo -e \"auto eth1\niface eth1 inet static\naddress %s\nnetmask 255.255.255.0\" | sudo tee -a /etc/network/interfaces", ip), sshPort, 2*time.Minute, ioutil.Discard, ioutil.Discard)
}

func (v *VBox) configureEnvironment(ip string, sshPort string) error {
	proxySettings, err := v.proxySettings(ip)
	if err != nil {
		return err
	}

	return v.SSH.RunSSHCommand(fmt.Sprintf("echo -e \"%s\" | sudo tee -a /etc/environment", proxySettings), sshPort, 2*time.Minute, ioutil.Discard, ioutil.Discard)
}

func (v *VBox) proxySettings(ip string) (settings string, err error) {
	subnet, err := address.SubnetForIP(ip)
	if err != nil {
		return "", err
	}

	domain, err := address.DomainForIP(ip)
	if err != nil {
		return "", err
	}

	httpProxy := strings.Replace(v.Config.HTTPProxy, "127.0.0.1", subnet, -1)
	httpsProxy := strings.Replace(v.Config.HTTPSProxy, "127.0.0.1", subnet, -1)
	noProxy := strings.Join([]string{
		"localhost",
		"127.0.0.1",
		subnet,
		ip,
		domain,
		v.Config.NoProxy}, ",")

	return strings.Join([]string{
		"HTTP_PROXY=" + httpProxy,
		"HTTPS_PROXY=" + httpsProxy,
		"NO_PROXY=" + noProxy,
		"http_proxy=" + httpProxy,
		"https_proxy=" + httpsProxy,
		"no_proxy=" + noProxy,
	}, "\n"), nil
}

func (v *VBox) ImportVM(vmName string) error {
	_, sshPort, err := v.SSH.GenerateAddress()

	if err != nil {
		return err
	}

	ovaPath := filepath.Join(v.Config.OVADir, vmName+".ova")

	virtualSystemNumbers, err := v.Driver.GetVirtualSystemNumbersOfHardDiskImages(ovaPath)
	if err != nil {
		return err
	}

	importArguments := []string{
		"import",
		ovaPath,
		"--vsys", "0",
	}

	for i, number := range virtualSystemNumbers {
		importArguments = append(importArguments, "--unit")
		importArguments = append(importArguments, number)
		importArguments = append(importArguments, "--disk")
		importArguments = append(importArguments, filepath.Join(v.Config.PCFDevHome, fmt.Sprintf("%s-disk%d.vmdk", vmName, i)))
	}

	if _, err := v.Driver.VBoxManage(importArguments...); err != nil {
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
	if v.Config.DesiredMemory != 0 {
		memory = v.Config.DesiredMemory
	} else {
		freeMemory, err := v.System.FreeMemory()
		if err != nil {
			return uint64(0), err
		}
		if freeMemory <= v.Config.MinMemory {
			memory = v.Config.MinMemory
		} else if freeMemory >= v.Config.MaxMemory {
			memory = v.Config.MaxMemory
		} else {
			memory = freeMemory
		}
	}
	return memory, nil
}

func (v *VBox) DestroyVM(vmName string) error {
	return v.Driver.DestroyVM(vmName)
}

func (v *VBox) PowerOffVM(vmName string) error {
	return v.Driver.PowerOffVM(vmName)
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

func (v *VBox) SuspendVM(vmName string) error {
	return v.Driver.SuspendVM(vmName)
}

func (v *VBox) ResumeVM(vmName string) error {
	return v.Driver.ResumeVM(vmName)
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

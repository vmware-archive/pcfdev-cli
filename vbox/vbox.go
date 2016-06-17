package vbox

import (
	"errors"
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
	ConfigureHostOnlyInterface(interfaceName string, ip string) error
	AttachNetworkInterface(interfaceName string, vmName string) error
	ForwardPort(vmName string, ruleName string, hostPort string, guestPort string) error
	IsInterfaceInUse(interfaceName string) (bool, error)
	GetHostForwardPort(vmName string, ruleName string) (port string, err error)
	GetHostOnlyInterfaces() (interfaces []*network.Interface, err error)
	GetVMIP(vmName string) (vmIP string, err error)
	SetCPUs(vmName string, cpuNumber int) error
	SetMemory(vmName string, memory uint64) error
	CreateVM(vmName string, baseDirectory string) error
	AttachDisk(vmName string, diskPath string) error
	CloneDisk(src string, dest string) error
}

//go:generate mockgen -package mocks -destination mocks/fs.go github.com/pivotal-cf/pcfdev-cli/vbox FS
type FS interface {
	Extract(archivePath string, destinationPath string, filename string) error
	Remove(path string) error
}

//go:generate mockgen -package mocks -destination mocks/ssh.go github.com/pivotal-cf/pcfdev-cli/vbox SSH
type SSH interface {
	GenerateAddress() (host string, port string, err error)
	RunSSHCommand(command string, port string, timeout time.Duration, stdout io.Writer, stderr io.Writer) error
}

//go:generate mockgen -package mocks -destination mocks/picker.go github.com/pivotal-cf/pcfdev-cli/vbox NetworkPicker
type NetworkPicker interface {
	SelectAvailableIP(vboxnets []*network.Interface) (ip string, err error)
}

//go:generate mockgen -package mocks -destination mocks/address.go github.com/pivotal-cf/pcfdev-cli/vbox Address
type Address interface {
	DomainForIP(vmIP string) (domain string, err error)
	SubnetForIP(vmIP string) (subnetIP string, err error)
}

type VBox struct {
	Driver Driver
	SSH    SSH
	FS     FS
	Picker NetworkPicker
	Config *config.Config
}

const (
	StatusRunning    = "Running"
	StatusSuspended  = "Suspended"
	StatusStopped    = "Stopped"
	StatusNotCreated = "Not created"
)

func (v *VBox) StartVM(vmConfig *config.VMConfig) error {
	if err := v.Driver.StartVM(vmConfig.Name); err != nil {
		return err
	}

	if err := v.configureNetwork(vmConfig.IP, vmConfig.SSHPort); err != nil {
		return err
	}
	if err := v.configureEnvironment(vmConfig.IP, vmConfig.SSHPort); err != nil {
		return err
	}

	if err := v.Driver.StopVM(vmConfig.Name); err != nil {
		return err
	}

	return v.Driver.StartVM(vmConfig.Name)
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

func (v *VBox) ImportVM(vmConfig *config.VMConfig) error {
	if err := v.Driver.CreateVM(vmConfig.Name, v.Config.VMDir); err != nil {
		return err
	}

	compressedDisk := filepath.Join(v.Config.VMDir, vmConfig.DiskName) + ".compressed"
	uncompressedDisk := filepath.Join(v.Config.VMDir, vmConfig.Name, vmConfig.DiskName)
	if err := v.FS.Extract(filepath.Join(v.Config.OVADir, vmConfig.Name+".ova"), compressedDisk, vmConfig.DiskName); err != nil {
		return err
	}

	if err := v.Driver.CloneDisk(compressedDisk, uncompressedDisk); err != nil {
		return err
	}

	if err := v.FS.Remove(compressedDisk); err != nil {
		return err
	}

	if err := v.Driver.AttachDisk(vmConfig.Name, uncompressedDisk); err != nil {
		return err
	}

	vboxInterfaces, err := v.Driver.GetHostOnlyInterfaces()
	if err != nil {
		return err
	}

	ip, err := v.Picker.SelectAvailableIP(vboxInterfaces)
	if err != nil {
		return err
	}

	interfaceName := ""
	for _, iface := range vboxInterfaces {
		inUse, err := v.Driver.IsInterfaceInUse(iface.Name)
		if err != nil {
			return err
		}
		if !inUse {
			interfaceName = iface.Name
			break
		}
	}

	if interfaceName == "" {
		interfaceName, err = v.Driver.CreateHostOnlyInterface(ip)
		if err != nil {
			return err
		}
	} else {
		err = v.Driver.ConfigureHostOnlyInterface(interfaceName, ip)
		if err != nil {
			return err
		}
	}

	if err := v.Driver.AttachNetworkInterface(interfaceName, vmConfig.Name); err != nil {
		return err
	}

	_, sshPort, err := v.SSH.GenerateAddress()
	if err != nil {
		return err
	}

	if err := v.Driver.ForwardPort(vmConfig.Name, "ssh", sshPort, "22"); err != nil {
		return err
	}

	if err := v.Driver.SetCPUs(vmConfig.Name, vmConfig.CPUs); err != nil {
		return err
	}

	if err := v.Driver.SetMemory(vmConfig.Name, vmConfig.Memory); err != nil {
		return err
	}

	return nil
}

func (v *VBox) DestroyVM(vmConfig *config.VMConfig) error {
	return v.Driver.DestroyVM(vmConfig.Name)
}

func (v *VBox) PowerOffVM(vmConfig *config.VMConfig) error {
	return v.Driver.PowerOffVM(vmConfig.Name)
}

func (v *VBox) ConflictingVMPresent(vmConfig *config.VMConfig) (conflict bool, err error) {
	vms, err := v.Driver.RunningVMs()
	if err != nil {
		return false, err
	}

	for _, vm := range vms {
		if strings.HasPrefix(vm, "pcfdev-") && vm != vmConfig.Name {
			return true, nil
		}
	}
	return false, nil
}

func (v *VBox) StopVM(vmConfig *config.VMConfig) error {
	return v.Driver.StopVM(vmConfig.Name)
}

func (v *VBox) SuspendVM(vmConfig *config.VMConfig) error {
	return v.Driver.SuspendVM(vmConfig.Name)
}

func (v *VBox) ResumeVM(vmConfig *config.VMConfig) error {
	return v.Driver.ResumeVM(vmConfig.Name)
}

func (v *VBox) DestroyPCFDevVMs() error {
	vms, err := v.Driver.VMs()
	if err != nil {
		return err
	}

	for _, vm := range vms {
		if strings.HasPrefix(vm, "pcfdev-") {
			v.Driver.PowerOffVM(vm)
			v.Driver.DestroyVM(vm)
		}
	}

	vms, err = v.Driver.VMs()
	if err != nil {
		return err
	}

	for _, vm := range vms {
		if strings.HasPrefix(vm, "pcfdev-") {
			return errors.New("failed to destroy all pcfdev vms")
		}
	}

	return nil
}

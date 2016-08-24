package vbox

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"
	"text/template"
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
	Disks() (disks []string, err error)
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
	DeleteDisk(diskPath string) error
	UseDNSProxy(vmName string) error
	GetMemory(vmName string) (uint64, error)
	VMState(vmName string) (string, error)
	Version() (version *VBoxDriverVersion, err error)
}

//go:generate mockgen -package mocks -destination mocks/fs.go github.com/pivotal-cf/pcfdev-cli/vbox FS
type FS interface {
	Extract(archivePath string, destinationPath string, filename string) error
	Remove(path string) error
}

//go:generate mockgen -package mocks -destination mocks/ssh.go github.com/pivotal-cf/pcfdev-cli/vbox SSH
type SSH interface {
	GenerateAddress() (host string, port string, err error)
	RunSSHCommand(command string, ip string, port string, timeout time.Duration, stdout io.Writer, stderr io.Writer) error
}

//go:generate mockgen -package mocks -destination mocks/picker.go github.com/pivotal-cf/pcfdev-cli/vbox NetworkPicker
type NetworkPicker interface {
	SelectAvailableIP(vboxnets []*network.Interface) (ip string, err error)
}

type VBox struct {
	Config *config.Config
	Driver Driver
	FS     FS
	Picker NetworkPicker
	SSH    SSH
}

type VMProperties struct {
	IPAddress string
}

type ProxyTypes struct {
	HTTPProxy  string
	HTTPSProxy string
	NOProxy    string
}

const (
	StatusRunning    = "Running"
	StatusSaved      = "Saved"
	StatusPaused     = "Paused"
	StatusStopped    = "Stopped"
	StatusNotCreated = "Not created"
	StatusUnknown    = "Unknown"
)

var (
	networkTemplate = `
auto lo
iface lo inet loopback

auto eth0
iface eth0 inet dhcp

auto eth1
iface eth1 inet static
address {{.IPAddress}}
netmask 255.255.255.0`

	proxyTemplate = `
PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/games:/usr/local/games
{{if .HTTPProxy}}HTTP_PROXY={{.HTTPProxy}}{{end}}
{{if .HTTPSProxy}}HTTPS_PROXY={{.HTTPSProxy}}{{end}}
NO_PROXY={{.NOProxy}}
{{if .HTTPProxy}}http_proxy={{.HTTPProxy}}{{end}}
{{if .HTTPSProxy}}https_proxy={{.HTTPSProxy}}{{end}}
no_proxy={{.NOProxy}}`
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
	t, err := template.New("properties template").Parse(networkTemplate)
	if err != nil {
		return err
	}

	var sshCommand bytes.Buffer
	if err = t.Execute(&sshCommand, VMProperties{IPAddress: ip}); err != nil {
		return err
	}

	return v.SSH.RunSSHCommand(
		fmt.Sprintf("echo -e '%s' | sudo tee /etc/network/interfaces", sshCommand.String()),
		"127.0.0.1",
		sshPort,
		5*time.Minute,
		ioutil.Discard,
		ioutil.Discard,
	)
}

func (v *VBox) configureEnvironment(ip string, sshPort string) error {
	proxySettings, err := v.proxySettings(ip)
	if err != nil {
		return err
	}

	return v.SSH.RunSSHCommand(fmt.Sprintf("echo -e '%s' | sudo tee /etc/environment", proxySettings), "127.0.0.1", sshPort, 5*time.Minute, ioutil.Discard, ioutil.Discard)
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
		"." + domain,
	}, ",")
	if v.Config.NoProxy != "" {
		noProxy = strings.Join([]string{noProxy, v.Config.NoProxy}, ",")
	}

	t, err := template.New("proxy template").Parse(proxyTemplate)
	if err != nil {
		return "", err
	}

	var proxySettings bytes.Buffer
	if err = t.Execute(&proxySettings, ProxyTypes{HTTPProxy: httpProxy, HTTPSProxy: httpsProxy, NOProxy: noProxy}); err != nil {
		return "", err
	}

	return proxySettings.String(), nil
}

func (v *VBox) ImportVM(vmConfig *config.VMConfig) error {
	if err := v.Driver.CreateVM(vmConfig.Name, v.Config.VMDir); err != nil {
		return err
	}

	compressedDisk := filepath.Join(v.Config.VMDir, vmConfig.Name+"-disk1.vmdk") + ".compressed"
	uncompressedDisk := filepath.Join(v.Config.VMDir, vmConfig.Name, vmConfig.Name+"-disk1.vmdk")
	if err := v.FS.Extract(vmConfig.OVAPath, compressedDisk, `\w+\.vmdk`); err != nil {
		return err
	}

	if err := v.Driver.CloneDisk(compressedDisk, uncompressedDisk); err != nil {
		return err
	}

	if err := v.Driver.DeleteDisk(compressedDisk); err != nil {
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

	if err := v.Driver.UseDNSProxy(vmConfig.Name); err != nil {
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

func (v *VBox) GetVMName() (name string, err error) {
	vms, err := v.Driver.VMs()
	if err != nil {
		return "", err
	}
	for _, vm := range vms {
		if strings.HasPrefix(vm, "pcfdev-") {
			if name == "" {
				name = vm
			} else {
				return "", errors.New("multiple PCF Dev VMs found")
			}
		}
	}
	return name, nil
}

func (v *VBox) StopVM(vmConfig *config.VMConfig) error {
	return v.Driver.StopVM(vmConfig.Name)
}

func (v *VBox) SuspendVM(vmConfig *config.VMConfig) error {
	return v.Driver.SuspendVM(vmConfig.Name)
}

func (v *VBox) ResumePausedVM(vmConfig *config.VMConfig) error {
	return v.Driver.ResumeVM(vmConfig.Name)
}

func (v *VBox) ResumeSavedVM(vmConfig *config.VMConfig) error {
	return v.Driver.StartVM(vmConfig.Name)
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

	disks, err := v.Driver.Disks()
	if err != nil {
		return err
	}

	for _, disk := range disks {
		filename := filepath.Base(disk)
		if strings.HasPrefix(filename, "pcfdev-") {
			v.Driver.DeleteDisk(disk)
		}
	}

	disks, err = v.Driver.Disks()
	if err != nil {
		return err
	}

	for _, disk := range disks {
		filename := filepath.Base(disk)
		if strings.HasPrefix(filename, "pcfdev-") {
			return errors.New("failed to destroy all pcfdev disks")
		}
	}
	return nil
}

func (v *VBox) VMConfig(vmName string) (*config.VMConfig, error) {
	memory, err := v.Driver.GetMemory(vmName)
	if err != nil {
		return nil, err
	}
	port, err := v.Driver.GetHostForwardPort(vmName, "ssh")
	if err != nil {
		return nil, err
	}
	ip, err := v.Driver.GetVMIP(vmName)
	if err != nil {
		return nil, err
	}
	domain, err := address.DomainForIP(ip)
	if err != nil {
		return nil, err
	}

	return &config.VMConfig{
		Domain:  domain,
		IP:      ip,
		Memory:  memory,
		Name:    vmName,
		SSHPort: port,
	}, nil
}

func (v *VBox) VMStatus(vmName string) (status string, err error) {
	exists, err := v.Driver.VMExists(vmName)
	if err != nil {
		return "", err
	}

	if !exists {
		return StatusNotCreated, nil
	}

	state, err := v.Driver.VMState(vmName)
	if err != nil {
		return "", err
	}

	switch state {
	case StateRunning:
		return StatusRunning, nil
	case StateStopped, StateAborted:
		return StatusStopped, nil
	case StateSaved:
		return StatusSaved, nil
	case StatePaused:
		return StatusPaused, nil
	default:
		return StatusUnknown, nil
	}
}

func (v *VBox) Version() (version *VBoxDriverVersion, err error) {
	return v.Driver.Version()
}

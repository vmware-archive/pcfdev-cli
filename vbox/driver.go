package vbox

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pivotal-cf/pcfdev-cli/helpers"
	"github.com/pivotal-cf/pcfdev-cli/network"
)

type VBoxDriver struct{}

const (
	StateRunning = "running"
	StateSaved   = "saved"
	StateStopped = "poweroff"
)

func (*VBoxDriver) VBoxManage(arg ...string) (output []byte, err error) {
	vBoxManagePath, err := helpers.VBoxManagePath()
	if err != nil {
		return nil, errors.New("could not find VBoxManage executable")
	}

	output, err = exec.Command(vBoxManagePath, arg...).CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("failed to execute 'VBoxManage %s': %s: %s", strings.Join(arg, " "), err, output)
	}
	return output, nil
}

func (d *VBoxDriver) StartVM(vmName string) error {
	_, err := d.VBoxManage("startvm", vmName, "--type", "headless")
	return err
}

func (d *VBoxDriver) VMExists(vmName string) (exists bool, err error) {
	output, err := d.VBoxManage("list", "vms")
	if err != nil {
		return false, err
	}

	return strings.Contains(string(output), `"`+vmName+`"`), nil
}

func (d *VBoxDriver) VMState(vmName string) (string, error) {
	output, err := d.VBoxManage("showvminfo", vmName, "--machinereadable")
	if err != nil {
		return "", err
	}

	regex := regexp.MustCompile(`VMState="(.*)"`)
	matches := regex.FindStringSubmatch(string(output))
	if len(matches) <= 1 {
		return "", errors.New("no state identified for VM")
	}

	return matches[1], nil
}

func (d *VBoxDriver) StopVM(vmName string) error {
	if _, err := d.VBoxManage("controlvm", vmName, "acpipowerbutton"); err != nil {
		return err
	}

	for attempts := 0; attempts < 100; attempts++ {
		state, err := d.VMState(vmName)
		if err != nil {
			return err
		}
		if state == StateStopped {
			return nil
		}
		time.Sleep(time.Second)
	}
	return errors.New("timed out waiting for vm to stop")
}

func (d *VBoxDriver) SuspendVM(vmName string) error {
	if _, err := d.VBoxManage("controlvm", vmName, "savestate"); err != nil {
		return err
	}

	for attempts := 0; attempts < 100; attempts++ {
		state, err := d.VMState(vmName)
		if err != nil {
			return err
		}
		if state == StateSaved {
			return nil
		}
		time.Sleep(time.Second)
	}
	return errors.New("timed out waiting for vm to suspend")
}

func (d *VBoxDriver) ResumeVM(vmName string) error {
	return d.StartVM(vmName)
}

func (d *VBoxDriver) PowerOffVM(vmName string) error {
	_, err := d.VBoxManage("controlvm", vmName, "poweroff")
	return err
}

func (d *VBoxDriver) DestroyVM(vmName string) error {
	_, err := d.VBoxManage("unregistervm", vmName, "--delete")
	return err
}

func (d *VBoxDriver) CreateHostOnlyInterface(ip string) (interfaceName string, err error) {
	output, err := d.VBoxManage("hostonlyif", "create")
	if err != nil {
		return "", err
	}

	regex := regexp.MustCompile(`Interface '(.*)' was successfully created`)
	matches := regex.FindStringSubmatch(string(output))
	if len(matches) <= 1 {
		return "", errors.New("could not determine interface name")
	}

	interfaceName = matches[1]

	if _, err := d.VBoxManage("hostonlyif", "ipconfig", interfaceName, "--ip", ip, "--netmask", "255.255.255.0"); err != nil {
		return "", err
	}
	return interfaceName, nil
}

func (d *VBoxDriver) GetHostOnlyInterfaces() (interfaces []*network.Interface, err error) {
	output, err := d.VBoxManage("list", "hostonlyifs")
	if err != nil {
		return nil, err
	}

	nameRegex := regexp.MustCompile(`(?m:^Name:\s+(.*))`)
	nameMatches := nameRegex.FindAllStringSubmatch(string(output), -1)

	ipRegex := regexp.MustCompile(`(?m:^IPAddress:\s+(.*))`)
	ipMatches := ipRegex.FindAllStringSubmatch(string(output), -1)

	vboxnets := make([]*network.Interface, len(nameMatches))
	for i := 0; i < len(nameMatches); i++ {
		vboxnets[i] = &network.Interface{
			Name: strings.TrimSpace(nameMatches[i][1]),
			IP:   strings.TrimSpace(ipMatches[i][1]),
		}
	}

	return vboxnets, nil
}

func (d *VBoxDriver) GetMemory(vmName string) (uint64, error) {
	output, err := d.VBoxManage("showvminfo", vmName, "--machinereadable")
	if err != nil {
		return uint64(0), err
	}

	regex := regexp.MustCompile(`memory=(\d+)`)
	if matches := regex.FindStringSubmatch(string(output)); len(matches) > 1 {
		return strconv.ParseUint(matches[1], 10, 64)
	}

	return uint64(0), fmt.Errorf("failed to determine VM memory for '%s'", vmName)
}

func (d *VBoxDriver) SetMemory(vmName string, memory uint64) error {
	_, err := d.VBoxManage("modifyvm", vmName, "--memory", strconv.Itoa(int(memory)))
	return err
}

func (d *VBoxDriver) GetVMIP(vmName string) (string, error) {
	vboxnetName, err := d.getVBoxNetName(vmName)
	if err != nil {
		return "", err
	}
	if vboxnetName == "" {
		return "", fmt.Errorf("there is no attached hostonlyif for %s", vmName)
	}

	vboxnets, err := d.GetHostOnlyInterfaces()
	if err != nil {
		return "", err
	}

	for _, vboxnet := range vboxnets {
		if vboxnet.Name == vboxnetName {
			return d.getVMIPForSubnet(vboxnet.IP), nil
		}
	}

	return "", fmt.Errorf("couldnt find %s in list of hostonlyifs", vboxnetName)
}

func (d *VBoxDriver) AttachNetworkInterface(interfaceName string, vmName string) error {
	_, err := d.VBoxManage("modifyvm", vmName, "--nic2", "hostonly", "--hostonlyadapter2", interfaceName)
	return err
}

func (d *VBoxDriver) ForwardPort(vmName string, ruleName string, hostPort string, guestPort string) error {
	_, err := d.VBoxManage("modifyvm", vmName, "--natpf1", fmt.Sprintf("%s,tcp,127.0.0.1,%s,,%s", ruleName, hostPort, guestPort))
	return err
}

func (d *VBoxDriver) GetHostForwardPort(vmName string, ruleName string) (port string, err error) {
	output, err := d.VBoxManage("showvminfo", vmName, "--machinereadable")
	if err != nil {
		return "", err
	}

	regex := regexp.MustCompile(`Forwarding\(\d+\)="` + ruleName + `,tcp,127.0.0.1,(.*),,22"`)
	if matches := regex.FindStringSubmatch(string(output)); len(matches) > 1 {
		return matches[1], nil
	}

	return "", errors.New("could not find forwarded port")
}

func (d *VBoxDriver) VMs() ([]string, error) {
	output, err := d.VBoxManage("list", "vms")
	if err != nil {
		return nil, err
	}

	vms := []string{}
	for _, line := range strings.Split(strings.Trim(string(output), "\n"), "\n") {
		regex := regexp.MustCompile(`^"(.+)"\s`)
		if matches := regex.FindStringSubmatch(string(line)); len(matches) > 1 {
			vms = append(vms, matches[1])
		}
	}

	return vms, nil
}

func (d *VBoxDriver) GetVirtualSystemNumbersOfHardDiskImages(ovaPath string) (virtualSystemNumbers []string, err error) {
	output, err := d.VBoxManage("import", ovaPath, "-n")
	if err != nil {
		return nil, err
	}

	numbers := []string{}
	for _, line := range strings.Split(strings.Trim(string(output), "\n"), "\n") {
		regex := regexp.MustCompile(`(\d+):\s{1}Hard disk image:`)
		if matches := regex.FindStringSubmatch(string(line)); len(matches) > 1 {
			numbers = append(numbers, matches[1])
		}
	}

	if len(numbers) == 0 {
		return nil, fmt.Errorf("could not determine hard disk image virtual system numbers of '%s': %s", ovaPath, err)
	}

	return numbers, nil

}

func (d *VBoxDriver) RunningVMs() (vms []string, err error) {
	output, err := d.VBoxManage("list", "runningvms")
	if err != nil {
		return nil, err
	}

	runningVMs := []string{}
	for _, line := range strings.Split(strings.Trim(string(output), "\n"), "\n") {
		regex := regexp.MustCompile(`^"(.+)"\s`)
		if matches := regex.FindStringSubmatch(string(line)); len(matches) > 1 {
			runningVMs = append(runningVMs, matches[1])
		}
	}

	return runningVMs, nil
}

func (d *VBoxDriver) getVMIPForSubnet(subnetIP string) string {
	return subnetIP + "1"
}

func (d *VBoxDriver) getVBoxNetName(vmName string) (interfaceName string, err error) {
	output, err := d.VBoxManage("showvminfo", vmName, "--machinereadable")
	if err != nil {
		return "", err
	}

	regex := regexp.MustCompile(`hostonlyadapter2="(.*)"`)
	if matches := regex.FindStringSubmatch(string(output)); len(matches) > 1 {
		return matches[1], nil
	}

	return "", nil
}

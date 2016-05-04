package vbox

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

type VBoxDriver struct{}

func (*VBoxDriver) VBoxManage(arg ...string) (output []byte, err error) {
	output, err = exec.Command("VBoxManage", arg...).Output()
	if err != nil {
		return output, fmt.Errorf("failed to execute 'VBoxManage %s': %s", strings.Join(arg, " "), err)
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

func (d *VBoxDriver) IsVMRunning(vmName string) bool {
	vmStatus, err := d.VBoxManage("showvminfo", vmName, "--machinereadable")
	if err != nil {
		return false
	}
	return strings.Contains(string(vmStatus), `VMState="running"`)
}

func (d *VBoxDriver) StopVM(vmName string) error {
	if _, err := d.VBoxManage("controlvm", vmName, "acpipowerbutton"); err != nil {
		return err
	}
	for attempts := 0; attempts < 100; attempts++ {
		if !d.IsVMRunning(vmName) {
			return nil
		}
		time.Sleep(time.Second)
	}
	return errors.New("timed out waiting for vm to stop")
}

func (d *VBoxDriver) PowerOffVM(vmName string) error {
	_, err := d.VBoxManage("controlvm", vmName, "poweroff")
	return err
}

func (d *VBoxDriver) DestroyVM(vmName string) error {
	interfaceName, err := d.getVBoxNetName(vmName)
	if err != nil {
		return err
	}

	if _, err := d.VBoxManage("unregistervm", vmName, "--delete"); err != nil {
		return err
	}

	if interfaceName != "" {
		if _, err := d.VBoxManage("hostonlyif", "remove", interfaceName); err != nil {
			return err
		}
	}

	return nil
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

	_, err = d.VBoxManage("hostonlyif", "ipconfig", interfaceName, "--ip", ip, "--netmask", "255.255.255.0")
	if err != nil {
		return "", err
	}
	return interfaceName, nil
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
		return []string{}, err
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

func (d *VBoxDriver) RunningVMs() (vms []string, err error) {
	output, err := d.VBoxManage("list", "runningvms")
	if err != nil {
		return []string{}, err
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

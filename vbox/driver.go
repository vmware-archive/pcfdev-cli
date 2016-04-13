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

func (*VBoxDriver) VBoxManage(arg ...string) ([]byte, error) {
	return exec.Command("VBoxManage", arg...).Output()
}

func (d *VBoxDriver) StartVM(name string) error {
	_, err := d.VBoxManage("startvm", name, "--type", "headless")
	if err != nil {
		return fmt.Errorf("failed to execute 'VBoxManage startvm %s':%s", name, err)
	}
	return nil
}

func (d *VBoxDriver) VMExists(name string) (bool, error) {
	output, err := d.VBoxManage("list", "vms")

	if err != nil {
		return false, fmt.Errorf("failed to execute 'VBoxManage list vms: %s", err)
	}

	return strings.Contains(string(output), `"`+name+`"`), nil
}

func (d *VBoxDriver) StopVM(name string) error {
	var err error
	_, err = d.VBoxManage("controlvm", name, "acpipowerbutton")
	if err != nil {
		return fmt.Errorf("failed to execute 'VBoxManage controlvm some-bad-vm-name acipipowerbutton':%s", name, err)
	}
	for attempts := 0; attempts < 100; attempts++ {
		if !d.IsVMRunning(name) {
			return nil
		}
		time.Sleep(time.Second)
	}
	return errors.New("timed out waiting for vm to stop")
}

func (d *VBoxDriver) ForwardPort(vmName string, ruleName string, guestPort string, hostPort string) error {
	var err error
	_, err = d.VBoxManage("modifyvm", vmName, "--natpf1", fmt.Sprintf("%s,tcp,127.0.0.1,%s,,%s", ruleName, hostPort, guestPort))
	if err != nil {
		return fmt.Errorf("failed to forward guest port %s to host port %s:%s", guestPort, hostPort, err)
	}
	return nil
}

func (d *VBoxDriver) GetHostForwardPort(vmName string, ruleName string) (string, error) {
	output, err := d.VBoxManage("showvminfo", vmName, "--machinereadable")
	if err != nil {
		return "", fmt.Errorf("failed to execute 'VBoxManage showvminfo %s --machinereadable': %s", vmName, err)
	}

	regex := regexp.MustCompile(`Forwarding\(\d+\)="` + ruleName + `,tcp,127.0.0.1,(.*),,22"`)
	return regex.FindStringSubmatch(string(output))[1], nil
}

func (d *VBoxDriver) CreateHostOnlyInterface(ip string) (string, error) {
	var err error
	output, err := d.VBoxManage("hostonlyif", "create")
	if err != nil {
		return "", fmt.Errorf("failed to create hostonlyif:%s", err)
	}
	regex := regexp.MustCompile(`Interface '(.*)' was successfully created`)
	name := regex.FindStringSubmatch(string(output))[1]

	_, err = d.VBoxManage("hostonlyif", "ipconfig", name, "--ip", ip, "--netmask", "255.255.255.0")
	if err != nil {
		return "", fmt.Errorf("failed to configure hostonlyif:%s", err)
	}
	return name, nil
}

func (d *VBoxDriver) AttachNetworkInterface(vboxnet string, vmName string) error {
	_, err := d.VBoxManage("modifyvm", vmName, "--nic2", "hostonly", "--hostonlyadapter2", vboxnet)
	if err != nil {
		return fmt.Errorf("failed to attach %s interface to vm %s: %s", vboxnet, vmName, err)
	}
	return nil
}

func (d *VBoxDriver) IsVMRunning(name string) bool {
	vmStatus, err := d.VBoxManage("showvminfo", name, "--machinereadable")
	if err != nil {
		return false
	}
	return strings.Contains(string(vmStatus), `VMState="running"`)
}

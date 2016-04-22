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
	output, err := exec.Command("VBoxManage", arg...).Output()
	if err != nil {
		return output, fmt.Errorf("failed to execute 'VBoxManage %s': %s", strings.Join(arg, " "), err)
	}
	return output, nil
}

func (d *VBoxDriver) StartVM(name string) error {
	_, err := d.VBoxManage("startvm", name, "--type", "headless")
	return err
}

func (d *VBoxDriver) VMExists(name string) (bool, error) {
	output, err := d.VBoxManage("list", "vms")

	if err != nil {
		return false, err
	}

	return strings.Contains(string(output), `"`+name+`"`), nil
}

func (d *VBoxDriver) StopVM(name string) error {
	_, err := d.VBoxManage("controlvm", name, "acpipowerbutton")
	if err != nil {
		return err
	}
	for attempts := 0; attempts < 100; attempts++ {
		if !d.IsVMRunning(name) {
			return nil
		}
		time.Sleep(time.Second)
	}
	return errors.New("timed out waiting for vm to stop")
}

func (d *VBoxDriver) DestroyVM(name string) error {
	vboxnet, err := d.GetVBoxNetName(name)
	if err != nil {
		return err
	}

	_, err = d.VBoxManage("unregistervm", name, "--delete")
	if err != nil {
		return err
	}

	if vboxnet != "" {
		err = d.DestroyHostOnlyInterface(vboxnet)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *VBoxDriver) ForwardPort(vmName string, ruleName string, guestPort string, hostPort string) error {
	_, err := d.VBoxManage("modifyvm", vmName, "--natpf1", fmt.Sprintf("%s,tcp,127.0.0.1,%s,,%s", ruleName, hostPort, guestPort))
	return err
}

func (d *VBoxDriver) GetHostForwardPort(vmName string, ruleName string) (string, error) {
	output, err := d.VBoxManage("showvminfo", vmName, "--machinereadable")
	if err != nil {
		return "", err
	}

	regex := regexp.MustCompile(`Forwarding\(\d+\)="` + ruleName + `,tcp,127.0.0.1,(.*),,22"`)
	return regex.FindStringSubmatch(string(output))[1], nil
}

func (d *VBoxDriver) CreateHostOnlyInterface(ip string) (string, error) {
	output, err := d.VBoxManage("hostonlyif", "create")
	if err != nil {
		return "", err
	}
	regex := regexp.MustCompile(`Interface '(.*)' was successfully created`)
	name := regex.FindStringSubmatch(string(output))[1]

	_, err = d.VBoxManage("hostonlyif", "ipconfig", name, "--ip", ip, "--netmask", "255.255.255.0")
	if err != nil {
		return "", err
	}
	return name, nil
}

func (d *VBoxDriver) AttachNetworkInterface(vboxnet string, vmName string) error {
	_, err := d.VBoxManage("modifyvm", vmName, "--nic2", "hostonly", "--hostonlyadapter2", vboxnet)
	return err
}

func (d *VBoxDriver) IsVMRunning(name string) bool {
	vmStatus, err := d.VBoxManage("showvminfo", name, "--machinereadable")
	if err != nil {
		return false
	}
	return strings.Contains(string(vmStatus), `VMState="running"`)
}

func (d *VBoxDriver) DestroyHostOnlyInterface(name string) error {
	_, err := d.VBoxManage("hostonlyif", "remove", name)
	return err
}

func (d *VBoxDriver) GetVBoxNetName(name string) (string, error) {
	output, err := d.VBoxManage("showvminfo", name, "--machinereadable")
	if err != nil {
		return "", err
	}

	regex := regexp.MustCompile(`hostonlyadapter2="(.*)"`)
	matches := regex.FindStringSubmatch(string(output))
	if len(matches) > 1 {
		return matches[1], nil
	} else {
		return "", nil
	}
}

package vbox

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pivotal-cf/pcfdev-cli/fs"
	"github.com/pivotal-cf/pcfdev-cli/helpers"
	"github.com/pivotal-cf/pcfdev-cli/network"
	"github.com/pivotal-cf/pcfdev-cli/runner"
)

type VBoxDriverVersion struct {
	Major, Minor, Build int
}

type VBoxDriver struct {
	FS        *fs.FS
	CmdRunner *runner.CmdRunner
}

const (
	StateRunning = "running"
	StateSaved   = "saved"
	StateStopped = "poweroff"
	StateAborted = "aborted"
	StatePaused  = "paused"
)

func (v *VBoxDriver) VBoxManage(arg ...string) (output []byte, err error) {
	vBoxManagePath, err := helpers.VBoxManagePath()
	if err != nil {
		return nil, errors.New("could not find VBoxManage executable")
	}

	return v.CmdRunner.Run(vBoxManagePath, arg...)
}

func (d *VBoxDriver) StartVM(vmName string) error {
	_, err := d.VBoxManage("startvm", vmName, "--type", "headless")
	return err
}

func (d *VBoxDriver) CreateVM(vmName string, basedir string) error {
	if _, err := d.VBoxManage("createvm", "--name", vmName, "--ostype", "Ubuntu_64", "--basefolder", basedir, "--register"); err != nil {
		return err
	}
	if _, err := d.VBoxManage("modifyvm", vmName, "--paravirtprovider", "minimal"); err != nil {
		return err
	}
	_, err := d.VBoxManage("modifyvm", vmName, "--nic1", "nat", "--nictype1", "virtio")
	return err
}

func (d *VBoxDriver) UseDNSProxy(vmName string) error {
	_, err := d.VBoxManage("modifyvm", vmName, "--natdnshostresolver1", "on")
	return err
}

func (d *VBoxDriver) AttachDisk(vmName string, diskPath string) error {
	if _, err := d.VBoxManage("storagectl", vmName, "--name", "SATA", "--add", "sata"); err != nil {
		return err
	}
	if _, err := d.VBoxManage("storageattach", vmName, "--storagectl", "SATA", "--medium", diskPath, "--type", "hdd", "--port", "0", "--device", "0"); err != nil {
		return err
	}
	return nil
}

func (d *VBoxDriver) VMExists(vmName string) (exists bool, err error) {
	output, err := d.VBoxManage("list", "vms")
	if err != nil {
		return false, err
	}

	return strings.Contains(string(output), `"`+vmName+`"`), nil
}

func (d *VBoxDriver) VMState(vmName string) (string, error) {
	var output []byte
	err := helpers.ExecuteWithAttempts(func() error {
		var err error
		output, err = d.VBoxManage("showvminfo", vmName, "--machinereadable")
		return err
	}, 3, time.Second)

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

	return helpers.ExecuteWithTimeout(func() error {
		state, err := d.VMState(vmName)
		if err != nil {
			return fmt.Errorf("timed out waiting for vm to stop: %s", err)
		}
		if state != StateStopped {
			return fmt.Errorf("timed out waiting for vm to stop")
		}
		return nil
	},
		time.Minute,
		time.Second,
	)
}

func (d *VBoxDriver) SuspendVM(vmName string) error {
	if _, err := d.VBoxManage("controlvm", vmName, "savestate"); err != nil {
		return err
	}

	return helpers.ExecuteWithTimeout(func() error {
		state, err := d.VMState(vmName)
		if err != nil {
			return fmt.Errorf("timed out waiting for vm to suspend: %s", err)
		}
		if state != StateSaved {
			return fmt.Errorf("timed out waiting for vm to suspend")
		}
		return nil
	},
		2*time.Minute,
		time.Second,
	)
}

func (d *VBoxDriver) ResumeVM(vmName string) error {
	_, err := d.VBoxManage("controlvm", vmName, "resume")
	return err
}

func (d *VBoxDriver) PowerOffVM(vmName string) error {
	_, err := d.VBoxManage("controlvm", vmName, "poweroff")
	return err
}

func (d *VBoxDriver) DestroyVM(vmName string) error {
	return helpers.ExecuteWithTimeout(func() error {
		if _, err := d.VBoxManage("unregistervm", vmName, "--delete"); err != nil {
			return fmt.Errorf("timed out waiting for vm to destroy: %s", err)
		}
		return nil
	},
		time.Minute,
		time.Second,
	)
}

func (d *VBoxDriver) CreateHostOnlyInterface(ip string) (string, error) {
	var interfaceName string
	err := helpers.ExecuteWithAttempts(func() error {
		output, err := d.VBoxManage("hostonlyif", "create")
		if err != nil {
			return err
		}

		regex := regexp.MustCompile(`Interface '(.*)' was successfully created`)
		matches := regex.FindStringSubmatch(string(output))
		if len(matches) <= 1 {
			return errors.New("could not determine interface name")
		}

		interfaceName = matches[1]
		return nil
	}, 3, time.Second)

	if err != nil {
		return "", err
	}

	if _, err := d.VBoxManage("hostonlyif", "ipconfig", interfaceName, "--ip", ip, "--netmask", "255.255.255.0"); err != nil {
		return "", err
	}
	return interfaceName, nil
}

func (d *VBoxDriver) ConfigureHostOnlyInterface(interfaceName string, ip string) error {
	if _, err := d.VBoxManage("hostonlyif", "ipconfig", interfaceName, "--ip", ip); err != nil {
		return err
	}

	return nil
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

	hardwareAddressRegex := regexp.MustCompile(`(?m:^HardwareAddress:\s+(.*))`)
	hardwareAddressMatches := hardwareAddressRegex.FindAllStringSubmatch(string(output), -1)

	vboxnets := make([]*network.Interface, len(nameMatches))
	for i := 0; i < len(nameMatches); i++ {
		vboxnets[i] = &network.Interface{
			Name:            strings.TrimSpace(nameMatches[i][1]),
			IP:              strings.TrimSpace(ipMatches[i][1]),
			HardwareAddress: strings.TrimSpace(hardwareAddressMatches[i][1]),
			Exists:          true,
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

func (d *VBoxDriver) SetCPUs(vmName string, cpus int) error {
	_, err := d.VBoxManage("modifyvm", vmName, "--cpus", strconv.Itoa(cpus))
	return err
}

func (d *VBoxDriver) AttachNetworkInterface(interfaceName string, vmName string) error {
	_, err := d.VBoxManage("modifyvm", vmName, "--nic2", "hostonly", "--nictype2", "virtio", "--hostonlyadapter2", interfaceName)
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

func (d *VBoxDriver) IsInterfaceInUse(interfaceName string) (bool, error) {
	output, err := d.VBoxManage("list", "vms", "--long")
	if err != nil {
		return false, err
	}

	regex := regexp.MustCompile(`NIC\s.*Attachment: Host-only Interface '(` + interfaceName + `)'`)
	if matches := regex.FindStringSubmatch(string(output)); len(matches) > 1 {
		return true, nil
	}

	return false, nil
}

func (d *VBoxDriver) CloneDisk(src, dst string) error {
	if _, err := d.VBoxManage("clonemedium", "disk", src, dst); err != nil {
		return err
	}
	if _, err := d.VBoxManage("closemedium", "disk", src); err != nil {
		return err
	}
	return nil
}

func (d *VBoxDriver) DeleteDisk(diskPath string) error {
	exists, err := d.FS.Exists(diskPath)
	if err != nil {
		return err
	}

	var args []string
	if exists {
		args = []string{"closemedium", "disk", diskPath, "--delete"}
	} else {
		args = []string{"closemedium", "disk", diskPath}
	}

	if _, err := d.VBoxManage(args...); err != nil {
		return err
	}

	return nil
}

func (d *VBoxDriver) Disks() ([]string, error) {
	output, err := d.VBoxManage("list", "hdds")
	if err != nil {
		return nil, err
	}

	disks := []string{}
	for _, line := range strings.Split(strings.Trim(string(output), "\n"), "\n") {
		regex := regexp.MustCompile(`^Location:\s+(.+)`)
		if matches := regex.FindStringSubmatch(strings.TrimSpace(string(line))); len(matches) > 1 {
			disks = append(disks, matches[1])
		}
	}

	return disks, nil
}

func (d *VBoxDriver) Version() (*VBoxDriverVersion, error) {
	output, err := d.VBoxManage("--version")
	if err != nil {
		return nil, err
	}

	regex := regexp.MustCompile(`^(\d+\.\d+\.\d+)\D`)
	if matches := regex.FindStringSubmatch(strings.TrimSpace(string(output))); len(matches) > 1 {
		if versionParts := strings.SplitN(matches[1], ".", 3); len(versionParts) == 3 {
			majorVersion, errMajor := strconv.Atoi(versionParts[0])
			minorVersion, errMinor := strconv.Atoi(versionParts[1])
			buildVersion, errBuild := strconv.Atoi(versionParts[2])

			if errMajor != nil || errMinor != nil || errBuild != nil {
				return nil, fmt.Errorf("failed to parse version from 'VBoxManage --version': %s", string(output))
			}

			return &VBoxDriverVersion{
				Major: majorVersion,
				Minor: minorVersion,
				Build: buildVersion,
			}, nil
		}
	}

	return nil, fmt.Errorf("failed to parse version from 'VBoxManage --version': %s", string(output))
}

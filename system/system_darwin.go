package system

import (
	"os/exec"
	"strconv"
	"strings"
)

func (s *System) PhysicalCores() (int, error) {
	sysctlPath, err := exec.LookPath("sysctl")
	if err != nil {
		sysctlPath = "/usr/sbin/sysctl"
	}
	output, err := exec.Command(sysctlPath, "-n", "hw.physicalcpu").Output()
	if err != nil {
		return 0, nil
	}
	return strconv.Atoi(strings.TrimSpace(string(output)))
}

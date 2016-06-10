package system

import (
	"errors"
	"os/exec"
	"regexp"
	"strconv"
)

func (s *System) PhysicalCores() (int, error) {
	output, err := exec.Command("wmic", "computersystem", "get", "numberofprocessors").Output()
	if err != nil {
		return 0, err
	}
	regex := regexp.MustCompile(`NumberOfProcessors\s+(\d+)`)
	matches := regex.FindStringSubmatch(string(output))
	if len(matches) <= 1 {
		return 0, errors.New("could not determine number of cores")
	}
	cores, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, err
	}
	return cores, nil
}

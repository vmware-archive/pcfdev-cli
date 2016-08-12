package system

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

func (s *System) PhysicalCores() (int, error) {
	output, err := exec.Command("wmic", "computersystem", "get", "numberofprocessors").CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "Access is denied") {
			fmt.Println(`Warning: unable to determine the number of CPU cores due to restrictive user account settings. If not specified manually with "-c", the PCF Dev VM will start with a single core.`)
			return 1, nil
		}
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

// +build windows

package requirements

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
)

func (m *Memory) freeMemory() (int, error) {
	freeOutput, err := exec.Command("wmic", "OS", "get", "FreePhysicalMemory").Output()
	if err != nil {
		return 0, fmt.Errorf("could not get memory stats: %s", err)
	}
	regex := regexp.MustCompile(`FreePhysicalMemory\s+([\d]+)`)
	matches := regex.FindStringSubmatch(string(freeOutput))
	if len(matches) < 2 {
		return 0, errors.New("FreePhysicalMemory output did not match expected format")
	}
	freeKB, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, err
	}
	freeMB := freeKB / 1024
	return freeMB, nil
}

// +build linux

package requirements

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
)

func (m *Memory) freeMemory() (int, error) {
	freeOutput, err := exec.Command("free", "-m").Output()
	if err != nil {
		return 0, fmt.Errorf("could not get memory stats: %s", err)
	}

	regex := regexp.MustCompile(`Mem:\s+\d+\s+\d+\s+(\d+)\s+\d+\s+(\d+)\s+(\d+)`)
	matches := regex.FindStringSubmatch(string(freeOutput))
	if len(matches) < 4 {
		return 0, errors.New("`free` output did not match expected format")
	}

	freeMemory, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, err
	}

	buffCacheMemory, err := strconv.Atoi(matches[2])
	if err != nil {
		return 0, err
	}

	availableMemory, err := strconv.Atoi(matches[3])
	if err != nil {
		return 0, err
	}

	return freeMemory + buffCacheMemory + availableMemory, nil
}

// +build linux

package requirements

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

func (m *Memory) freeMemory() (int, error) {
	freeOutput, err := exec.Command("free", "-m").Output()
	if err != nil {
		return 0, fmt.Errorf("could not get memory stats: %s", err)
	}

	memFields := strings.Fields(strings.Split(string(freeOutput), "\n")[1])

	freeMemory, err := strconv.Atoi(memFields[3])
	if err != nil {
		return 0, err
	}

	buffCacheMemory, err := strconv.Atoi(memFields[5])
	if err != nil {
		return 0, err
	}

	availableMemory, err := strconv.Atoi(memFields[6])
	if err != nil {
		return 0, err
	}

	return freeMemory + buffCacheMemory + availableMemory, nil
}

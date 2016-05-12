// +build darwin

package requirements

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
)

const (
	PAGE_SIZE_IN_BYTES = 4096
	BYTES_IN_MEGABYTE  = 1048576
)

func (m *Memory) freeMemory() (int, error) {
	freePages, err := pagesFromVMStat("free")
	if err != nil {
		return 0, err
	}

	inactivePages, err := pagesFromVMStat("inactive")
	if err != nil {
		return 0, err
	}

	speculativePages, err := pagesFromVMStat("speculative")
	if err != nil {
		return 0, err
	}

	return (freePages + inactivePages + speculativePages) * PAGE_SIZE_IN_BYTES / BYTES_IN_MEGABYTE, nil
}

func pagesFromVMStat(pageType string) (int, error) {
	vmStatOutput, err := exec.Command("vm_stat").Output()
	if err != nil {
		return 0, fmt.Errorf("could not get memory stats: %s", err)
	}

	regex := regexp.MustCompile(fmt.Sprintf(`Pages %s: +([0-9]+).`, pageType))
	matches := regex.FindStringSubmatch(string(vmStatOutput))
	if len(matches) <= 1 {
		return 0, errors.New("could not determine number of %s pages")
	}

	return strconv.Atoi(matches[1])
}

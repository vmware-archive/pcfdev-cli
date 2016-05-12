package requirements

import (
	"fmt"
	"strconv"
)

type Memory struct {
	MinimumFreeMemory int
}

func (m *Memory) Check() error {
	freeMemory, err := m.freeMemory()
	if err != nil {
		panic(err)
	}
	if freeMemory < m.MinimumFreeMemory {
		return fmt.Errorf(
			"PCF Dev requires %sMB of free memory. This host machine has %sMB free.",
			numberWithCommas(m.MinimumFreeMemory),
			numberWithCommas(freeMemory))
	}
	return nil
}

func numberWithCommas(num int) string {
	str := ""
	for num/1000 >= 1 {
		str = fmt.Sprintf(",%03s%s", strconv.Itoa(num%1000), str)
		num /= 1000
	}
	return strconv.Itoa(num) + str
}

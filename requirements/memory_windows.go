// +build windows

package requirements

import "errors"

func (m *Memory) freeMemory() (int, error) {
	return 0, errors.New("getting free memory on windows is not yet implemented")
}

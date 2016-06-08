package system

import "github.com/cloudfoundry/gosigar"

type System struct{}

func (s *System) FreeMemory() (uint64, error) {
	mem := &sigar.Mem{}
	err := mem.Get()
	if err != nil {
		return 0, err
	}
	return mem.ActualFree, nil
}

func (s *System) TotalMemory() (uint64, error) {
	mem := &sigar.Mem{}
	err := mem.Get()
	if err != nil {
		return 0, err
	}
	return mem.Total, nil
}

package system

import "github.com/cloudfoundry/gosigar"

const BYTES_IN_MEGABYTE = 1048576

//go:generate mockgen -package mocks -destination mocks/fs.go github.com/pivotal-cf/pcfdev-cli/system FS
type FS interface {
	Read(path string) ([]byte, error)
}

type System struct {
	FS FS
}

func (s *System) FreeMemory() (uint64, error) {
	mem := &sigar.Mem{}
	err := mem.Get()
	if err != nil {
		return 0, err
	}
	return mem.ActualFree / BYTES_IN_MEGABYTE, nil
}

func (s *System) TotalMemory() (uint64, error) {
	mem := &sigar.Mem{}
	err := mem.Get()
	if err != nil {
		return 0, err
	}
	return mem.Total / BYTES_IN_MEGABYTE, nil
}

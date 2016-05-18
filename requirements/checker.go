package requirements

import "fmt"

type Checker struct {
	System System
	Config Config
}

const BYTES_IN_MEGABYTE = 1048576

//go:generate mockgen -package mocks -destination mocks/system.go github.com/pivotal-cf/pcfdev-cli/requirements System
type System interface {
	FreeMemory() (freeBytes uint64, err error)
}

//go:generate mockgen -package mocks -destination mocks/config.go github.com/pivotal-cf/pcfdev-cli/requirements Config
type Config interface {
	GetMinMemory() (minMB uint64)
}

func (c *Checker) Check() error {
	return c.checkMemory()
}

func (c *Checker) checkMemory() error {
	freeBytes, err := c.System.FreeMemory()
	if err != nil {
		return err
	}
	minMB := c.Config.GetMinMemory()
	minBytes := minMB * BYTES_IN_MEGABYTE
	if freeBytes < minBytes {
		return fmt.Errorf("PCF Dev requires %dMB of free memory, this host has %dMB", minMB, (freeBytes / BYTES_IN_MEGABYTE))
	}
	return nil
}

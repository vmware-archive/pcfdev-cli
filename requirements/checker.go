package requirements

import "github.com/pivotal-cf/pcfdev-cli/config"

type Checker struct {
	System System
	Config *config.Config
}

const BYTES_IN_MEGABYTE = 1048576

//go:generate mockgen -package mocks -destination mocks/system.go github.com/pivotal-cf/pcfdev-cli/requirements System
type System interface {
	FreeMemory() (freeBytes uint64, err error)
}

func (c *Checker) CheckMemory(desiredMemory uint64) error {
	if desiredMemory < c.Config.MinMemory {
		return &RequestedMemoryTooLittleError{
			DesiredMemory: desiredMemory,
			MinMemory:     c.Config.MinMemory,
		}
	}

	freeBytes, err := c.System.FreeMemory()
	if err != nil {
		return err
	}

	if freeBytes < desiredMemory*BYTES_IN_MEGABYTE {
		return &NotEnoughMemoryError{
			FreeMemory:    (freeBytes / BYTES_IN_MEGABYTE),
			DesiredMemory: desiredMemory,
		}
	}
	return nil
}

func (c *Checker) CheckMinMemory() error {
	return c.CheckMemory(c.Config.MinMemory)
}

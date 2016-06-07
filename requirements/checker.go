package requirements

import (
	"fmt"

	"github.com/pivotal-cf/pcfdev-cli/config"
)

type Checker struct {
	System System
	Config *config.Config
}

const BYTES_IN_MEGABYTE = 1048576

//go:generate mockgen -package mocks -destination mocks/system.go github.com/pivotal-cf/pcfdev-cli/requirements System
type System interface {
	FreeMemory() (freeBytes uint64, err error)
}

func (c *Checker) Check(desiredMemory uint64) error {
	return c.checkMemory(desiredMemory)
}

func (c *Checker) checkMemory(desiredMemory uint64) error {
	if desiredMemory < c.Config.MinMemory {
		return fmt.Errorf("PCF Dev requires at least %dMB of memory to run.", c.Config.MinMemory)
	}

	freeBytes, err := c.System.FreeMemory()
	if err != nil {
		return err
	}

	if freeBytes < desiredMemory*BYTES_IN_MEGABYTE {
		return fmt.Errorf("PCF Dev requires %dMB of free memory, this host has %dMB", c.Config.MinMemory, (freeBytes / BYTES_IN_MEGABYTE))
	}
	return nil
}

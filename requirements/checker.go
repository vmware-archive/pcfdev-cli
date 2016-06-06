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

func (c *Checker) Check() error {
	return c.checkMemory()
}

func (c *Checker) checkMemory() error {
	freeBytes, err := c.System.FreeMemory()
	if err != nil {
		return err
	}
	minBytes := c.Config.MinMemory * BYTES_IN_MEGABYTE
	if freeBytes < minBytes {
		return fmt.Errorf("PCF Dev requires %dMB of free memory, this host has %dMB", c.Config.MinMemory, (freeBytes / BYTES_IN_MEGABYTE))
	}
	return nil
}

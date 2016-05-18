package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	UI        UI
	MinMemory uint64
	MaxMemory uint64
	token     string
}

//go:generate mockgen -package mocks -destination mocks/ui.go github.com/pivotal-cf/pcfdev-cli/config UI
type UI interface {
	AskForPassword(string, ...interface{}) string
	Say(string, ...interface{})
}

func (c *Config) GetToken() string {
	if c.token != "" {
		return c.token
	}

	if envToken := os.Getenv("PIVNET_TOKEN"); envToken != "" {
		return envToken
	}

	c.UI.Say("Please retrieve your Pivotal Network API from:")
	c.UI.Say("https://network.pivotal.io/users/dashboard/edit-profile")
	c.token = c.UI.AskForPassword("API token")
	return c.token
}

func (c *Config) GetMinMemory() uint64 {
	return c.MinMemory
}

func (c *Config) GetMaxMemory() uint64 {
	return c.MaxMemory
}

func (c *Config) GetDesiredMemory() (uint64, error) {
	if memory := os.Getenv("VM_MEMORY"); memory != "" {
		mb, err := strconv.ParseUint(memory, 10, 64)
		if err != nil {
			return uint64(0), fmt.Errorf("could not convert VM_MEMORY \"%s\" to integer: %s", memory, err)
		}
		return mb, nil
	}
	return uint64(0), nil
}

package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pivotal-cf/pcfdev-cli/user"
)

type Config struct {
	UI        UI
	FS        FS
	MinMemory uint64
	MaxMemory uint64
	token     string
}

//go:generate mockgen -package mocks -destination mocks/fs.go github.com/pivotal-cf/pcfdev-cli/config FS
type FS interface {
	Exists(path string) (bool, error)
	Write(path string, contents io.Reader) error
	Read(path string) (contents []byte, err error)
	RemoveFile(path string) error
}

//go:generate mockgen -package mocks -destination mocks/ui.go github.com/pivotal-cf/pcfdev-cli/config UI
type UI interface {
	AskForPassword(string, ...interface{}) string
	Say(string, ...interface{})
}

func (c *Config) GetToken() (token string, err error) {
	if c.token != "" {
		return c.token, nil
	}

	pcfdevDir, err := c.PCFDevDir()
	if err != nil {
		return "", err
	}

	if envToken := os.Getenv("PIVNET_TOKEN"); envToken != "" {
		c.UI.Say("PIVNET_TOKEN set, ignored saved PivNet API token.")
		return envToken, nil
	}

	exists, err := c.FS.Exists(filepath.Join(pcfdevDir, "token"))
	if err != nil {
		return "", err
	}

	if exists {
		token, err := c.FS.Read(filepath.Join(pcfdevDir, "token"))
		if err != nil {
			return "", err
		}
		c.token = string(token)
		return c.token, nil
	}

	c.UI.Say("Please retrieve your Pivotal Network API from:")
	c.UI.Say("https://network.pivotal.io/users/dashboard/edit-profile")
	c.token = c.UI.AskForPassword("API token")
	return c.token, nil
}

func (c *Config) SaveToken() error {
	if envToken := os.Getenv("PIVNET_TOKEN"); envToken != "" {
		return nil
	}

	pcfdevDir, err := c.PCFDevDir()
	if err != nil {
		return err
	}

	return c.FS.Write(filepath.Join(pcfdevDir, "token"), strings.NewReader(c.token))
}

func (c *Config) DestroyToken() error {
	pcfdevDir, err := c.PCFDevDir()
	if err != nil {
		panic(err)
	}

	exists, err := c.FS.Exists(filepath.Join(pcfdevDir, "token"))
	if err != nil {
		panic(err)
	}

	if exists {
		err := c.FS.RemoveFile(filepath.Join(pcfdevDir, "token"))
		if err != nil {
			panic(err)
		}
	}
	return nil
}

func (c *Config) PCFDevDir() (dir string, err error) {
	if pcfdevHome := os.Getenv("PCFDEV_HOME"); pcfdevHome != "" {
		return filepath.Join(pcfdevHome, ".pcfdev"), nil
	}

	homeDir, err := user.GetHome()
	if err != nil {
		return "", fmt.Errorf("failed to find home directory: %s", err)
	}

	return filepath.Join(homeDir, ".pcfdev"), nil
}

func (c *Config) GetHTTPProxy() string {
	if proxy := os.Getenv("HTTP_PROXY"); proxy != "" {
		return proxy
	}
	return os.Getenv("http_proxy")
}

func (c *Config) GetHTTPSProxy() string {
	if proxy := os.Getenv("HTTPS_PROXY"); proxy != "" {
		return proxy
	}
	return os.Getenv("https_proxy")
}

func (c *Config) GetNoProxy() string {
	if proxy := os.Getenv("NO_PROXY"); proxy != "" {
		return proxy
	}
	return os.Getenv("no_proxy")
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

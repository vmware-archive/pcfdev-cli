package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pivotal-cf/pcfdev-cli/user"
)

type Config struct {
	DefaultVMName string
	PCFDevHome    string
	OVADir        string
	HTTPProxy     string
	HTTPSProxy    string
	NoProxy       string
	MinMemory     uint64
	MaxMemory     uint64
	TotalMemory   uint64
	FreeMemory    uint64
	DefaultMemory uint64
	DefaultCPUs   int
}

//go:generate mockgen -package mocks -destination mocks/system.go github.com/pivotal-cf/pcfdev-cli/config System
type System interface {
	TotalMemory() (uint64, error)
	FreeMemory() (uint64, error)
	PhysicalCores() (int, error)
}

func New(defaultVMName string, system System) (*Config, error) {
	pcfdevHome, err := getPCFDevHome()
	if err != nil {
		return nil, err
	}
	freeMemory, err := system.FreeMemory()
	if err != nil {
		return nil, err
	}
	totalMemory, err := system.TotalMemory()
	if err != nil {
		return nil, err
	}
	cores, err := system.PhysicalCores()
	if err != nil {
		return nil, err
	}
	minMemory := uint64(3072)
	maxMemory := uint64(4096)

	return &Config{
		DefaultVMName: defaultVMName,
		PCFDevHome:    pcfdevHome,
		OVADir:        filepath.Join(pcfdevHome, "ova"),
		HTTPProxy:     getHTTPProxy(),
		HTTPSProxy:    getHTTPSProxy(),
		NoProxy:       getNoProxy(),
		MinMemory:     minMemory,
		MaxMemory:     maxMemory,
		TotalMemory:   totalMemory,
		FreeMemory:    freeMemory,
		DefaultMemory: getDefaultMemory(totalMemory, minMemory, maxMemory),
		DefaultCPUs:   cores,
	}, nil
}

func getPCFDevHome() (string, error) {
	if pcfdevHome := os.Getenv("PCFDEV_HOME"); pcfdevHome != "" {
		return pcfdevHome, nil
	}

	homeDir, err := user.GetHome()
	if err != nil {
		return "", fmt.Errorf("failed to find home directory: %s", err)
	}

	return filepath.Join(homeDir, ".pcfdev"), nil
}

func getHTTPProxy() string {
	if proxy := os.Getenv("HTTP_PROXY"); proxy != "" {
		return proxy
	}
	return os.Getenv("http_proxy")
}

func getHTTPSProxy() string {
	if proxy := os.Getenv("HTTPS_PROXY"); proxy != "" {
		return proxy
	}
	return os.Getenv("https_proxy")
}

func getNoProxy() string {
	if proxy := os.Getenv("NO_PROXY"); proxy != "" {
		return proxy
	}
	return os.Getenv("no_proxy")
}

func getDefaultMemory(totalMemory, minMemory, maxMemory uint64) uint64 {
	halfTotal := totalMemory / 2
	if halfTotal <= minMemory {
		return minMemory
	} else if halfTotal >= maxMemory {
		return maxMemory
	}
	return halfTotal
}

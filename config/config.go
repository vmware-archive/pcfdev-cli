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
	DesiredMemory uint64
}

func New(defaultVMName string, minMemory uint64, maxMemory uint64) (*Config, error) {
	pcfdevHome, err := getPCFDevHome()
	if err != nil {
		return nil, err
	}

	return &Config{
		DefaultVMName: defaultVMName,
		PCFDevHome:    pcfdevHome,
		OVADir:        filepath.Join(pcfdevHome, "ova"),
		HTTPProxy:     getHTTPProxy(),
		HTTPSProxy:    getHTTPSProxy(),
		NoProxy:       getNoProxy(),
		MinMemory:     minMemory,
		MaxMemory:     maxMemory,
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

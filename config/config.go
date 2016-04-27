package config

import "os"

type Config struct {
	UI UI
}

//go:generate mockgen -package mocks -destination mocks/ui.go github.com/pivotal-cf/pcfdev-cli/config UI
type UI interface {
	AskForPassword(string, ...interface{}) string
}

func (c *Config) GetToken() string {
	envToken := os.Getenv("PIVNET_TOKEN")
	if envToken != "" {
		return envToken
	}
	return c.UI.AskForPassword("Enter your Pivotal Network API token:")
}

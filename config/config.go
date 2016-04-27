package config

import "os"

type Config struct {
	UI    UI
	token string
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

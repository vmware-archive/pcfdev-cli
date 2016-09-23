package pivnet

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pivotal-cf/pcfdev-cli/config"
)

//go:generate mockgen -package mocks -destination mocks/fs.go github.com/pivotal-cf/pcfdev-cli/pivnet FS
type FS interface {
	Exists(path string) (bool, error)
	Read(path string) (contents []byte, err error)
	Write(path string, contents io.Reader) error
	Remove(path string) error
}

//go:generate mockgen -package mocks -destination mocks/client.go github.com/pivotal-cf/pcfdev-cli/pivnet PivnetClient
type PivnetClient interface {
	GetToken(string, string) (string, error)
}

//go:generate mockgen -package mocks -destination mocks/ui.go github.com/pivotal-cf/pcfdev-cli/pivnet UI
type UI interface {
	Ask(string) string
	AskForPassword(string) string
	Say(string, ...interface{})
}

type Token struct {
	Config *config.Config
	FS     FS
	Client PivnetClient
	UI     UI
	token  string
}

func (t *Token) Get() (string, error) {
	if t.token != "" {
		return t.token, nil
	}

	if envToken := os.Getenv("PIVNET_TOKEN"); envToken != "" {
		t.UI.Say("PIVNET_TOKEN set, ignored saved PivNet API token.")
		t.token = strings.TrimSpace(envToken)
		return t.token, nil
	}

	exists, err := t.FS.Exists(filepath.Join(t.Config.PCFDevHome, "token"))
	if err != nil {
		return "", err
	}

	if exists {
		token, err := t.FS.Read(filepath.Join(t.Config.PCFDevHome, "token"))
		if err != nil {
			return "", err
		}
		t.token = string(token)
		return t.token, nil
	}

	t.UI.Say("Please sign in with your Pivotal Network account.")
	t.UI.Say("Need an account? Join Pivotal Network: https://network.pivotal.io")
	username := t.UI.Ask("Email")
	password := t.UI.AskForPassword("Password")

	if t.token, err = t.Client.GetToken(username, password); err != nil {
		return "", err
	}

	return t.token, nil
}

func (t *Token) Save() error {
	if envToken := os.Getenv("PIVNET_TOKEN"); envToken != "" {
		return nil
	}

	exists, err := t.FS.Exists(filepath.Join(t.Config.PCFDevHome, "token"))
	if err != nil {
		return err
	}
	if exists {
		if err := t.FS.Remove(filepath.Join(t.Config.PCFDevHome, "token")); err != nil {
			return err
		}
	}

	return t.FS.Write(filepath.Join(t.Config.PCFDevHome, "token"), strings.NewReader(t.token))
}

func (t *Token) Destroy() error {
	if os.Getenv("PIVNET_TOKEN") != "" {
		return nil
	}
	exists, err := t.FS.Exists(filepath.Join(t.Config.PCFDevHome, "token"))
	if err != nil {
		return err
	}
	if exists {
		if err := t.FS.Remove(filepath.Join(t.Config.PCFDevHome, "token")); err != nil {
			return err
		}
	}
	return nil
}

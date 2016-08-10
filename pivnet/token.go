package pivnet

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pivotal-cf/pcfdev-cli/config"
)

type Token struct {
	Config *config.Config
	FS     FS
	UI     UI

	token string
}

func (t *Token) Get() (string, error) {
	if t.token != "" {
		return t.token, nil
	}

	if envToken := os.Getenv("PIVNET_TOKEN"); envToken != "" {
		t.UI.Say("PIVNET_TOKEN set, ignored saved PivNet API token.")
		t.token = envToken
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

	t.UI.Say("Please retrieve your Pivotal Network API token from:")
	t.UI.Say("https://network.pivotal.io/users/dashboard/edit-profile")
	t.token = t.UI.AskForPassword("API token")
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

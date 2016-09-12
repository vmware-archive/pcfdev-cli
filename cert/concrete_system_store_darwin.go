package cert

import (
	"os/exec"
	"path/filepath"

	"github.com/pivotal-cf/pcfdev-cli/user"
)

func (c *ConcreteSystemStore) Store(path string) error {
	home, err := user.GetHome()
	if err != nil {
		return err
	}

	return exec.Command("security", "add-trusted-cert", "-d", "-r", "trustRoot", "-k", filepath.Join(home, "Library", "Keychains", "login.keychain"), path).Run()
}

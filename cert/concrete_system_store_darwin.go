package cert

import (
	"os/exec"
	"path/filepath"

	"github.com/pivotal-cf/pcfdev-cli/user"
)

func (c *ConcreteSystemStore) Store(path string) error {
	pcfdevKeychain, err := c.pcfdevKeychain()
	if err != nil {
		return err
	}

	exec.Command("security", "create-keychain", "-p", "pcfdev", "pcfdev.keychain").Run()
	return exec.Command("security", "add-trusted-cert", "-d", "-r", "trustRoot", "-k", pcfdevKeychain, path).Run()
}

func (c *ConcreteSystemStore) Unstore() error {
	pcfdevKeychain, err := c.pcfdevKeychain()
	if err != nil {
		return err
	}

	tmpDir, err := c.FS.TempDir()
	if err != nil {
		return err
	}
	defer c.FS.Remove(tmpDir)

	certsPath := filepath.Join(tmpDir, "certs.pem")

	if err := exec.Command("security", "export", "-k", pcfdevKeychain, "-p", "-o", certsPath).Run(); err != nil {
		return nil
	}

	exec.Command("security", "remove-trusted-cert", "-d", certsPath).Run()
	exec.Command("security", "delete-keychain", "pcfdev.keychain").Run()

	return nil
}

func (c *ConcreteSystemStore) pcfdevKeychain() (string, error) {
	home, err := user.GetHome()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, "Library", "Keychains", "pcfdev.keychain"), nil
}

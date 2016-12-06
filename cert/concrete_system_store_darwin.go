package cert

import (
	"path/filepath"

	"github.com/pivotal-cf/pcfdev-cli/helpers"
	"github.com/pivotal-cf/pcfdev-cli/user"
)

func (c *ConcreteSystemStore) Store(path string) error {
	pcfdevKeychain, err := c.pcfdevKeychain()
	if err != nil {
		return err
	}

	c.createOrReplaceKeychain("pcfdev.keychain", "pcfdev")
	if err := c.loadKeychain("pcfdev.keychain"); err != nil {
		return err
	}
	_, err = c.CmdRunner.Run("security", "add-trusted-cert", "-d", "-r", "trustRoot", "-k", pcfdevKeychain, path)
	return err
}

func (c *ConcreteSystemStore) Unstore() error {
	pcfdevKeychainPath, err := c.pcfdevKeychain()
	if err != nil {
		return err
	}

	pcfdevKeychain := &Keychain{
		CommandRunner: c.CmdRunner,
		Name:          "pcfdev.keychain",
		Path:          pcfdevKeychainPath,
		FS:            c.FS,
	}

	if !pcfdevKeychain.Exists() {
		return nil
	}

	return pcfdevKeychain.Delete()
}

func (c *ConcreteSystemStore) createOrReplaceKeychain(keychain string, password string) {
	helpers.IgnoreErrorFrom(c.CmdRunner.Run("security", "create-keychain", "-p", password, keychain))
}

func (c *ConcreteSystemStore) loadKeychain(keychain string) error {
	_, err := c.CmdRunner.Run("security", "list-keychains", "-d", "user", "-s", "login.keychain", keychain)
	return err
}

func (c *ConcreteSystemStore) pcfdevKeychain() (string, error) {
	home, err := user.GetHome()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, "Library", "Keychains", "pcfdev.keychain"), nil
}

type Keychain struct {
	CommandRunner CmdRunner
	Name          string
	Path          string
	FS            FS
}

func (k *Keychain) Exists() bool {
	_, err := k.CommandRunner.Run("security", "showkeychaininfo", k.Path)
	return err == nil
}

func (k *Keychain) Delete() error {
	tmpDir, err := k.FS.TempDir()
	if err != nil {
		return err
	}
	defer k.FS.Remove(tmpDir)
	certsPath := filepath.Join(tmpDir, "certs.pem")

	if _, err := k.CommandRunner.Run("security", "export", "-k", k.Path, "-p", "-o", certsPath); err != nil {
		return err
	}

	if _, err := k.CommandRunner.Run("security", "remove-trusted-cert", "-d", certsPath); err != nil {
		return err
	}

	if _, err := k.CommandRunner.Run("security", "delete-keychain", k.Name); err != nil {
		return err
	}

	return nil
}

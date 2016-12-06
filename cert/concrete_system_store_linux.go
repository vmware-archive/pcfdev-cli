package cert

import (
	"bytes"
	"errors"
	"github.com/pivotal-cf/pcfdev-cli/helpers"
	"os/exec"
)

func (c *ConcreteSystemStore) Store(path string) error {
	caCertsPath, err := c.getCaCertPath()
	if err != nil {
		return err
	}

	contents, err := c.FS.Read(path)
	if err != nil {
		return err
	}

	if err := c.FS.Write(caCertsPath, bytes.NewReader(contents), true); err != nil {
		return err
	}

	return nil
}

func (c *ConcreteSystemStore) Unstore() error {
	helpers.IgnoreErrorFrom(exec.Command("sudo", "update-ca-certificates", "--fresh").Run())
	helpers.IgnoreErrorFrom(exec.Command("sudo", "update-ca-trust", "extract").Run())
	return nil
}

func (c *ConcreteSystemStore) getCaCertPath() (string, error) {
	paths := []string{
		"/etc/ssl/certs/ca-certificates.crt",
		"/etc/pki/tls/certs/ca-bundle.crt",
		"/etc/ssl/ca-bundle.pem",
		"/etc/pki/tls/cacert.pem",
	}

	for _, path := range paths {
		exists, err := c.FS.Exists(path)
		if err != nil {
			return "", err
		}

		if exists {
			return path, nil
		}
	}

	return "", errors.New("failed to determine path to CA Cert Store")
}

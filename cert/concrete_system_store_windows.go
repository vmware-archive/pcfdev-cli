package cert

import (
	"os/exec"

	"github.com/pivotal-cf/pcfdev-cli/address"
)

func (c *ConcreteSystemStore) Store(path string) error {
	return exec.Command("certutil", "-addstore", "-f", "ROOT", path).Run()
}

func (c *ConcreteSystemStore) Unstore() error {
	for _, domain := range address.AllowedAddresses {
		exec.Command("certutil", "-delstore", "ROOT", domain).Run()
	}
	return nil
}

package cert

import "github.com/pivotal-cf/pcfdev-cli/address"

func (c *ConcreteSystemStore) Store(path string) error {
	_, err := c.CmdRunner.Run("certutil", "-addstore", "-f", "ROOT", path)
	return err
}

func (c *ConcreteSystemStore) Unstore() error {
	for _, domain := range address.AllowedAddresses {
		c.CmdRunner.Run("certutil", "-delstore", "ROOT", domain)
	}

	return nil
}

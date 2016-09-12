package cert

import "os/exec"

func (c *ConcreteSystemStore) Store(path string) error {
	return exec.Command("certutil", "-addstore", "-f", "ROOT", path).Run()
}

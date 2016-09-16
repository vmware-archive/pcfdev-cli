package runner

import "os/exec"

type CmdRunner struct{}

func (c *CmdRunner) Run(command string, args ...string) ([]byte, error) {
	return exec.Command(command, args...).CombinedOutput()
}

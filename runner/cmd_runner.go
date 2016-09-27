package runner

import (
	"fmt"
	"os/exec"
	"strings"
)

type CmdRunner struct{}

func (c *CmdRunner) Run(command string, args ...string) ([]byte, error) {
	output, err := exec.Command(command, args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to execute '%s %s': %s: %s", command, strings.Join(args, " "), err, output)
	}

	return output, nil
}

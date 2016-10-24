package ssh

import "golang.org/x/crypto/ssh/terminal"

type TerminalWrapper struct{}

func (TerminalWrapper) MakeRaw(fd int) (*terminal.State, error) {
	return terminal.MakeRaw(fd)
}

func (TerminalWrapper) Restore(fd int, state *terminal.State) error {
	return terminal.Restore(fd, state)
}
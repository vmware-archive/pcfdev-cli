package ssh

import "github.com/docker/docker/pkg/term"

type TerminalWrapper struct{}

func (*TerminalWrapper) SetRawTerminal(fd uintptr) (*term.State, error) {
	return term.SetRawTerminal(fd)
}


func (*TerminalWrapper) RestoreTerminal(fd uintptr, state *term.State) error {
	return term.RestoreTerminal(fd, state)
}

func (*TerminalWrapper) GetWinsize(fd uintptr) (*term.Winsize, error) {
	return term.GetWinsize(fd)
}
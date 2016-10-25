// +build !windows

package ssh

import (
	"golang.org/x/crypto/ssh"
	"os"
	"os/signal"
	"syscall"
)

func (w *WindowResizer) StartResizing(session *ssh.Session) {
	go func() {
		var previousWidth, previousHeight uint16

		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGWINCH)
		defer close(c)

		for {
			select {
			case <-w.DoneChannel:
				return
			case <-c:
				previousWidth, previousWidth = w.resize(
					session,
					previousHeight,
					previousWidth,
				)
			}
		}
	}()
}
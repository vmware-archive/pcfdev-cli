package ssh

import (
	"golang.org/x/crypto/ssh"
	"time"
)

func (w *ConcreteWindowResizer) StartResizing(session *ssh.Session) {
	go func() {
		var previousWidth, previousHeight uint16

		ticker := time.NewTicker(250 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-w.DoneChannel:
				return
			case <-ticker.C:
				previousWidth, previousHeight = w.resize(
					session,
					previousHeight,
					previousWidth,
				)
			}
		}
	}()
}

package ssh

import (
	"github.com/docker/docker/pkg/term"
	"golang.org/x/crypto/ssh"
	"os"
)

type ConcreteWindowResizer struct {
	DoneChannel chan bool
}

func (w *ConcreteWindowResizer) StopResizing() {
	close(w.DoneChannel)
}

func (w *ConcreteWindowResizer) resize(session *ssh.Session, previousHeight, previousWidth uint16) (newHeight, newWidth uint16) {
	winSize, err := term.GetWinsize(os.Stdout.Fd())
	if err != nil {
		return previousWidth, previousHeight
	}

	height := winSize.Height
	width := winSize.Width

	if width == previousWidth && height == previousHeight {
		return previousWidth, previousHeight
	}

	if _, err := session.SendRequest(
		"window-change",
		false,
		ssh.Marshal(struct {
			Width       uint32
			Height      uint32
			PixelWidth  uint32
			PixelHeight uint32
		}{
			uint32(width),
			uint32(height),
			0,
			0,
		})); err != nil {
		return width, height
	}

	return previousHeight, previousWidth
}

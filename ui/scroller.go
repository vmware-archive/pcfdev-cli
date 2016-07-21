package ui

import (
	"io"
	"math"
	"strings"
	"sync"

	"github.com/gizak/termui"
)

type Scroller struct {
	sync.RWMutex

	textHeight int
	textWidth  int
	items      []string
	index      int
	text       string
}

var file io.WriteCloser

const borderSize int = 2

func NewScroller(height int, width int) *Scroller {
	return &Scroller{
		textHeight: height - borderSize,
		textWidth:  width - borderSize,
	}
}

func (s *Scroller) SetText(text string) {
	s.Lock()
	defer s.Unlock()
	s.setText(text)
}

func (s *Scroller) setText(text string) {
	s.text = text
	s.index = 0
	text = strings.Replace(text, "\r", " ", -1)
	lines := strings.Split(text, "\n")
	items := make([]string, 0, 0)
	for _, line := range lines {
		for i := 0; i < len(line); i += s.textWidth - 2 {
			endIndex := int(math.Min(float64(i+s.textWidth-2), float64(len(line))))
			displayLine := line[i:endIndex]
			items = append(items, " "+displayLine)
		}
	}
	s.items = items
}

func (s *Scroller) Resize(height int, width int) {
	s.Lock()
	defer s.Unlock()
	s.textHeight = height - borderSize
	s.textWidth = width - borderSize
	s.setText(s.text)
}

func (s *Scroller) Buffer() termui.Buffer {
	s.Lock()
	defer s.Unlock()
	list := termui.NewList()
	list.Height = s.textHeight + borderSize
	list.Width = s.textWidth + borderSize
	maxItems := int(math.Min(float64(s.textHeight), float64(len(s.items))))
	list.Items = s.items[s.index : s.index+maxItems]
	return list.Buffer()
}

func (s *Scroller) ScrollDown() {
	s.Lock()
	defer s.Unlock()
	if s.index < s.maxIndex() {
		s.index++
	}
}

func (s *Scroller) ScrollUp() {
	s.Lock()
	defer s.Unlock()
	if s.index > 0 {
		s.index--
	}
}

func (s *Scroller) PageDown() {
	s.Lock()
	defer s.Unlock()
	nextIndex := s.index + s.textHeight
	if nextIndex < s.maxIndex() {
		s.index = nextIndex
	} else {
		s.index = s.maxIndex()
	}
}

func (s *Scroller) PageUp() {
	s.Lock()
	defer s.Unlock()
	if s.index-s.textHeight < 0 {
		s.index = 0
	} else {
		s.index -= s.textHeight
	}
}

func (s *Scroller) Home() {
	s.Lock()
	defer s.Unlock()
	s.index = 0
}

func (s *Scroller) End() {
	s.Lock()
	defer s.Unlock()
	s.index = s.maxIndex()
}

func (s *Scroller) maxIndex() int {
	if len(s.items) < s.textHeight {
		return 0
	}
	return len(s.items) - s.textHeight
}

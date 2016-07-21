package ui

import (
	"os"
	"os/exec"
	"runtime"

	"github.com/gizak/termui"
)

type UI struct {
	Scroller    TextScroller
	OptionsList *termui.List
}

const optionsListHeight int = 6

//go:generate mockgen -package mocks -destination mocks/text_scroller.go github.com/pivotal-cf/pcfdev-cli/ui TextScroller
type TextScroller interface {
	SetText(string)
	Resize(height int, width int)
	Buffer() termui.Buffer
	ScrollDown()
	ScrollUp()
	PageDown()
	PageUp()
	Home()
	End()
}

func (u *UI) Init() error {
	if err := termui.Init(); err != nil {
		return err
	}
	height := termui.TermHeight() - optionsListHeight
	width := termui.TermWidth()
	optionsList := termui.NewList()
	optionsList.Items = []string{"[<up>, <previous>] Scroll up", "[<down>, <next>] Scroll down", "[y] Accept", "[n] Do Not Accept"}
	optionsList.Height = optionsListHeight
	optionsList.Width = width
	optionsList.Y = height
	u.Scroller = NewScroller(height, width)
	u.OptionsList = optionsList
	return nil
}

func (u *UI) Close() error {
	termui.Close()
	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		return cmd.Run()
	}
	return nil
}

func (u *UI) ConfirmText(text string) bool {
	u.Scroller.SetText(text)
	accepted := false
	termui.Render(u.Scroller, u.OptionsList)

	termui.Handle("/sys/kbd/y", func(termui.Event) {
		accepted = true
		termui.StopLoop()
	})
	termui.Handle("/sys/kbd/Y", func(termui.Event) {
		accepted = true
		termui.StopLoop()
	})
	termui.Handle("/sys/kbd/n", func(termui.Event) {
		termui.StopLoop()
	})
	termui.Handle("/sys/kbd/N", func(termui.Event) {
		termui.StopLoop()
	})
	termui.Handle("/sys/kbd/<down>", func(termui.Event) {
		u.Scroller.ScrollDown()
		termui.Render(u.Scroller)
	})
	termui.Handle("/sys/kbd/<up>", func(termui.Event) {
		u.Scroller.ScrollUp()
		termui.Render(u.Scroller)
	})
	termui.Handle("/sys/kbd/<next>", func(termui.Event) {
		u.Scroller.PageDown()
		termui.Render(u.Scroller)
	})
	termui.Handle("/sys/kbd/<previous>", func(termui.Event) {
		u.Scroller.PageUp()
		termui.Render(u.Scroller)
	})
	termui.Handle("/sys/kbd/<home>", func(termui.Event) {
		u.Scroller.Home()
		termui.Render(u.Scroller)
	})
	termui.Handle("/sys/kbd/<end>", func(termui.Event) {
		u.Scroller.End()
		termui.Render(u.Scroller)
	})
	termui.Handle("/sys/wnd/resize", func(evt termui.Event) {
		e := evt.Data.(termui.EvtWnd)
		u.Scroller.Resize(e.Height-optionsListHeight, e.Width)
		u.OptionsList.Width = e.Width
		u.OptionsList.Y = e.Height - optionsListHeight
		termui.Render(u.Scroller, u.OptionsList)
	})
	termui.Loop()

	return accepted
}

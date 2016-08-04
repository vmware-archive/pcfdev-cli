package ui_test

import (
	"github.com/gizak/termui"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/pcfdev-cli/ui"
)

var _ = Describe("scroller", func() {
	It("should return a termui buffer sized to fit the terminal", func() {
		scroller := ui.NewScroller(5, 13)
		scroller.SetText("some-text\nsome-more-text\nextra stuff")
		buffer := scroller.Buffer()

		list := termui.NewList()
		list.Height = 5
		list.Width = 13
		list.Items = []string{
			" some-text",
			" some-more",
			" -text",
		}

		Expect(buffer).To(Equal(list.Buffer()))
	})

	Describe("ScrollDown", func() {
		It("should scroll down by one line", func() {
			scroller := ui.NewScroller(4, 13)
			scroller.SetText("some-text\nsome-more-text")
			scroller.ScrollDown()
			buffer := scroller.Buffer()

			list := termui.NewList()
			list.Height = 4
			list.Width = 13
			list.Items = []string{
				" some-more",
				" -text",
			}

			Expect(buffer).To(Equal(list.Buffer()))
		})

		It("should not go past the last line", func() {
			scroller := ui.NewScroller(4, 13)
			scroller.SetText("some-text\nsome-more-text")
			scroller.ScrollDown()
			scroller.ScrollDown()
			scroller.ScrollDown()
			buffer := scroller.Buffer()

			list := termui.NewList()
			list.Height = 4
			list.Width = 13
			list.Items = []string{
				" some-more",
				" -text",
			}

			Expect(buffer).To(Equal(list.Buffer()))
		})
	})

	Describe("ScrollUp", func() {
		It("should scroll up by one line", func() {
			scroller := ui.NewScroller(4, 14)
			scroller.SetText("first-line\nsome-text\nsome-more-text")

			scroller.ScrollDown()
			scroller.ScrollDown()
			scroller.ScrollUp()

			list := termui.NewList()
			list.Height = 4
			list.Width = 14
			list.Items = []string{
				" some-text",
				" some-more-",
			}

			Expect(scroller.Buffer()).To(Equal(list.Buffer()))
		})

		It("should not go past the first line", func() {
			scroller := ui.NewScroller(4, 14)
			scroller.SetText("first-line\nsome-text\nsome-more-text")

			scroller.ScrollUp()

			list := termui.NewList()
			list.Height = 4
			list.Width = 14
			list.Items = []string{
				" first-line",
				" some-text",
			}

			Expect(scroller.Buffer()).To(Equal(list.Buffer()))
		})
	})

	Describe("PageDown", func() {
		It("should scroll down by the scroller height", func() {
			scroller := ui.NewScroller(4, 12)
			scroller.SetText("line 1\nline 2\nline 3\nline 4")

			scroller.PageDown()

			list := termui.NewList()
			list.Height = 4
			list.Width = 12
			list.Items = []string{
				" line 3",
				" line 4",
			}

			Expect(scroller.Buffer()).To(Equal(list.Buffer()))
		})

		It("should not go past the last line", func() {
			scroller := ui.NewScroller(4, 12)
			scroller.SetText("line 1\nline 2\nline 3\nline 4")

			scroller.PageDown()
			scroller.PageDown()

			list := termui.NewList()
			list.Height = 4
			list.Width = 12
			list.Items = []string{
				" line 3",
				" line 4",
			}

			Expect(scroller.Buffer()).To(Equal(list.Buffer()))
		})
	})

	Describe("PageUp", func() {
		It("should scroll up by the scroller height", func() {
			scroller := ui.NewScroller(4, 12)
			scroller.SetText("line 1\nline 2\nline 3\nline 4\nline 5\nline 6")

			scroller.PageDown()
			scroller.PageDown()
			scroller.PageUp()

			list := termui.NewList()
			list.Height = 4
			list.Width = 12
			list.Items = []string{
				" line 3",
				" line 4",
			}

			Expect(scroller.Buffer()).To(Equal(list.Buffer()))
		})

		It("should not scroll past the first line", func() {
			scroller := ui.NewScroller(4, 12)
			scroller.SetText("line 1\nline 2\nline 3\nline 4\nline 5\nline 6")

			scroller.PageUp()
			scroller.PageUp()

			list := termui.NewList()
			list.Height = 4
			list.Width = 12
			list.Items = []string{
				" line 1",
				" line 2",
			}

			Expect(scroller.Buffer()).To(Equal(list.Buffer()))
		})
	})

	Describe("Home", func() {
		It("should scroll to the beginning of the text", func() {
			scroller := ui.NewScroller(4, 12)
			scroller.SetText("line 1\nline 2\nline 3\nline 4\nline 5\nline 6")

			scroller.PageDown()
			scroller.PageDown()
			scroller.Home()

			list := termui.NewList()
			list.Height = 4
			list.Width = 12
			list.Items = []string{
				" line 1",
				" line 2",
			}

			Expect(scroller.Buffer()).To(Equal(list.Buffer()))
		})
	})

	Describe("End", func() {
		It("should scroll to the end of the text", func() {
			scroller := ui.NewScroller(4, 12)
			scroller.SetText("line 1\nline 2\nline 3\nline 4\nline 5\nline 6")
			scroller.End()

			list := termui.NewList()
			list.Height = 4
			list.Width = 12
			list.Items = []string{
				" line 5",
				" line 6",
			}

			Expect(scroller.Buffer()).To(Equal(list.Buffer()))
		})
	})

	Describe("Resize", func() {
		It("should reset height and width", func() {
			scroller := ui.NewScroller(4, 30)
			scroller.SetText("line 1\nline 2line2\nline 3\nline 4\nline 5\nline 6")

			list := termui.NewList()
			list.Height = 4
			list.Width = 30
			list.Items = []string{
				" line 1",
				" line 2line2",
			}

			Expect(scroller.Buffer()).To(Equal(list.Buffer()))

			scroller.Resize(5, 10)
			resizedList := termui.NewList()
			resizedList.Height = 5
			resizedList.Width = 10
			resizedList.Items = []string{
				" line 1",
				" line 2",
				" line2",
			}

			Expect(scroller.Buffer()).To(Equal(resizedList.Buffer()))
		})
	})
})

// +build !linux

package ui_test

import (
	"runtime"
	"time"

	"github.com/gizak/termui"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/pcfdev-cli/ui"
	"github.com/pivotal-cf/pcfdev-cli/ui/mocks"
)

var _ = Describe("ui", func() {
	var (
		mockCtrl         *gomock.Controller
		mockTextScroller *mocks.MockTextScroller
		u                *ui.UI
		optionsList      *termui.List
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockTextScroller = mocks.NewMockTextScroller(mockCtrl)
		Expect(termui.Init()).To(Succeed())
		optionsList = termui.NewList()

		u = &ui.UI{
			Scroller:    mockTextScroller,
			OptionsList: optionsList,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("ConfirmText", func() {
		AfterEach(func() {
			termui.Close()
		})

		Context("when the user presses 'y'", func() {
			It("should return true", func(done Done) {
				mockTextScroller.EXPECT().SetText("some-text")
				mockTextScroller.EXPECT().Buffer().Return(termui.NewBuffer())
				allowResizeOnWindows(mockTextScroller)

				go func() {
					defer GinkgoRecover()
					Expect(u.ConfirmText("some-text")).To(BeTrue())
					close(done)
				}()
				time.Sleep(time.Second)
				termui.SendCustomEvt("/sys/kbd/y", nil)
			}, 2)
		})

		Context("when the user presses 'y'", func() {
			It("should return true", func(done Done) {
				mockTextScroller.EXPECT().SetText("some-text")
				mockTextScroller.EXPECT().Buffer().Return(termui.NewBuffer())
				allowResizeOnWindows(mockTextScroller)

				go func() {
					defer GinkgoRecover()
					Expect(u.ConfirmText("some-text")).To(BeTrue())
					close(done)
				}()
				time.Sleep(time.Second)
				termui.SendCustomEvt("/sys/kbd/Y", nil)
			}, 2)
		})

		Context("when the user presses 'n'", func() {
			It("should return false", func(done Done) {
				mockTextScroller.EXPECT().SetText("some-text")
				mockTextScroller.EXPECT().Buffer().Return(termui.NewBuffer())
				allowResizeOnWindows(mockTextScroller)

				go func() {
					defer GinkgoRecover()
					Expect(u.ConfirmText("some-text")).To(BeFalse())
					close(done)
				}()
				time.Sleep(time.Second)
				termui.SendCustomEvt("/sys/kbd/n", nil)
			}, 2)
		})

		Context("when the user presses 'N'", func() {
			It("should return false", func(done Done) {
				mockTextScroller.EXPECT().SetText("some-text")
				mockTextScroller.EXPECT().Buffer().Return(termui.NewBuffer())
				allowResizeOnWindows(mockTextScroller)

				go func() {
					defer GinkgoRecover()
					Expect(u.ConfirmText("some-text")).To(BeFalse())
					close(done)
				}()
				time.Sleep(time.Second)
				termui.SendCustomEvt("/sys/kbd/N", nil)
			}, 2)
		})

		Context("when the user presses 'down'", func() {
			It("should scroll down", func(done Done) {
				mockTextScroller.EXPECT().SetText("some-text")
				mockTextScroller.EXPECT().Buffer().Times(2).Return(termui.NewBuffer())
				mockTextScroller.EXPECT().ScrollDown()
				allowResizeOnWindows(mockTextScroller)

				go func() {
					defer GinkgoRecover()
					Expect(u.ConfirmText("some-text")).To(BeFalse())
					close(done)
				}()
				termui.SendCustomEvt("/sys/kbd/<down>", nil)
				time.Sleep(time.Second)
				termui.SendCustomEvt("/sys/kbd/n", nil)
			}, 2)
		})

		Context("when the user presses 'up'", func() {
			It("should scroll up", func(done Done) {
				mockTextScroller.EXPECT().SetText("some-text")
				mockTextScroller.EXPECT().Buffer().Times(2).Return(termui.NewBuffer())
				mockTextScroller.EXPECT().ScrollUp()
				allowResizeOnWindows(mockTextScroller)

				go func() {
					defer GinkgoRecover()
					Expect(u.ConfirmText("some-text")).To(BeFalse())
					close(done)
				}()
				termui.SendCustomEvt("/sys/kbd/<up>", nil)
				time.Sleep(time.Second)
				termui.SendCustomEvt("/sys/kbd/n", nil)
			}, 2)
		})

		Context("when the user presses 'page down'", func() {
			It("should scroll page down", func(done Done) {
				mockTextScroller.EXPECT().SetText("some-text")
				mockTextScroller.EXPECT().Buffer().Times(2).Return(termui.NewBuffer())
				mockTextScroller.EXPECT().PageDown()
				allowResizeOnWindows(mockTextScroller)

				go func() {
					defer GinkgoRecover()
					Expect(u.ConfirmText("some-text")).To(BeFalse())
					close(done)
				}()
				termui.SendCustomEvt("/sys/kbd/<next>", nil)
				time.Sleep(time.Second)
				termui.SendCustomEvt("/sys/kbd/n", nil)
			}, 2)
		})

		Context("when the user presses 'page up'", func() {
			It("should page up", func(done Done) {
				mockTextScroller.EXPECT().SetText("some-text")
				mockTextScroller.EXPECT().Buffer().Times(2).Return(termui.NewBuffer())
				mockTextScroller.EXPECT().PageUp()
				allowResizeOnWindows(mockTextScroller)

				go func() {
					defer GinkgoRecover()
					Expect(u.ConfirmText("some-text")).To(BeFalse())
					close(done)
				}()
				termui.SendCustomEvt("/sys/kbd/<previous>", nil)
				time.Sleep(time.Second)
				termui.SendCustomEvt("/sys/kbd/n", nil)
			}, 2)
		})

		Context("when the user presses 'home'", func() {
			It("should go to the top", func(done Done) {
				mockTextScroller.EXPECT().SetText("some-text")
				mockTextScroller.EXPECT().Buffer().Times(2).Return(termui.NewBuffer())
				mockTextScroller.EXPECT().Home()
				allowResizeOnWindows(mockTextScroller)

				go func() {
					defer GinkgoRecover()
					Expect(u.ConfirmText("some-text")).To(BeFalse())
					close(done)
				}()
				termui.SendCustomEvt("/sys/kbd/<home>", nil)
				time.Sleep(time.Second)
				termui.SendCustomEvt("/sys/kbd/n", nil)
			}, 2)
		})

		Context("when the user presses 'end'", func() {
			It("should go to the bottom", func(done Done) {
				mockTextScroller.EXPECT().SetText("some-text")
				mockTextScroller.EXPECT().Buffer().Times(2).Return(termui.NewBuffer())
				mockTextScroller.EXPECT().End()
				allowResizeOnWindows(mockTextScroller)

				go func() {
					defer GinkgoRecover()
					Expect(u.ConfirmText("some-text")).To(BeFalse())
					close(done)
				}()
				termui.SendCustomEvt("/sys/kbd/<end>", nil)
				time.Sleep(time.Second)
				termui.SendCustomEvt("/sys/kbd/n", nil)
			}, 2)
		})

		Context("when the user resizes the window", func() {
			It("should reset scroller height and width", func(done Done) {
				if runtime.GOOS == "windows" {
					Skip("cannot resize terminal on windows")
				}

				mockTextScroller.EXPECT().SetText("some-text")
				mockTextScroller.EXPECT().Resize(6, 7)
				mockTextScroller.EXPECT().Buffer().Times(2).Return(termui.NewBuffer())
				optionsList.Height = 4

				go func() {
					defer GinkgoRecover()
					Expect(u.ConfirmText("some-text")).To(BeFalse())
					Expect(u.OptionsList.Width).To(Equal(7))
					Expect(u.OptionsList.Height).To(Equal(4))
					close(done)
				}()
				termui.SendCustomEvt("/sys/wnd/resize", termui.EvtWnd{Height: 12, Width: 7})
				time.Sleep(time.Second)
				termui.SendCustomEvt("/sys/kbd/n", nil)
			}, 3)
		})
	})
})

func allowResizeOnWindows(mockTextScroller *mocks.MockTextScroller) {
	if runtime.GOOS == "windows" {
		mockTextScroller.EXPECT().Buffer().Return(termui.NewBuffer()).MaxTimes(2)
		mockTextScroller.EXPECT().Resize(gomock.Any(), gomock.Any()).MaxTimes(2)
	}
}

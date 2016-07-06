package vm_test

import (
	"github.com/golang/mock/gomock"
	conf "github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/vm"
	"github.com/pivotal-cf/pcfdev-cli/vm/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Recoverable", func() {
	var (
		mockCtrl    *gomock.Controller
		mockUI      *mocks.MockUI
		mockVBox    *mocks.MockVBox
		recoverable vm.Recoverable
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockVBox = mocks.NewMockVBox(mockCtrl)
		mockUI = mocks.NewMockUI(mockCtrl)

		recoverable = vm.Recoverable{
			UI:   mockUI,
			VBox: mockVBox,
			VMConfig: &conf.VMConfig{
				Name:    "some-vm",
				Domain:  "some-domain",
				IP:      "some-ip",
				SSHPort: "some-port",
			},
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("Stop", func() {
		It("should stop the VM", func() {
			gomock.InOrder(
				mockUI.EXPECT().Say("Stopping VM..."),
				mockVBox.EXPECT().StopVM(recoverable.VMConfig),
				mockUI.EXPECT().Say("PCF Dev is now stopped."),
			)

			recoverable.Stop()
		})
	})

	Describe("VerifyStartOpts", func() {
		It("should say a message", func() {
			Expect(recoverable.VerifyStartOpts(
				&vm.StartOpts{},
			)).To(MatchError("PCF Dev is in an invalid state. Please run 'cf dev destroy' or 'cf dev stop' before attempting to start again"))
		})
	})

	Describe("Start", func() {
		It("should start vm", func() {
			mockUI.EXPECT().Failed("PCF Dev is in an invalid state. Please run 'cf dev destroy' or 'cf dev stop' before attempting to start again.")

			recoverable.Start(&vm.StartOpts{})
		})
	})

	Describe("Status", func() {
		It("should return 'Stopped'", func() {
			Expect(recoverable.Status()).To(Equal("PCF Dev is in an invalid state. Please run 'cf dev destroy' or 'cf dev stop' before attempting to start again."))
		})
	})

	Describe("Suspend", func() {
		It("should say a message", func() {
			mockUI.EXPECT().Failed("PCF Dev is in an invalid state. Please run 'cf dev destroy' or 'cf dev stop' before attempting to start again.")

			recoverable.Suspend()
		})
	})

	Describe("Resume", func() {
		It("should say a message", func() {
			mockUI.EXPECT().Failed("PCF Dev is in an invalid state. Please run 'cf dev destroy' or 'cf dev stop' before attempting to start again.")

			recoverable.Resume()
		})
	})
})

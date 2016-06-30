package vm_test

import (
	"github.com/golang/mock/gomock"
	"github.com/pivotal-cf/pcfdev-cli/vm"
	"github.com/pivotal-cf/pcfdev-cli/vm/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Invalid", func() {
	var (
		mockCtrl *gomock.Controller
		mockUI   *mocks.MockUI
		invalid  vm.Invalid
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockUI = mocks.NewMockUI(mockCtrl)

		invalid = vm.Invalid{
			UI: mockUI,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("Stop", func() {
		It("should say a message", func() {
			mockUI.EXPECT().Failed("PCF Dev is in an invalid state. Please run 'cf dev destroy'.")

			invalid.Stop()
		})
	})

	Describe("VerifyStartOpts", func() {
		It("should say a message", func() {
			Expect(invalid.VerifyStartOpts(
				&vm.StartOpts{},
			)).To(MatchError("PCF Dev is in an invalid state. Please run 'cf dev destroy'"))
		})
	})

	Describe("Start", func() {
		It("should start vm", func() {
			mockUI.EXPECT().Failed("PCF Dev is in an invalid state. Please run 'cf dev destroy'.")

			invalid.Start(&vm.StartOpts{})
		})
	})

	Describe("Status", func() {
		It("should return 'Stopped'", func() {
			Expect(invalid.Status()).To(Equal("PCF Dev is in an invalid state. Please run 'cf dev destroy'."))
		})
	})

	Describe("Suspend", func() {
		It("should say a message", func() {
			mockUI.EXPECT().Failed("PCF Dev is in an invalid state. Please run 'cf dev destroy'.")

			invalid.Suspend()
		})
	})

	Describe("Resume", func() {
		It("should say a message", func() {
			mockUI.EXPECT().Failed("PCF Dev is in an invalid state. Please run 'cf dev destroy'.")

			invalid.Resume()
		})
	})
})

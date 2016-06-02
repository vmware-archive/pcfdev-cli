package vm_test

import (
	"errors"

	"github.com/golang/mock/gomock"
	"github.com/pivotal-cf/pcfdev-cli/vm"
	"github.com/pivotal-cf/pcfdev-cli/vm/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Stopped", func() {
	var (
		mockCtrl *gomock.Controller
		mockUI   *mocks.MockUI
		mockVBox *mocks.MockVBox

		runningVM vm.Running
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockUI = mocks.NewMockUI(mockCtrl)
		mockVBox = mocks.NewMockVBox(mockCtrl)

		runningVM = vm.Running{
			Name:    "some-vm",
			Domain:  "some-domain",
			IP:      "some-ip",
			SSHPort: "some-port",

			VBox: mockVBox,
			UI:   mockUI,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("Stop", func() {
		It("should stop the vm", func() {
			gomock.InOrder(
				mockUI.EXPECT().Say("Stopping VM..."),
				mockVBox.EXPECT().StopVM("some-vm"),
				mockUI.EXPECT().Say("PCF Dev is now stopped"),
			)

			runningVM.Stop()
		})

		Context("when stopped the vm fails", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Stopping VM..."),
					mockVBox.EXPECT().StopVM("some-vm").Return(errors.New("some-error")),
				)

				Expect(runningVM.Stop()).To(MatchError("failed to stop VM: some-error"))
			})
		})
	})

	Describe("Start", func() {
		It("should say a message", func() {
			mockUI.EXPECT().Say("PCF Dev is running")

			runningVM.Start()
		})
	})

	Describe("Status", func() {
		It("should say Running", func() {
			mockUI.EXPECT().Say("Running")

			runningVM.Status()
		})
	})

	Describe("Destroy", func() {
		It("should poweroff and destroy the vm", func() {
			gomock.InOrder(
				mockVBox.EXPECT().PowerOffVM("some-vm").Return(nil),
				mockVBox.EXPECT().DestroyVM("some-vm").Return(nil),
			)

			Expect(runningVM.Destroy()).To(Succeed())
		})

		Context("when powering off the vm fails", func() {
			It("should return an error", func() {
				mockVBox.EXPECT().PowerOffVM("some-vm").Return(errors.New("some-error"))

				Expect(runningVM.Destroy()).To(MatchError("failed to destroy VM: some-error"))
			})
		})

		Context("when destroying the vm fails", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockVBox.EXPECT().PowerOffVM("some-vm").Return(nil),
					mockVBox.EXPECT().DestroyVM("some-vm").Return(errors.New("some-error")),
				)

				Expect(runningVM.Destroy()).To(MatchError("failed to destroy VM: some-error"))
			})
		})
	})

	Describe("Suspend", func() {
		It("should suspend the vm", func() {
			mockUI.EXPECT().Say("Suspending VM...")
			mockVBox.EXPECT().SuspendVM("some-vm").Return(nil)
			mockUI.EXPECT().Say("PCF Dev is now suspended")

			Expect(runningVM.Suspend()).To(Succeed())
		})

		Context("when suspending the vm fails", func() {
			It("should return an error", func() {
				mockUI.EXPECT().Say("Suspending VM...")
				mockVBox.EXPECT().SuspendVM("some-vm").Return(errors.New("some-error"))

				Expect(runningVM.Suspend()).To(MatchError("failed to suspend VM: some-error"))
			})
		})
	})

	Describe("Resume", func() {
		It("should say a message", func() {
			mockUI.EXPECT().Say("PCF Dev is running")

			Expect(runningVM.Resume()).To(Succeed())
		})
	})
})

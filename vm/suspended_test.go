package vm_test

import (
	"errors"

	"github.com/golang/mock/gomock"
	"github.com/pivotal-cf/pcfdev-cli/vm"
	"github.com/pivotal-cf/pcfdev-cli/vm/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Suspended", func() {
	var (
		mockCtrl                *gomock.Controller
		mockUI                  *mocks.MockUI
		mockVBox                *mocks.MockVBox
		mockRequirementsChecker *mocks.MockRequirementsChecker
		suspendedVM             vm.Suspended
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockUI = mocks.NewMockUI(mockCtrl)
		mockVBox = mocks.NewMockVBox(mockCtrl)
		mockRequirementsChecker = mocks.NewMockRequirementsChecker(mockCtrl)

		suspendedVM = vm.Suspended{
			Name:    "some-vm",
			Domain:  "some-domain",
			IP:      "some-ip",
			SSHPort: "some-port",

			VBox:                mockVBox,
			UI:                  mockUI,
			RequirementsChecker: mockRequirementsChecker,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("Suspend", func() {
		It("should say a message", func() {
			mockUI.EXPECT().Say("Your VM is suspended.")
			Expect(suspendedVM.Suspend()).To(Succeed())
		})
	})

	Describe("Stop", func() {
		It("should say a message", func() {
			mockUI.EXPECT().Say("Your VM is currently suspended. You must resume your VM with `cf dev resume` to shut it down.")
			Expect(suspendedVM.Stop()).To(Succeed())
		})
	})

	Describe("Start", func() {
		It("should start vm", func() {
			gomock.InOrder(
				mockRequirementsChecker.EXPECT().Check().Return(nil),
				mockUI.EXPECT().Say("Resuming VM..."),
				mockVBox.EXPECT().ResumeVM("some-vm").Return(nil),
				mockUI.EXPECT().Say("PCF Dev is now running"),
			)

			Expect(suspendedVM.Start()).To(Succeed())
		})

		Context("when the system does not meet requirements and the user accepts to continue", func() {
			It("should print a warning and prompt for the response to continue", func() {
				gomock.InOrder(
					mockRequirementsChecker.EXPECT().Check().Return(errors.New("some-message")),
					mockUI.EXPECT().Confirm("Less than 3 GB of memory detected, continue (y/N): ").Return(true),
					mockUI.EXPECT().Say("Resuming VM..."),
					mockVBox.EXPECT().ResumeVM("some-vm").Return(nil),
					mockUI.EXPECT().Say("PCF Dev is now running"),
				)

				Expect(suspendedVM.Start()).To(Succeed())
			})
		})

		Context("when the system does not meet requirements and the user declines to continue", func() {
			It("should print a warning and prompt for the response to continue", func() {
				gomock.InOrder(
					mockRequirementsChecker.EXPECT().Check().Return(errors.New("some-message")),
					mockUI.EXPECT().Confirm("Less than 3 GB of memory detected, continue (y/N): ").Return(false),
					mockUI.EXPECT().Say("Exiting..."),
				)

				Expect(suspendedVM.Start()).To(Succeed())
			})
		})

		Context("when starting the vm fails", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockRequirementsChecker.EXPECT().Check().Return(nil),
					mockUI.EXPECT().Say("Resuming VM..."),
					mockVBox.EXPECT().ResumeVM("some-vm").Return(errors.New("some-error")),
				)

				Expect(suspendedVM.Start()).To(MatchError("could not resume PCF Dev: some-error"))
			})
		})
	})

	Describe("Resume", func() {
		It("should start vm", func() {
			gomock.InOrder(
				mockRequirementsChecker.EXPECT().Check().Return(nil),
				mockUI.EXPECT().Say("Resuming VM..."),
				mockVBox.EXPECT().ResumeVM("some-vm").Return(nil),
				mockUI.EXPECT().Say("PCF Dev is now running"),
			)

			Expect(suspendedVM.Resume()).To(Succeed())
		})

		Context("when the system does not meet requirements and the user accepts to continue", func() {
			It("should print a warning and prompt for the response to continue", func() {
				gomock.InOrder(
					mockRequirementsChecker.EXPECT().Check().Return(errors.New("some-message")),
					mockUI.EXPECT().Confirm("Less than 3 GB of memory detected, continue (y/N): ").Return(true),
					mockUI.EXPECT().Say("Resuming VM..."),
					mockVBox.EXPECT().ResumeVM("some-vm").Return(nil),
					mockUI.EXPECT().Say("PCF Dev is now running"),
				)

				Expect(suspendedVM.Resume()).To(Succeed())
			})
		})

		Context("when the system does not meet requirements and the user declines to continue", func() {
			It("should print a warning and prompt for the response to continue", func() {
				gomock.InOrder(
					mockRequirementsChecker.EXPECT().Check().Return(errors.New("some-message")),
					mockUI.EXPECT().Confirm("Less than 3 GB of memory detected, continue (y/N): ").Return(false),
					mockUI.EXPECT().Say("Exiting..."),
				)

				Expect(suspendedVM.Resume()).To(Succeed())
			})
		})

		Context("when starting the vm fails", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockRequirementsChecker.EXPECT().Check().Return(nil),
					mockUI.EXPECT().Say("Resuming VM..."),
					mockVBox.EXPECT().ResumeVM("some-vm").Return(errors.New("some-error")),
				)

				Expect(suspendedVM.Resume()).To(MatchError("could not resume PCF Dev: some-error"))
			})
		})
	})

	Describe("Status", func() {
		It("should say Suspended", func() {
			mockUI.EXPECT().Say("Suspended")

			suspendedVM.Status()
		})
	})

	Describe("Destroy", func() {
		It("should destroy the vm", func() {
			mockVBox.EXPECT().DestroyVM("some-vm").Return(nil)

			Expect(suspendedVM.Destroy()).To(Succeed())
		})
	})
})

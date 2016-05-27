package vm_test

import (
	"errors"

	"github.com/golang/mock/gomock"
	"github.com/pivotal-cf/pcfdev-cli/user"
	"github.com/pivotal-cf/pcfdev-cli/vm"
	"github.com/pivotal-cf/pcfdev-cli/vm/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Not Created", func() {
	var (
		mockCtrl     *gomock.Controller
		mockUI       *mocks.MockUI
		mockVBox     *mocks.MockVBox
		mockBuilder  *mocks.MockBuilder
		mockStopped  *mocks.MockVM
		notCreatedVM vm.NotCreated
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockUI = mocks.NewMockUI(mockCtrl)
		mockVBox = mocks.NewMockVBox(mockCtrl)
		mockBuilder = mocks.NewMockBuilder(mockCtrl)
		mockStopped = mocks.NewMockVM(mockCtrl)

		notCreatedVM = vm.NotCreated{
			Name: "some-vm",

			VBox:    mockVBox,
			UI:      mockUI,
			Builder: mockBuilder,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("Stop", func() {
		Context("when no conflicting vm is present", func() {
			It("should print a message", func() {
				gomock.InOrder(
					mockVBox.EXPECT().ConflictingVMPresent("some-vm").Return(false, nil),
					mockUI.EXPECT().Say("PCF Dev VM has not been created"),
				)

				notCreatedVM.Stop()
			})
		})
		Context("when conflicting vm is present", func() {
			It("should return an error", func() {
				mockVBox.EXPECT().ConflictingVMPresent("some-vm").Return(true, nil)

				Expect(notCreatedVM.Stop()).To(MatchError("old version of PCF Dev already running"))
			})
		})
		Context("when there is an error seeing if there is an conflicting VM present", func() {
			It("should return an error", func() {
				mockVBox.EXPECT().ConflictingVMPresent("some-vm").Return(false, errors.New("some-error"))

				Expect(notCreatedVM.Stop()).To(MatchError("failed to stop vm: some-error"))
			})
		})
	})

	Describe("Start", func() {
		var home string

		BeforeEach(func() {
			var err error
			home, err = user.GetHome()
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when no conflicting vm is present", func() {
			It("should import and start the vm", func() {
				gomock.InOrder(
					mockVBox.EXPECT().ConflictingVMPresent("some-vm").Return(false, nil),
					mockUI.EXPECT().Say("Importing VM..."),
					mockVBox.EXPECT().ImportVM("some-vm").Return(nil),
					mockUI.EXPECT().Say("PCF Dev is now imported to Virtualbox"),
					mockBuilder.EXPECT().VM("some-vm").Return(mockStopped, nil),
					mockStopped.EXPECT().Start(),
				)

				notCreatedVM.Start()
			})
		})

		Context("when there is an error seeing if conflicting vms are present", func() {
			It("should return an error", func() {
				mockVBox.EXPECT().ConflictingVMPresent("some-vm").Return(false, errors.New("some-error"))

				Expect(notCreatedVM.Start()).To(MatchError("could not start PCF Dev: some-error"))
			})
		})

		Context("when there are conflicting vms present", func() {
			It("should return an error", func() {
				mockVBox.EXPECT().ConflictingVMPresent("some-vm").Return(true, nil)

				Expect(notCreatedVM.Start()).To(MatchError("old version of PCF Dev already running"))
			})
		})

		Context("when there is an error importing the VM", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockVBox.EXPECT().ConflictingVMPresent("some-vm").Return(false, nil),
					mockUI.EXPECT().Say("Importing VM..."),
					mockVBox.EXPECT().ImportVM("some-vm").Return(errors.New("some-error")),
				)

				Expect(notCreatedVM.Start()).To(MatchError("could not start PCF Dev: some-error"))
			})
		})

		Context("when there is an error constructing a stopped VM", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockVBox.EXPECT().ConflictingVMPresent("some-vm").Return(false, nil),
					mockUI.EXPECT().Say("Importing VM..."),
					mockVBox.EXPECT().ImportVM("some-vm").Return(nil),
					mockUI.EXPECT().Say("PCF Dev is now imported to Virtualbox"),
					mockBuilder.EXPECT().VM("some-vm").Return(nil, errors.New("some-error")),
				)

				Expect(notCreatedVM.Start()).To(MatchError("could not start PCF Dev: some-error"))
			})
		})

		Context("when there is an error starting the stopped VM", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockVBox.EXPECT().ConflictingVMPresent("some-vm").Return(false, nil),
					mockUI.EXPECT().Say("Importing VM..."),
					mockVBox.EXPECT().ImportVM("some-vm").Return(nil),
					mockUI.EXPECT().Say("PCF Dev is now imported to Virtualbox"),
					mockBuilder.EXPECT().VM("some-vm").Return(mockStopped, nil),
					mockStopped.EXPECT().Start().Return(errors.New("some-error")),
				)

				Expect(notCreatedVM.Start()).To(MatchError("could not start PCF Dev: some-error"))
			})
		})
	})

	Describe("Status", func() {
		It("should say Not Created", func() {
			mockUI.EXPECT().Say("Not Created")

			notCreatedVM.Status()
		})
	})
})

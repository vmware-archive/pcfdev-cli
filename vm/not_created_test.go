package vm_test

import (
	"errors"

	"github.com/golang/mock/gomock"
	conf "github.com/pivotal-cf/pcfdev-cli/config"
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
		config       *conf.VMConfig
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockUI = mocks.NewMockUI(mockCtrl)
		mockVBox = mocks.NewMockVBox(mockCtrl)
		mockBuilder = mocks.NewMockBuilder(mockCtrl)
		mockStopped = mocks.NewMockVM(mockCtrl)
		config = &conf.VMConfig{DesiredMemory: uint64(3072)}

		notCreatedVM = vm.NotCreated{
			Name: "some-vm",

			VBox:    mockVBox,
			UI:      mockUI,
			Builder: mockBuilder,
			Config:  config,
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

				Expect(notCreatedVM.Stop()).To(MatchError("failed to stop VM: some-error"))
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
					mockVBox.EXPECT().ImportVM("some-vm", &conf.VMConfig{DesiredMemory: uint64(3072)}).Return(nil),
					mockBuilder.EXPECT().VM("some-vm", &conf.VMConfig{DesiredMemory: uint64(3072)}).Return(mockStopped, nil),
					mockStopped.EXPECT().Start(),
				)

				notCreatedVM.Start()
			})
		})

		Context("when there is an error seeing if conflicting vms are present", func() {
			It("should return an error", func() {
				mockVBox.EXPECT().ConflictingVMPresent("some-vm").Return(false, errors.New("some-error"))

				Expect(notCreatedVM.Start()).To(MatchError("failed to start VM: some-error"))
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
					mockVBox.EXPECT().ImportVM("some-vm", &conf.VMConfig{DesiredMemory: uint64(3072)}).Return(errors.New("some-error")),
				)

				Expect(notCreatedVM.Start()).To(MatchError("failed to import VM: some-error"))
			})
		})

		Context("when there is an error constructing a stopped VM", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockVBox.EXPECT().ConflictingVMPresent("some-vm").Return(false, nil),
					mockUI.EXPECT().Say("Importing VM..."),
					mockVBox.EXPECT().ImportVM("some-vm", &conf.VMConfig{DesiredMemory: uint64(3072)}).Return(nil),
					mockBuilder.EXPECT().VM("some-vm", &conf.VMConfig{DesiredMemory: uint64(3072)}).Return(nil, errors.New("some-error")),
				)

				Expect(notCreatedVM.Start()).To(MatchError("failed to start VM: some-error"))
			})
		})

		Context("when there is an error starting the stopped VM", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockVBox.EXPECT().ConflictingVMPresent("some-vm").Return(false, nil),
					mockUI.EXPECT().Say("Importing VM..."),
					mockVBox.EXPECT().ImportVM("some-vm", &conf.VMConfig{DesiredMemory: uint64(3072)}).Return(nil),
					mockBuilder.EXPECT().VM("some-vm", &conf.VMConfig{DesiredMemory: uint64(3072)}).Return(mockStopped, nil),
					mockStopped.EXPECT().Start().Return(errors.New("some-error")),
				)

				Expect(notCreatedVM.Start()).To(MatchError("failed to start VM: some-error"))
			})
		})
	})

	Describe("Status", func() {
		It("should say Not Created", func() {
			mockUI.EXPECT().Say("Not Created")

			notCreatedVM.Status()
		})
	})

	Describe("Suspend", func() {
		It("should say message", func() {
			mockUI.EXPECT().Say("No VM running, cannot suspend.")

			Expect(notCreatedVM.Suspend()).To(Succeed())
		})
	})

	Describe("Resume", func() {
		It("should say message", func() {
			mockUI.EXPECT().Say("No VM suspended, cannot resume.")

			Expect(notCreatedVM.Resume()).To(Succeed())
		})
	})

	Describe("Config", func() {
		It("should return the config", func() {
			Expect(notCreatedVM.GetConfig()).To(BeIdenticalTo(config))
		})
	})
})

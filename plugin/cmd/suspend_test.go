package cmd_test

import (
	"errors"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/plugin/cmd"
	"github.com/pivotal-cf/pcfdev-cli/plugin/cmd/mocks"
	vmMocks "github.com/pivotal-cf/pcfdev-cli/vm/mocks"
)

var _ = Describe("SuspendCmd", func() {
	var (
		suspendCmd    *cmd.SuspendCmd
		mockCtrl      *gomock.Controller
		mockVMBuilder *mocks.MockVMBuilder
		mockVBox      *mocks.MockVBox
		mockVM        *vmMocks.MockVM
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockVMBuilder = mocks.NewMockVMBuilder(mockCtrl)
		mockVBox = mocks.NewMockVBox(mockCtrl)
		mockVM = vmMocks.NewMockVM(mockCtrl)
		suspendCmd = &cmd.SuspendCmd{
			VBox:      mockVBox,
			VMBuilder: mockVMBuilder,
			Config: &config.Config{
				DefaultVMName: "some-default-vm-name",
			},
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})
	Describe("Parse", func() {
		Context("when the correct number of arguments are passed", func() {
			It("should succeed", func() {
				Expect(suspendCmd.Parse([]string{})).To(Succeed())
			})
		})
		Context("when the wrong number of arguments are passed", func() {
			It("should fail", func() {
				Expect(suspendCmd.Parse([]string{"some-bad-arg"})).NotTo(Succeed())
			})
		})
		Context("when an unknown flag is passed", func() {
			It("should fail", func() {
				Expect(suspendCmd.Parse([]string{"--some-bad-flag"})).NotTo(Succeed())
			})
		})
	})

	Describe("Run", func() {
		Context("when the default VM is present", func() {
			It("should suspend the VM", func() {
				gomock.InOrder(
					mockVBox.EXPECT().GetVMName().Return("some-default-vm-name", nil),
					mockVMBuilder.EXPECT().VM("some-default-vm-name").Return(mockVM, nil),
					mockVM.EXPECT().Suspend(),
				)

				Expect(suspendCmd.Run()).To(Succeed())
			})
		})

		Context("when the custom vm is present", func() {
			It("should suspend the VM", func() {
				gomock.InOrder(
					mockVBox.EXPECT().GetVMName().Return("pcfdev-custom", nil),
					mockVMBuilder.EXPECT().VM("pcfdev-custom").Return(mockVM, nil),
					mockVM.EXPECT().Suspend(),
				)

				Expect(suspendCmd.Run()).To(Succeed())
			})
		})

		Context("when there is no vm present", func() {
			It("should suspend the default VM", func() {
				gomock.InOrder(
					mockVBox.EXPECT().GetVMName().Return("", nil),
					mockVMBuilder.EXPECT().VM("some-default-vm-name").Return(mockVM, nil),
					mockVM.EXPECT().Suspend(),
				)

				Expect(suspendCmd.Run()).To(Succeed())
			})
		})

		Context("when there is an old vm present", func() {
			It("should tell the user to destroy pcfdev", func() {
				mockVBox.EXPECT().GetVMName().Return("some-old-vm-name", nil)

				Expect(suspendCmd.Run()).To(MatchError("old version of PCF Dev already running, please run `cf dev destroy` to continue"))
			})
		})

		Context("when there is an error checking for an old vm present", func() {
			It("should return the error", func() {
				mockVBox.EXPECT().GetVMName().Return("", errors.New("some-error"))

				Expect(suspendCmd.Run()).To(MatchError("some-error"))
			})
		})

		Context("when it fails to get VM", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockVBox.EXPECT().GetVMName().Return("", nil),
					mockVMBuilder.EXPECT().VM("some-default-vm-name").Return(nil, errors.New("some-error")),
				)

				Expect(suspendCmd.Run()).To(MatchError("some-error"))
			})
		})

		Context("when it fails to suspend VM", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockVBox.EXPECT().GetVMName().Return("", nil),
					mockVMBuilder.EXPECT().VM("some-default-vm-name").Return(mockVM, nil),
					mockVM.EXPECT().Suspend().Return(errors.New("some-error")),
				)

				Expect(suspendCmd.Run()).To(MatchError("some-error"))
			})
		})
	})
})

package cmd_test

import (
	"errors"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/plugin/cmd"
	"github.com/pivotal-cf/pcfdev-cli/plugin/cmd/mocks"
	"github.com/pivotal-cf/pcfdev-cli/vm"
	vmMocks "github.com/pivotal-cf/pcfdev-cli/vm/mocks"
)

var _ = Describe("TrustCmd", func() {
	var (
		trustCmd      *cmd.TrustCmd
		mockCtrl      *gomock.Controller
		mockVBox      *mocks.MockVBox
		mockVMBuilder *mocks.MockVMBuilder
		mockVM        *vmMocks.MockVM
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockVBox = mocks.NewMockVBox(mockCtrl)
		mockVMBuilder = mocks.NewMockVMBuilder(mockCtrl)
		mockVM = vmMocks.NewMockVM(mockCtrl)
		trustCmd = &cmd.TrustCmd{
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
		Context("when flags are passed", func() {
			It("should set start options", func() {
				Expect(trustCmd.Parse([]string{
					"-p",
				})).To(Succeed())

				Expect(trustCmd.Opts.PrintCA).To(BeTrue())
			})
		})

		Context("when no flags are passed", func() {
			It("should not set start options", func() {
				Expect(trustCmd.Parse([]string{})).To(Succeed())

				Expect(trustCmd.Opts.PrintCA).To(BeFalse())
			})
		})

		Context("when the wrong number of arguments are passed", func() {
			It("should fail", func() {
				Expect(trustCmd.Parse([]string{"some-bad-arg"})).NotTo(Succeed())
			})
		})
		Context("when an unknown flag is passed", func() {
			It("should fail", func() {
				Expect(trustCmd.Parse([]string{"--some-bad-flag"})).NotTo(Succeed())
			})
		})
	})

	Describe("Run", func() {
		It("should call Trust on the VM", func() {
			trustOpts := &vm.StartOpts{}
			trustCmd.Opts = trustOpts
			gomock.InOrder(
				mockVBox.EXPECT().GetVMName().Return("some-default-vm-name", nil),
				mockVMBuilder.EXPECT().VM("some-default-vm-name").Return(mockVM, nil),
				mockVM.EXPECT().Trust(trustOpts),
			)

			Expect(trustCmd.Run()).To(Succeed())
		})

		Context("when flags are passed in", func() {
			It("should call Trust on the VM with the arguments passed in", func() {
				trustOpts := &vm.StartOpts{
					PrintCA: true,
				}
				trustCmd.Opts = trustOpts

				gomock.InOrder(
					mockVBox.EXPECT().GetVMName().Return("some-default-vm-name", nil),
					mockVMBuilder.EXPECT().VM("some-default-vm-name").Return(mockVM, nil),
					mockVM.EXPECT().Trust(trustOpts),
				)

				Expect(trustCmd.Run()).To(Succeed())
			})
		})

		Context("when there is an error getting the VM name", func() {
			It("should return the error", func() {
				mockVBox.EXPECT().GetVMName().Return("", errors.New("some-error"))

				Expect(trustCmd.Run()).To(MatchError("some-error"))
			})
		})

		Context("when there is an error building the VM", func() {
			It("should return the error", func() {
				gomock.InOrder(
					mockVBox.EXPECT().GetVMName().Return("some-default-vm-name", nil),
					mockVMBuilder.EXPECT().VM("some-default-vm-name").Return(nil, errors.New("some-error")),
				)

				Expect(trustCmd.Run()).To(MatchError("some-error"))
			})
		})

		Context("when there is an error trusting the cert", func() {
			It("should return the error", func() {
				trustOpts := &vm.StartOpts{}
				trustCmd.Opts = trustOpts
				gomock.InOrder(
					mockVBox.EXPECT().GetVMName().Return("some-default-vm-name", nil),
					mockVMBuilder.EXPECT().VM("some-default-vm-name").Return(mockVM, nil),
					mockVM.EXPECT().Trust(trustOpts).Return(errors.New("some-error")),
				)

				Expect(trustCmd.Run()).To(MatchError("some-error"))
			})
		})
	})
})

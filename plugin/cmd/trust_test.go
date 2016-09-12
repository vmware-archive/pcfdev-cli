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

var _ = Describe("StatusCmd", func() {
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
		Context("when the correct number of arguments are passed", func() {
			It("should succeed", func() {
				Expect(trustCmd.Parse([]string{})).To(Succeed())
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
			gomock.InOrder(
				mockVBox.EXPECT().GetVMName().Return("some-default-vm-name", nil),
				mockVMBuilder.EXPECT().VM("some-default-vm-name").Return(mockVM, nil),
				mockVM.EXPECT().Trust(),
			)

			Expect(trustCmd.Run()).To(Succeed())
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
				gomock.InOrder(
					mockVBox.EXPECT().GetVMName().Return("some-default-vm-name", nil),
					mockVMBuilder.EXPECT().VM("some-default-vm-name").Return(mockVM, nil),
					mockVM.EXPECT().Trust().Return(errors.New("some-error")),
				)

				Expect(trustCmd.Run()).To(MatchError("some-error"))
			})
		})
	})
})

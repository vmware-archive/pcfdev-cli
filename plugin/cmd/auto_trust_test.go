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

var _ = Describe("AutoTrustCmd", func() {
	var (
		autoTrustCmd  *cmd.AutoTrustCmd
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
		autoTrustCmd = &cmd.AutoTrustCmd{
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

	Describe("Run", func() {
		It("should call Trust on the VM", func() {
			gomock.InOrder(
				mockVBox.EXPECT().GetVMName().Return("some-default-vm-name", nil),
				mockVMBuilder.EXPECT().VM("some-default-vm-name").Return(mockVM, nil),
				mockVM.EXPECT().Trust(&vm.StartOpts{}),
			)

			Expect(autoTrustCmd.Run()).To(Succeed())
		})

		Context("when there is an error getting the VM name", func() {
			It("should return the error", func() {
				mockVBox.EXPECT().GetVMName().Return("", errors.New("some-error"))

				Expect(autoTrustCmd.Run()).To(MatchError("some-error"))
			})
		})

		Context("when there is an error building the VM", func() {
			It("should return the error", func() {
				gomock.InOrder(
					mockVBox.EXPECT().GetVMName().Return("some-default-vm-name", nil),
					mockVMBuilder.EXPECT().VM("some-default-vm-name").Return(nil, errors.New("some-error")),
				)

				Expect(autoTrustCmd.Run()).To(MatchError("some-error"))
			})
		})

		Context("when there is an error trusting the cert", func() {
			It("should return the error", func() {
				gomock.InOrder(
					mockVBox.EXPECT().GetVMName().Return("some-default-vm-name", nil),
					mockVMBuilder.EXPECT().VM("some-default-vm-name").Return(mockVM, nil),
					mockVM.EXPECT().Trust(&vm.StartOpts{}).Return(errors.New("some-error")),
				)

				Expect(autoTrustCmd.Run()).To(MatchError("some-error"))
			})
		})
	})
})

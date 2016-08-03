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

var _ = Describe("ProvisionCmd", func() {
	var (
		provisionCmd  *cmd.ProvisionCmd
		mockCtrl      *gomock.Controller
		mockVBox      *mocks.MockVBox
		mockVM        *vmMocks.MockVM
		mockVMBuilder *mocks.MockVMBuilder
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockVBox = mocks.NewMockVBox(mockCtrl)
		mockVM = vmMocks.NewMockVM(mockCtrl)
		mockVMBuilder = mocks.NewMockVMBuilder(mockCtrl)
		provisionCmd = &cmd.ProvisionCmd{
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
				provisionCommand := &cmd.ProvisionCmd{}
				Expect(provisionCommand.Parse([]string{})).To(Succeed())
			})
		})
		Context("when the wrong number of arguments are passed", func() {
			It("should fail", func() {
				provisionCommand := &cmd.ProvisionCmd{}
				Expect(provisionCommand.Parse([]string{"some-bad-arg"})).NotTo(Succeed())
			})
		})
		Context("when an unknown flag is passed", func() {
			It("should fail", func() {
				provisionCommand := &cmd.ProvisionCmd{}
				Expect(provisionCommand.Parse([]string{"--some-bad-flag"})).NotTo(Succeed())
			})
		})
	})

	Describe("Run", func() {
		It("should provision the VM", func() {
			gomock.InOrder(
				mockVBox.EXPECT().GetVMName().Return("some-default-vm-name", nil),
				mockVMBuilder.EXPECT().VM("some-default-vm-name").Return(mockVM, nil),
				mockVM.EXPECT().Provision(),
			)

			Expect(provisionCmd.Run()).To(Succeed())
		})

		Context("when there is an error", func() {
			It("should print the error message", func() {
				gomock.InOrder(
					mockVBox.EXPECT().GetVMName().Return("some-default-vm-name", nil),
					mockVMBuilder.EXPECT().VM("some-default-vm-name").Return(mockVM, nil),
					mockVM.EXPECT().Provision().Return(errors.New("some-error")),
				)

				Expect(provisionCmd.Run()).To(MatchError("some-error"))
			})
		})
	})
})

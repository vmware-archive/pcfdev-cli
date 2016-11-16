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

var _ = Describe("SSHCmd", func() {
	var (
		sshCmd        *cmd.SSHCmd
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
		sshCmd = &cmd.SSHCmd{
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
				Expect(sshCmd.Parse([]string{})).To(Succeed())
			})
		})
		Context("when the wrong number of arguments are passed", func() {
			It("should fail", func() {
				Expect(sshCmd.Parse([]string{"some-bad-arg"})).NotTo(Succeed())
			})
		})
		Context("when an unknown flag is passed", func() {
			It("should fail", func() {
				Expect(sshCmd.Parse([]string{"--some-bad-flag"})).NotTo(Succeed())
			})
		})
		Context("when -c flag is passed", func() {
			It("should parse command", func() {
				parse := sshCmd.Parse([]string{"-c", "echo hello"})
				Expect(parse).To(Succeed())
				Expect(sshCmd.Opts.Command).To(Equal("echo hello"))
			})
		})
	})

	Describe("Run", func() {
		It("should call SSH on the VM", func() {
			sshCmd.Parse([]string{})
			opts := sshCmd.Opts
			gomock.InOrder(
				mockVBox.EXPECT().GetVMName().Return("some-default-vm-name", nil),
				mockVMBuilder.EXPECT().VM("some-default-vm-name").Return(mockVM, nil),
				mockVM.EXPECT().SSH(opts),
			)

			Expect(sshCmd.Run()).To(Succeed())
		})

		Context("when there is an error getting the VM name", func() {
			It("should return the error", func() {
				mockVBox.EXPECT().GetVMName().Return("", errors.New("some-error"))

				Expect(sshCmd.Run()).To(MatchError("some-error"))
			})
		})

		Context("when there is an error building the VM", func() {
			It("should return the error", func() {
				gomock.InOrder(
					mockVBox.EXPECT().GetVMName().Return("some-default-vm-name", nil),
					mockVMBuilder.EXPECT().VM("some-default-vm-name").Return(nil, errors.New("some-error")),
				)

				Expect(sshCmd.Run()).To(MatchError("some-error"))
			})
		})

		Context("when there is an error SSHing to PCF Dev", func() {
			It("should return the error", func() {
				sshCmd.Parse([]string{})
				opts := sshCmd.Opts
				gomock.InOrder(
					mockVBox.EXPECT().GetVMName().Return("some-default-vm-name", nil),
					mockVMBuilder.EXPECT().VM("some-default-vm-name").Return(mockVM, nil),
					mockVM.EXPECT().SSH(opts).Return(errors.New("some-error")),
				)

				Expect(sshCmd.Run()).To(MatchError("some-error"))
			})
		})

		Context("when -c flag is set", func() {
			It("should run the command passed", func() {
				sshCmd.Parse([]string{"-c", "echo hello"})
				opts := sshCmd.Opts
				gomock.InOrder(
					mockVBox.EXPECT().GetVMName().Return("some-default-vm-name", nil),
					mockVMBuilder.EXPECT().VM("some-default-vm-name").Return(mockVM, nil),
					mockVM.EXPECT().SSH(opts),
				)
				Expect(sshCmd.Run()).To(Succeed())
			})
		})
	})
})

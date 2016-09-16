package cmd_test

import (
	"errors"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/plugin/cmd"
	"github.com/pivotal-cf/pcfdev-cli/plugin/cmd/mocks"
)

var _ = Describe("DestroyCmd", func() {
	var (
		mockCtrl       *gomock.Controller
		mockUI         *mocks.MockUI
		mockVBox       *mocks.MockVBox
		mockFS         *mocks.MockFS
		mockUntrustCmd *mocks.MockCmd
		destroyCmd     *cmd.DestroyCmd
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockUI = mocks.NewMockUI(mockCtrl)
		mockVBox = mocks.NewMockVBox(mockCtrl)
		mockFS = mocks.NewMockFS(mockCtrl)
		mockUntrustCmd = mocks.NewMockCmd(mockCtrl)
		destroyCmd = &cmd.DestroyCmd{
			UI:         mockUI,
			VBox:       mockVBox,
			FS:         mockFS,
			UntrustCmd: mockUntrustCmd,
			Config: &config.Config{
				VMDir: "some-vm-dir",
			},
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("Parse", func() {
		Context("when the correct number of arguments are passed", func() {
			It("should succeed", func() {
				Expect(destroyCmd.Parse([]string{})).To(Succeed())
			})
		})
		Context("when the wrong number of arguments are passed", func() {
			It("should fail", func() {
				Expect(destroyCmd.Parse([]string{"some-bad-arg"})).NotTo(Succeed())
			})
		})
		Context("when an unknown flag is passed", func() {
			It("should fail", func() {
				Expect(destroyCmd.Parse([]string{"--some-bad-flag"})).NotTo(Succeed())
			})
		})
	})

	Describe("Run", func() {
		It("should destroy all PCF Dev VMs created by the CLI and the VM dir", func() {
			gomock.InOrder(
				mockUntrustCmd.EXPECT().Run(),
				mockVBox.EXPECT().DestroyPCFDevVMs(),
				mockUI.EXPECT().Say("PCF Dev VM has been destroyed."),
				mockFS.EXPECT().Remove("some-vm-dir"),
			)

			Expect(destroyCmd.Run()).To(Succeed())
		})

		Context("when there is an error destroying PCF Dev VMs", func() {
			It("should remove the VM dir and return an errpr", func() {
				gomock.InOrder(
					mockUntrustCmd.EXPECT().Run(),
					mockVBox.EXPECT().DestroyPCFDevVMs().Return(errors.New("some-error")),
					mockFS.EXPECT().Remove("some-vm-dir"),
				)

				Expect(destroyCmd.Run()).To(MatchError("error destroying PCF Dev VM: some-error"))
			})
		})

		Context("when there is an error removing the VM dir", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockUntrustCmd.EXPECT().Run(),
					mockVBox.EXPECT().DestroyPCFDevVMs(),
					mockUI.EXPECT().Say("PCF Dev VM has been destroyed."),
					mockFS.EXPECT().Remove("some-vm-dir").Return(errors.New("some-error")),
				)

				Expect(destroyCmd.Run()).To(MatchError("error removing some-vm-dir: some-error"))
			})
		})

		Context("when there is an error destroying PCF Dev VMs and removing the VM dir", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockUntrustCmd.EXPECT().Run(),
					mockVBox.EXPECT().DestroyPCFDevVMs().Return(errors.New("some-error")),
					mockFS.EXPECT().Remove("some-vm-dir").Return(errors.New("some-error")),
				)

				Expect(destroyCmd.Run()).To(MatchError("error destroying PCF Dev VM: some-error\nerror removing some-vm-dir: some-error"))
			})
		})

		Context("when there is an error deleting from the trust store", func() {
			It("should remove the VM dir and keep going and return an error", func() {
				gomock.InOrder(
					mockUntrustCmd.EXPECT().Run().Return(errors.New("some-error")),
					mockVBox.EXPECT().DestroyPCFDevVMs(),
					mockUI.EXPECT().Say("PCF Dev VM has been destroyed."),
					mockFS.EXPECT().Remove("some-vm-dir"),
				)

				Expect(destroyCmd.Run()).To(MatchError("error removing certificates from trust store: some-error"))
			})
		})
	})
})

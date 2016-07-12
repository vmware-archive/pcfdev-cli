package vm_test

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/golang/mock/gomock"
	conf "github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/vm"
	"github.com/pivotal-cf/pcfdev-cli/vm/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Recoverable", func() {
	var (
		mockCtrl    *gomock.Controller
		mockFS      *mocks.MockFS
		mockUI      *mocks.MockUI
		mockVBox    *mocks.MockVBox
		mockSSH     *mocks.MockSSH
		recoverable vm.Recoverable
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockVBox = mocks.NewMockVBox(mockCtrl)
		mockUI = mocks.NewMockUI(mockCtrl)
		mockFS = mocks.NewMockFS(mockCtrl)
		mockSSH = mocks.NewMockSSH(mockCtrl)

		recoverable = vm.Recoverable{
			UI:   mockUI,
			VBox: mockVBox,
			FS:   mockFS,
			SSH:  mockSSH,
			Config: &conf.Config{
				VMDir: "some-vm-dir",
			},
			VMConfig: &conf.VMConfig{
				Name:    "some-vm",
				Domain:  "some-domain",
				IP:      "some-ip",
				SSHPort: "some-port",
			},
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("Stop", func() {
		It("should stop the VM", func() {
			gomock.InOrder(
				mockUI.EXPECT().Say("Stopping VM..."),
				mockVBox.EXPECT().StopVM(recoverable.VMConfig),
				mockUI.EXPECT().Say("PCF Dev is now stopped."),
			)

			recoverable.Stop()
		})
	})

	Describe("VerifyStartOpts", func() {
		It("should say a message", func() {
			Expect(recoverable.VerifyStartOpts(
				&vm.StartOpts{},
			)).To(MatchError("PCF Dev is in an invalid state. Please run 'cf dev destroy' or 'cf dev stop' before attempting to start again"))
		})
	})

	Describe("Start", func() {
		It("should start vm", func() {
			mockUI.EXPECT().Failed("PCF Dev is in an invalid state. Please run 'cf dev destroy' or 'cf dev stop' before attempting to start again.")

			recoverable.Start(&vm.StartOpts{})
		})
	})

	Describe("Provision", func() {
		It("should provision the VM", func() {
			gomock.InOrder(
				mockFS.EXPECT().Exists(filepath.Join("some-vm-dir", "provision-options")).Return(true, nil),
				mockFS.EXPECT().Read(filepath.Join("some-vm-dir", "provision-options")).Return([]byte(`{"domain":"some-domain","ip":"some-ip","services":"some-service"}`), nil),
				mockUI.EXPECT().Say("Provisioning VM..."),
				mockSSH.EXPECT().RunSSHCommand("sudo -H /var/pcfdev/run some-domain some-ip some-service", "some-port", 5*time.Minute, os.Stdout, os.Stderr),
			)

			recoverable.Provision()
		})

		Context("when there is an error finding the provision config", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockFS.EXPECT().Exists(filepath.Join("some-vm-dir", "provision-options")).Return(false, errors.New("some-error")),
				)

				Expect(recoverable.Provision()).To(MatchError("failed to provision VM: missing provision configuration"))
			})
		})

		Context("when provision config is missing", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockFS.EXPECT().Exists(filepath.Join("some-vm-dir", "provision-options")).Return(false, nil),
				)

				Expect(recoverable.Provision()).To(MatchError("failed to provision VM: missing provision configuration"))
			})
		})

		Context("when there is an error reading the provision config", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockFS.EXPECT().Exists(filepath.Join("some-vm-dir", "provision-options")).Return(true, nil),
					mockFS.EXPECT().Read(filepath.Join("some-vm-dir", "provision-options")).Return([]byte{}, errors.New("some-error")),
				)

				Expect(recoverable.Provision()).To(MatchError("failed to provision VM: some-error"))
			})
		})

		Context("when there is an error parsing the provision config", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockFS.EXPECT().Exists(filepath.Join("some-vm-dir", "provision-options")).Return(true, nil),
					mockFS.EXPECT().Read(filepath.Join("some-vm-dir", "provision-options")).Return([]byte("some-bad-json"), nil),
				)

				Expect(recoverable.Provision()).To(MatchError(ContainSubstring(`failed to provision VM: invalid character 's'`)))
			})
		})
	})

	Describe("Status", func() {
		It("should return 'Stopped'", func() {
			Expect(recoverable.Status()).To(Equal("PCF Dev is in an invalid state. Please run 'cf dev destroy' or 'cf dev stop' before attempting to start again."))
		})
	})

	Describe("Suspend", func() {
		It("should say a message", func() {
			mockUI.EXPECT().Failed("PCF Dev is in an invalid state. Please run 'cf dev destroy' or 'cf dev stop' before attempting to start again.")

			recoverable.Suspend()
		})
	})

	Describe("Resume", func() {
		It("should say a message", func() {
			mockUI.EXPECT().Failed("PCF Dev is in an invalid state. Please run 'cf dev destroy' or 'cf dev stop' before attempting to start again.")

			recoverable.Resume()
		})
	})
})

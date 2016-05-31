package vm_test

import (
	"errors"
	"os"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/pivotal-cf/pcfdev-cli/vm"
	"github.com/pivotal-cf/pcfdev-cli/vm/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Stopped", func() {
	var (
		mockCtrl  *gomock.Controller
		mockUI    *mocks.MockUI
		mockVBox  *mocks.MockVBox
		mockSSH   *mocks.MockSSH
		stoppedVM vm.Stopped
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockUI = mocks.NewMockUI(mockCtrl)
		mockVBox = mocks.NewMockVBox(mockCtrl)
		mockSSH = mocks.NewMockSSH(mockCtrl)

		stoppedVM = vm.Stopped{
			Name:    "some-vm",
			Domain:  "some-domain",
			IP:      "some-ip",
			SSHPort: "some-port",

			VBox: mockVBox,
			UI:   mockUI,
			SSH:  mockSSH,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("Stop", func() {
		It("should say a message", func() {
			mockUI.EXPECT().Say("PCF Dev is stopped")
			stoppedVM.Stop()
		})
	})

	Describe("Start", func() {
		It("should start vm", func() {
			gomock.InOrder(
				mockUI.EXPECT().Say("Starting VM..."),
				mockVBox.EXPECT().StartVM("some-vm", "some-ip", "some-port", "some-domain").Return(nil),
				mockUI.EXPECT().Say("Provisioning VM..."),
				mockSSH.EXPECT().RunSSHCommand("sudo /var/pcfdev/run some-domain some-ip '$2a$04$EpJtIJ8w6hfCwbKYBkn3t.GCY18Pk6s7yN66y37fSJlLuDuMkdHtS'", "some-port", 2*time.Minute, os.Stdout, os.Stderr),
				mockUI.EXPECT().Say("PCF Dev is now running"),
			)

			stoppedVM.Start()
		})

		Context("when starting the vm fails", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Starting VM..."),
					mockVBox.EXPECT().StartVM("some-vm", "some-ip", "some-port", "some-domain").Return(errors.New("some-error")),
				)

				Expect(stoppedVM.Start()).To(MatchError("could not start PCF Dev: some-error"))
			})
		})

		Context("when provisioning the vm fails", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Starting VM..."),
					mockVBox.EXPECT().StartVM("some-vm", "some-ip", "some-port", "some-domain").Return(nil),
					mockUI.EXPECT().Say("Provisioning VM..."),
					mockSSH.EXPECT().RunSSHCommand("sudo /var/pcfdev/run some-domain some-ip '$2a$04$EpJtIJ8w6hfCwbKYBkn3t.GCY18Pk6s7yN66y37fSJlLuDuMkdHtS'", "some-port", 2*time.Minute, os.Stdout, os.Stderr).Return(errors.New("some-error")),
				)

				Expect(stoppedVM.Start()).To(MatchError("failed to provision vm: some-error"))
			})
		})
	})

	Describe("Status", func() {
		It("should say Stopped", func() {
			mockUI.EXPECT().Say("Stopped")

			stoppedVM.Status()
		})
	})

	Describe("Destroy", func() {
		It("should destroy the vm", func() {
			mockVBox.EXPECT().DestroyVM("some-vm").Return(nil)

			Expect(stoppedVM.Destroy()).To(Succeed())
		})
	})

	Describe("Suspend", func() {
		It("should say a message", func() {
			mockUI.EXPECT().Say("Your VM is currently stopped and cannot be suspended.")

			Expect(stoppedVM.Suspend()).To(Succeed())
		})
	})

	Describe("Resume", func() {
		It("should say a message", func() {
			mockUI.EXPECT().Say("Your VM is currently stopped. Only a suspended VM can be resumed.")

			Expect(stoppedVM.Resume()).To(Succeed())
		})
	})
})

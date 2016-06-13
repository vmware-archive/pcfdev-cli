package vm_test

import (
	"errors"
	"os"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/vm"
	"github.com/pivotal-cf/pcfdev-cli/vm/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Aborted", func() {
	var (
		mockCtrl  *gomock.Controller
		mockUI    *mocks.MockUI
		mockVBox  *mocks.MockVBox
		mockSSH   *mocks.MockSSH
		abortedVM vm.Aborted
		conf      *config.Config
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockUI = mocks.NewMockUI(mockCtrl)
		mockVBox = mocks.NewMockVBox(mockCtrl)
		mockSSH = mocks.NewMockSSH(mockCtrl)
		conf = &config.Config{}

		abortedVM = vm.Aborted{
			Name:    "some-vm",
			Domain:  "some-domain",
			IP:      "some-ip",
			SSHPort: "some-port",

			VBox:   mockVBox,
			UI:     mockUI,
			SSH:    mockSSH,
			Config: conf,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("Stop", func() {
		It("should say a message", func() {
			mockUI.EXPECT().Say("Your VM is currently aborted and cannot be stopped.")
			abortedVM.Stop()
		})
	})

	Describe("VerifyStartOpts", func() {
		Context("when desired memory is passed", func() {
			It("should return an error", func() {
				Expect(abortedVM.VerifyStartOpts(&vm.StartOpts{
					Memory: 4000,
				})).To(MatchError("memory cannot be changed once the vm has been created"))
			})
		})

		Context("when desired cores is passed", func() {
			It("should return an error", func() {
				Expect(abortedVM.VerifyStartOpts(&vm.StartOpts{
					CPUs: 2,
				})).To(MatchError("cores cannot be changed once the vm has been created"))
			})
		})

		Context("when no opts are passed", func() {
			Context("when free memory is greater than or equal to the VM's memory", func() {
				It("should succeed", func() {
					conf.FreeMemory = uint64(3000)
					abortedVM.Memory = uint64(2000)
					Expect(abortedVM.VerifyStartOpts(&vm.StartOpts{})).To(Succeed())
				})
			})

			Context("when free memory is less than the VM's memory", func() {
				Context("when the user accepts to continue", func() {
					It("should succeed", func() {
						conf.FreeMemory = uint64(2000)
						abortedVM.Memory = uint64(3000)

						mockUI.EXPECT().Confirm("Less than 3000 MB of free memory detected, continue (y/N): ").Return(true)

						Expect(abortedVM.VerifyStartOpts(&vm.StartOpts{})).To(Succeed())
					})
				})

				Context("when the user declines to continue", func() {
					It("should return an error", func() {
						conf.FreeMemory = uint64(2000)
						abortedVM.Memory = uint64(3000)

						mockUI.EXPECT().Confirm("Less than 3000 MB of free memory detected, continue (y/N): ").Return(false)

						Expect(abortedVM.VerifyStartOpts(&vm.StartOpts{})).To(MatchError("user declined to continue, exiting"))
					})
				})
			})
		})
	})

	Describe("Start", func() {
		It("should start vm", func() {
			gomock.InOrder(
				mockUI.EXPECT().Say("Starting VM..."),
				mockVBox.EXPECT().StartVM("some-vm", "some-ip", "some-port", "some-domain").Return(nil),
				mockUI.EXPECT().Say("Provisioning VM..."),
				mockSSH.EXPECT().RunSSHCommand("sudo /var/pcfdev/run some-domain some-ip '$2a$04$EpJtIJ8w6hfCwbKYBkn3t.GCY18Pk6s7yN66y37fSJlLuDuMkdHtS'", "some-port", 2*time.Minute, os.Stdout, os.Stderr),
			)

			abortedVM.Start(&vm.StartOpts{})
		})

		Context("when starting the vm fails", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Starting VM..."),
					mockVBox.EXPECT().StartVM("some-vm", "some-ip", "some-port", "some-domain").Return(errors.New("some-error")),
				)

				Expect(abortedVM.Start(&vm.StartOpts{})).To(MatchError("failed to start VM: some-error"))
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

				Expect(abortedVM.Start(&vm.StartOpts{})).To(MatchError("failed to provision VM: some-error"))
			})
		})
	})

	Describe("Status", func() {
		It("should return 'Aborted'", func() {
			Expect(abortedVM.Status()).To(Equal("Aborted"))
		})
	})

	Describe("Destroy", func() {
		It("should destroy the vm", func() {
			mockVBox.EXPECT().DestroyVM("some-vm").Return(nil)

			Expect(abortedVM.Destroy()).To(Succeed())
		})
	})

	Describe("Suspend", func() {
		It("should say a message", func() {
			mockUI.EXPECT().Say("Your VM is currently aborted and cannot be suspended.")

			Expect(abortedVM.Suspend()).To(Succeed())
		})
	})

	Describe("Resume", func() {
		It("should say a message", func() {
			mockUI.EXPECT().Say("Your VM is currently aborted. Only a suspended VM can be resumed.")

			Expect(abortedVM.Resume()).To(Succeed())
		})
	})
})

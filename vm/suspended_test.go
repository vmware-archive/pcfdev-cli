package vm_test

import (
	"errors"

	"github.com/golang/mock/gomock"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/vm"
	"github.com/pivotal-cf/pcfdev-cli/vm/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Suspended", func() {
	var (
		mockCtrl    *gomock.Controller
		mockUI      *mocks.MockUI
		mockVBox    *mocks.MockVBox
		suspendedVM vm.Suspended
		conf        *config.Config
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockUI = mocks.NewMockUI(mockCtrl)
		mockVBox = mocks.NewMockVBox(mockCtrl)
		conf = &config.Config{}

		suspendedVM = vm.Suspended{
			Name:    "some-vm",
			Domain:  "some-domain",
			IP:      "some-ip",
			SSHPort: "some-port",
			Config:  conf,

			VBox: mockVBox,
			UI:   mockUI,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("Suspend", func() {
		It("should say a message", func() {
			mockUI.EXPECT().Say("Your VM is suspended.")
			Expect(suspendedVM.Suspend()).To(Succeed())
		})
	})

	Describe("Stop", func() {
		It("should say a message", func() {
			mockUI.EXPECT().Say("Your VM is currently suspended. You must resume your VM with `cf dev resume` to shut it down.")
			Expect(suspendedVM.Stop()).To(Succeed())
		})
	})

	Describe("Start", func() {
		It("should start vm", func() {
			gomock.InOrder(
				mockUI.EXPECT().Say("Resuming VM..."),
				mockVBox.EXPECT().ResumeVM("some-vm").Return(nil),
			)

			Expect(suspendedVM.Start(&vm.StartOpts{})).To(Succeed())
		})

		Context("when starting the vm fails", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Resuming VM..."),
					mockVBox.EXPECT().ResumeVM("some-vm").Return(errors.New("some-error")),
				)

				Expect(suspendedVM.Start(&vm.StartOpts{})).To(MatchError("failed to resume VM: some-error"))
			})
		})
	})

	Describe("VerifyStartOpts", func() {
		Context("when desired memory is passed", func() {
			It("should return an error", func() {
				Expect(suspendedVM.VerifyStartOpts(&vm.StartOpts{
					Memory: 4000,
				})).To(MatchError("memory cannot be changed once the vm has been created"))
			})
		})

		Context("when desired cores is passed", func() {
			It("should return an error", func() {
				Expect(suspendedVM.VerifyStartOpts(&vm.StartOpts{
					CPUs: 2,
				})).To(MatchError("cores cannot be changed once the vm has been created"))
			})
		})

		Context("when no opts are passed", func() {
			Context("when free memory is greater than or equal to the VM's memory", func() {
				It("should succeed", func() {
					conf.FreeMemory = uint64(3000)
					suspendedVM.Memory = uint64(2000)
					Expect(suspendedVM.VerifyStartOpts(&vm.StartOpts{})).To(Succeed())
				})
			})

			Context("when free memory is less than the VM's memory", func() {
				Context("when the user accepts to continue", func() {
					It("should succeed", func() {
						conf.FreeMemory = uint64(2000)
						suspendedVM.Memory = uint64(3000)

						mockUI.EXPECT().Confirm("Less than 3000 MB of free memory detected, continue (y/N): ").Return(true)

						Expect(suspendedVM.VerifyStartOpts(&vm.StartOpts{})).To(Succeed())
					})
				})

				Context("when the user declines to continue", func() {
					It("should return an error", func() {
						conf.FreeMemory = uint64(2000)
						suspendedVM.Memory = uint64(3000)

						mockUI.EXPECT().Confirm("Less than 3000 MB of free memory detected, continue (y/N): ").Return(false)

						Expect(suspendedVM.VerifyStartOpts(&vm.StartOpts{})).To(MatchError("user declined to continue, exiting"))
					})
				})
			})
		})
	})

	Describe("Resume", func() {
		It("should start vm", func() {
			conf.FreeMemory = uint64(3000)
			suspendedVM.Memory = uint64(2000)
			gomock.InOrder(
				mockUI.EXPECT().Say("Resuming VM..."),
				mockVBox.EXPECT().ResumeVM("some-vm").Return(nil),
			)

			Expect(suspendedVM.Resume()).To(Succeed())
		})

		Context("when starting the vm fails", func() {
			It("should return an error", func() {
				conf.FreeMemory = uint64(3000)
				suspendedVM.Memory = uint64(2000)
				gomock.InOrder(
					mockUI.EXPECT().Say("Resuming VM..."),
					mockVBox.EXPECT().ResumeVM("some-vm").Return(errors.New("some-error")),
				)

				Expect(suspendedVM.Resume()).To(MatchError("failed to resume VM: some-error"))
			})
		})

		Context("when free memory is less than the VM's memory", func() {
			Context("when the user accepts to continue", func() {
				It("should succeed", func() {
					conf.FreeMemory = uint64(2000)
					suspendedVM.Memory = uint64(3000)

					gomock.InOrder(
						mockUI.EXPECT().Confirm("Less than 3000 MB of free memory detected, continue (y/N): ").Return(true),
						mockUI.EXPECT().Say("Resuming VM..."),
						mockVBox.EXPECT().ResumeVM("some-vm").Return(nil),
					)

					Expect(suspendedVM.Resume()).To(Succeed())
				})
			})

			Context("when the user declines to continue", func() {
				It("should return an error", func() {
					conf.FreeMemory = uint64(2000)
					suspendedVM.Memory = uint64(3000)

					mockUI.EXPECT().Confirm("Less than 3000 MB of free memory detected, continue (y/N): ").Return(false)

					Expect(suspendedVM.Resume()).To(MatchError("user declined to continue, exiting"))
				})
			})
		})
	})

	Describe("Status", func() {
		It("should return 'Suspended'", func() {
			Expect(suspendedVM.Status()).To(Equal("Suspended"))
		})
	})

	Describe("Destroy", func() {
		It("should destroy the vm", func() {
			mockVBox.EXPECT().DestroyVM("some-vm").Return(nil)

			Expect(suspendedVM.Destroy()).To(Succeed())
		})
	})
})

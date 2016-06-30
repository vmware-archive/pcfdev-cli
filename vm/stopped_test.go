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
			VMConfig: &config.VMConfig{
				Name:    "some-vm",
				Domain:  "some-domain",
				IP:      "some-ip",
				SSHPort: "some-port",
			},

			VBox:   mockVBox,
			UI:     mockUI,
			SSH:    mockSSH,
			Config: &config.Config{},
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("Stop", func() {
		It("should say a message", func() {
			mockUI.EXPECT().Say("PCF Dev is stopped.")
			stoppedVM.Stop()
		})
	})

	Describe("VerifyStartOpts", func() {
		Context("when desired memory is passed", func() {
			It("should return an error", func() {
				Expect(stoppedVM.VerifyStartOpts(&vm.StartOpts{
					Memory: 4000,
				})).To(MatchError("memory cannot be changed once the vm has been created"))
			})
		})

		Context("when desired cores is passed", func() {
			It("should return an error", func() {
				Expect(stoppedVM.VerifyStartOpts(&vm.StartOpts{
					CPUs: 2,
				})).To(MatchError("cores cannot be changed once the vm has been created"))
			})
		})

		Context("when services are passed", func() {
			It("should return an error", func() {
				Expect(stoppedVM.VerifyStartOpts(&vm.StartOpts{
					Services: "redis",
				})).To(MatchError("services cannot be changed once the vm has been created"))
			})
		})

		Context("when no opts are passed", func() {
			Context("when free memory is greater than or equal to the VM's memory", func() {
				It("should succeed", func() {
					stoppedVM.Config.FreeMemory = uint64(3000)
					stoppedVM.VMConfig.Memory = uint64(2000)
					Expect(stoppedVM.VerifyStartOpts(&vm.StartOpts{})).To(Succeed())
				})
			})

			Context("when free memory is less than the VM's memory", func() {
				Context("when the user accepts to continue", func() {
					It("should succeed", func() {
						stoppedVM.Config.FreeMemory = uint64(2000)
						stoppedVM.VMConfig.Memory = uint64(3000)

						mockUI.EXPECT().Confirm("Less than 3000 MB of free memory detected, continue (y/N): ").Return(true)

						Expect(stoppedVM.VerifyStartOpts(&vm.StartOpts{})).To(Succeed())
					})
				})

				Context("when the user declines to continue", func() {
					It("should return an error", func() {
						stoppedVM.Config.FreeMemory = uint64(2000)
						stoppedVM.VMConfig.Memory = uint64(3000)

						mockUI.EXPECT().Confirm("Less than 3000 MB of free memory detected, continue (y/N): ").Return(false)

						Expect(stoppedVM.VerifyStartOpts(&vm.StartOpts{})).To(MatchError("user declined to continue, exiting"))
					})
				})
			})
		})
	})

	Describe("Start", func() {
		Context("when 'none' services are specified", func() {
			It("should start vm with mysql", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Starting VM..."),
					mockVBox.EXPECT().StartVM(stoppedVM.VMConfig).Return(nil),
					mockUI.EXPECT().Say("Provisioning VM..."),
					mockSSH.EXPECT().RunSSHCommand("sudo /var/pcfdev/run some-domain some-ip mysql", "some-port", 5*time.Minute, os.Stdout, os.Stderr),
				)

				stoppedVM.Start(&vm.StartOpts{Services: "none"})
			})
		})

		Context("when 'all' services are specified", func() {
			It("should start the vm with services", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Starting VM..."),
					mockVBox.EXPECT().StartVM(stoppedVM.VMConfig).Return(nil),
					mockUI.EXPECT().Say("Provisioning VM..."),
					mockSSH.EXPECT().RunSSHCommand("sudo /var/pcfdev/run some-domain some-ip mysql,rabbitmq,redis", "some-port", 5*time.Minute, os.Stdout, os.Stderr),
				)

				stoppedVM.Start(&vm.StartOpts{Services: "all"})
			})
		})

		Context("when 'rabbitmq' services are specified", func() {
			It("should start the vm with services", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Starting VM..."),
					mockVBox.EXPECT().StartVM(stoppedVM.VMConfig).Return(nil),
					mockUI.EXPECT().Say("Provisioning VM..."),
					mockSSH.EXPECT().RunSSHCommand("sudo /var/pcfdev/run some-domain some-ip mysql,rabbitmq", "some-port", 5*time.Minute, os.Stdout, os.Stderr),
				)

				stoppedVM.Start(&vm.StartOpts{Services: "rabbitmq"})
			})
		})

		Context("when 'redis' services are specified", func() {
			It("should start the vm with services", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Starting VM..."),
					mockVBox.EXPECT().StartVM(stoppedVM.VMConfig).Return(nil),
					mockUI.EXPECT().Say("Provisioning VM..."),
					mockSSH.EXPECT().RunSSHCommand("sudo /var/pcfdev/run some-domain some-ip mysql,redis", "some-port", 5*time.Minute, os.Stdout, os.Stderr),
				)

				stoppedVM.Start(&vm.StartOpts{Services: "redis"})
			})
		})

		Context("when 'mysql' services are specified", func() {
			It("should start the vm with services", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Starting VM..."),
					mockVBox.EXPECT().StartVM(stoppedVM.VMConfig).Return(nil),
					mockUI.EXPECT().Say("Provisioning VM..."),
					mockSSH.EXPECT().RunSSHCommand("sudo /var/pcfdev/run some-domain some-ip mysql", "some-port", 5*time.Minute, os.Stdout, os.Stderr),
				)

				stoppedVM.Start(&vm.StartOpts{Services: "mysql"})
			})
		})

		Context("when '' services are specified", func() {
			It("should start the vm with all services", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Starting VM..."),
					mockVBox.EXPECT().StartVM(stoppedVM.VMConfig).Return(nil),
					mockUI.EXPECT().Say("Provisioning VM..."),
					mockSSH.EXPECT().RunSSHCommand("sudo /var/pcfdev/run some-domain some-ip mysql,rabbitmq,redis", "some-port", 5*time.Minute, os.Stdout, os.Stderr),
				)

				stoppedVM.Start(&vm.StartOpts{})
			})
		})

		Context("when starting the vm fails", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Starting VM..."),
					mockVBox.EXPECT().StartVM(stoppedVM.VMConfig).Return(errors.New("some-error")),
				)

				Expect(stoppedVM.Start(&vm.StartOpts{})).To(MatchError("failed to start VM: some-error"))
			})
		})

		Context("when provisioning the vm fails", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Starting VM..."),
					mockVBox.EXPECT().StartVM(stoppedVM.VMConfig).Return(nil),
					mockUI.EXPECT().Say("Provisioning VM..."),
					mockSSH.EXPECT().RunSSHCommand("sudo /var/pcfdev/run some-domain some-ip mysql,rabbitmq,redis", "some-port", 5*time.Minute, os.Stdout, os.Stderr).Return(errors.New("some-error")),
				)

				Expect(stoppedVM.Start(&vm.StartOpts{})).To(MatchError("failed to provision VM: some-error"))
			})
		})
		Context("when duplicate services are passed in", func() {
			It("should should start the vm without duplicate services", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Starting VM..."),
					mockVBox.EXPECT().StartVM(stoppedVM.VMConfig).Return(nil),
					mockUI.EXPECT().Say("Provisioning VM..."),
					mockSSH.EXPECT().RunSSHCommand("sudo /var/pcfdev/run some-domain some-ip mysql,redis", "some-port", 5*time.Minute, os.Stdout, os.Stderr),
				)
				stoppedVM.Start(&vm.StartOpts{Services: "redis,redis"})
			})
		})
	})

	Describe("Status", func() {
		It("should return 'Stopped'", func() {
			Expect(stoppedVM.Status()).To(Equal("Stopped"))
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

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
		mockCtrl          *gomock.Controller
		mockFS            *mocks.MockFS
		mockUI            *mocks.MockUI
		mockVBox          *mocks.MockVBox
		mockSSH           *mocks.MockSSH
		mockBuilder       *mocks.MockBuilder
		mockUnprovisioned *mocks.MockVM
		stoppedVM         vm.Stopped
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockFS = mocks.NewMockFS(mockCtrl)
		mockUI = mocks.NewMockUI(mockCtrl)
		mockVBox = mocks.NewMockVBox(mockCtrl)
		mockSSH = mocks.NewMockSSH(mockCtrl)
		mockBuilder = mocks.NewMockBuilder(mockCtrl)
		mockUnprovisioned = mocks.NewMockVM(mockCtrl)

		stoppedVM = vm.Stopped{
			VMConfig: &config.VMConfig{
				Name:    "some-vm",
				Domain:  "some-domain",
				IP:      "some-ip",
				SSHPort: "some-port",
			},

			VBox:    mockVBox,
			FS:      mockFS,
			UI:      mockUI,
			SSH:     mockSSH,
			Builder: mockBuilder,
			Config: &config.Config{
				VMDir: "some-vm-dir",
			},
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

		Context("when registries are passed", func() {
			It("should return an error", func() {
				Expect(stoppedVM.VerifyStartOpts(&vm.StartOpts{
					Registries: "some-private-registry",
				})).To(MatchError("private registries cannot be changed once the vm has been created"))
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
			It("should start vm with no extra services", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Starting VM..."),
					mockVBox.EXPECT().StartVM(stoppedVM.VMConfig),
					mockBuilder.EXPECT().VM("some-vm").Return(mockUnprovisioned, nil),
					mockSSH.EXPECT().RunSSHCommand("echo "+
						`'{"domain":"some-domain","ip":"some-ip","services":"","registries":[]}' | sudo tee /var/pcfdev/provision-options.json >/dev/null`,
						"127.0.0.1", "some-port", 5*time.Minute, os.Stdout, os.Stderr),
					mockUnprovisioned.EXPECT().Provision(),
				)

				stoppedVM.Start(&vm.StartOpts{Services: "none"})
			})
		})

		Context("when 'all' services are specified", func() {
			It("should start the vm with services", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Starting VM..."),
					mockVBox.EXPECT().StartVM(stoppedVM.VMConfig),
					mockBuilder.EXPECT().VM("some-vm").Return(mockUnprovisioned, nil),
					mockSSH.EXPECT().RunSSHCommand("echo "+
						`'{"domain":"some-domain","ip":"some-ip","services":"rabbitmq,redis,spring-cloud-services","registries":[]}' | sudo tee /var/pcfdev/provision-options.json >/dev/null`,
						"127.0.0.1", "some-port", 5*time.Minute, os.Stdout, os.Stderr),
					mockUnprovisioned.EXPECT().Provision(),
				)

				stoppedVM.Start(&vm.StartOpts{Services: "all"})
			})
		})

		Context("when 'default' services are specified", func() {
			It("should start the vm with rabbitmq and redis", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Starting VM..."),
					mockVBox.EXPECT().StartVM(stoppedVM.VMConfig),
					mockBuilder.EXPECT().VM("some-vm").Return(mockUnprovisioned, nil),
					mockSSH.EXPECT().RunSSHCommand("echo "+
						`'{"domain":"some-domain","ip":"some-ip","services":"rabbitmq,redis","registries":[]}' | sudo tee /var/pcfdev/provision-options.json >/dev/null`,
						"127.0.0.1", "some-port", 5*time.Minute, os.Stdout, os.Stderr),
					mockUnprovisioned.EXPECT().Provision(),
				)

				stoppedVM.Start(&vm.StartOpts{Services: "default"})
			})
		})

		Context("when 'spring-cloud-services' services are specified", func() {
			It("should start the vm with spring-cloud-services and rabbitmq", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Starting VM..."),
					mockVBox.EXPECT().StartVM(stoppedVM.VMConfig),
					mockBuilder.EXPECT().VM("some-vm").Return(mockUnprovisioned, nil),
					mockSSH.EXPECT().RunSSHCommand("echo "+
						`'{"domain":"some-domain","ip":"some-ip","services":"rabbitmq,spring-cloud-services","registries":[]}' | sudo tee /var/pcfdev/provision-options.json >/dev/null`,
						"127.0.0.1", "some-port", 5*time.Minute, os.Stdout, os.Stderr),
					mockUnprovisioned.EXPECT().Provision(),
				)

				stoppedVM.Start(&vm.StartOpts{Services: "spring-cloud-services"})
			})
		})

		Context("when 'scs' is specified", func() {
			It("should start the vm with spring-cloud-services and rabbitmq", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Starting VM..."),
					mockVBox.EXPECT().StartVM(stoppedVM.VMConfig),
					mockBuilder.EXPECT().VM("some-vm").Return(mockUnprovisioned, nil),
					mockSSH.EXPECT().RunSSHCommand("echo "+
						`'{"domain":"some-domain","ip":"some-ip","services":"rabbitmq,spring-cloud-services","registries":[]}' | sudo tee /var/pcfdev/provision-options.json >/dev/null`,
						"127.0.0.1", "some-port", 5*time.Minute, os.Stdout, os.Stderr),
					mockUnprovisioned.EXPECT().Provision(),
				)

				stoppedVM.Start(&vm.StartOpts{Services: "scs"})
			})
		})

		Context("when 'rabbitmq' services are specified", func() {
			It("should start the vm with rabbitmq", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Starting VM..."),
					mockVBox.EXPECT().StartVM(stoppedVM.VMConfig),
					mockBuilder.EXPECT().VM("some-vm").Return(mockUnprovisioned, nil),
					mockSSH.EXPECT().RunSSHCommand("echo "+
						`'{"domain":"some-domain","ip":"some-ip","services":"rabbitmq","registries":[]}' | sudo tee /var/pcfdev/provision-options.json >/dev/null`,
						"127.0.0.1", "some-port", 5*time.Minute, os.Stdout, os.Stderr),
					mockUnprovisioned.EXPECT().Provision(),
				)

				stoppedVM.Start(&vm.StartOpts{Services: "rabbitmq"})
			})
		})

		Context("when 'redis' services are specified", func() {
			It("should start the vm with redis", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Starting VM..."),
					mockVBox.EXPECT().StartVM(stoppedVM.VMConfig),
					mockBuilder.EXPECT().VM("some-vm").Return(mockUnprovisioned, nil),
					mockSSH.EXPECT().RunSSHCommand("echo "+
						`'{"domain":"some-domain","ip":"some-ip","services":"redis","registries":[]}' | sudo tee /var/pcfdev/provision-options.json >/dev/null`,
						"127.0.0.1", "some-port", 5*time.Minute, os.Stdout, os.Stderr),
					mockUnprovisioned.EXPECT().Provision(),
				)

				stoppedVM.Start(&vm.StartOpts{Services: "redis"})
			})
		})

		Context("when 'mysql' services are specified", func() {
			It("should start the vm with no extra services", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Starting VM..."),
					mockVBox.EXPECT().StartVM(stoppedVM.VMConfig),
					mockBuilder.EXPECT().VM("some-vm").Return(mockUnprovisioned, nil),
					mockSSH.EXPECT().RunSSHCommand("echo "+
						`'{"domain":"some-domain","ip":"some-ip","services":"","registries":[]}' | sudo tee /var/pcfdev/provision-options.json >/dev/null`,
						"127.0.0.1", "some-port", 5*time.Minute, os.Stdout, os.Stderr),
					mockUnprovisioned.EXPECT().Provision(),
				)

				stoppedVM.Start(&vm.StartOpts{Services: "mysql"})
			})
		})

		Context("when duplicate services are specified", func() {
			It("should start the vm without duplicates services", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Starting VM..."),
					mockVBox.EXPECT().StartVM(stoppedVM.VMConfig),
					mockBuilder.EXPECT().VM("some-vm").Return(mockUnprovisioned, nil),
					mockSSH.EXPECT().RunSSHCommand("echo "+
						`'{"domain":"some-domain","ip":"some-ip","services":"rabbitmq,redis,spring-cloud-services","registries":[]}' | sudo tee /var/pcfdev/provision-options.json >/dev/null`,
						"127.0.0.1", "some-port", 5*time.Minute, os.Stdout, os.Stderr),
					mockUnprovisioned.EXPECT().Provision(),
				)

				stoppedVM.Start(&vm.StartOpts{Services: "default,spring-cloud-services,scs"})
			})
		})

		Context("when '' services are specified", func() {
			It("should start the vm with default services", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Starting VM..."),
					mockVBox.EXPECT().StartVM(stoppedVM.VMConfig),
					mockBuilder.EXPECT().VM("some-vm").Return(mockUnprovisioned, nil),
					mockSSH.EXPECT().RunSSHCommand("echo "+
						`'{"domain":"some-domain","ip":"some-ip","services":"rabbitmq,redis","registries":[]}' | sudo tee /var/pcfdev/provision-options.json >/dev/null`,
						"127.0.0.1", "some-port", 5*time.Minute, os.Stdout, os.Stderr),
					mockUnprovisioned.EXPECT().Provision(),
				)

				stoppedVM.Start(&vm.StartOpts{})
			})
		})

		Context("when docker registries are specified", func() {
			It("should start the vm with the registries accessible", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Starting VM..."),
					mockVBox.EXPECT().StartVM(stoppedVM.VMConfig),
					mockBuilder.EXPECT().VM("some-vm").Return(mockUnprovisioned, nil),
					mockSSH.EXPECT().RunSSHCommand("echo "+
						`'{"domain":"some-domain","ip":"some-ip","services":"rabbitmq,redis","registries":["some-private-registry","some-other-private-registry"]}' | sudo tee /var/pcfdev/provision-options.json >/dev/null`,
						"127.0.0.1", "some-port", 5*time.Minute, os.Stdout, os.Stderr),
					mockUnprovisioned.EXPECT().Provision(),
				)

				stoppedVM.Start(&vm.StartOpts{Registries: "some-private-registry,some-other-private-registry"})
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

		Context("when SSHing provisions into the vm fails", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Starting VM..."),
					mockVBox.EXPECT().StartVM(stoppedVM.VMConfig),
					mockBuilder.EXPECT().VM("some-vm").Return(mockUnprovisioned, nil),
					mockSSH.EXPECT().RunSSHCommand("echo "+
						`'{"domain":"some-domain","ip":"some-ip","services":"rabbitmq,redis","registries":[]}' | sudo tee /var/pcfdev/provision-options.json >/dev/null`,
						"127.0.0.1", "some-port", 5*time.Minute, os.Stdout, os.Stderr).Return(errors.New("some-error")),
				)

				Expect(stoppedVM.Start(&vm.StartOpts{})).To(MatchError("failed to start VM: some-error"))
			})
		})

		Context("when '-n' (no-provision) flag is passed in", func() {
			It("should not provision the vm", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Starting VM..."),
					mockVBox.EXPECT().StartVM(stoppedVM.VMConfig),
					mockBuilder.EXPECT().VM("some-vm"),
					mockSSH.EXPECT().RunSSHCommand("echo "+
						`'{"domain":"some-domain","ip":"some-ip","services":"rabbitmq,redis","registries":[]}' | sudo tee /var/pcfdev/provision-options.json >/dev/null`,
						"127.0.0.1", "some-port", 5*time.Minute, os.Stdout, os.Stderr),
					mockUI.EXPECT().Say("VM will not be provisioned because '-n' (no-provision) flag was specified."),
				)

				stoppedVM.Start(&vm.StartOpts{
					NoProvision: true,
				})
			})
		})

		Context("when retrieving the unprovisioned vm fails", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Starting VM..."),
					mockVBox.EXPECT().StartVM(stoppedVM.VMConfig),
					mockBuilder.EXPECT().VM("some-vm").Return(nil, errors.New("some-error")),
				)

				Expect(stoppedVM.Start(&vm.StartOpts{})).To(MatchError("failed to start VM: some-error"))
			})
		})

		Context("when provisioning the unprovisioned vm fails", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Starting VM..."),
					mockVBox.EXPECT().StartVM(stoppedVM.VMConfig),
					mockBuilder.EXPECT().VM("some-vm").Return(mockUnprovisioned, nil),
					mockSSH.EXPECT().RunSSHCommand("echo "+
						`'{"domain":"some-domain","ip":"some-ip","services":"rabbitmq,redis","registries":[]}' | sudo tee /var/pcfdev/provision-options.json >/dev/null`,
						"127.0.0.1", "some-port", 5*time.Minute, os.Stdout, os.Stderr),
					mockUnprovisioned.EXPECT().Provision().Return(errors.New("some-error")),
				)

				Expect(stoppedVM.Start(&vm.StartOpts{})).To(MatchError("some-error"))
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

	Describe("GetDebugLogs", func() {
		It("should say a message", func() {
			mockUI.EXPECT().Say("Your VM is currently stopped. Start vm to retrieve debug logs.")
			Expect(stoppedVM.GetDebugLogs()).To(Succeed())
		})
	})
})

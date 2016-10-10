package vm_test

import (
	"errors"
	"path/filepath"

	"github.com/golang/mock/gomock"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/user"
	"github.com/pivotal-cf/pcfdev-cli/vm"
	"github.com/pivotal-cf/pcfdev-cli/vm/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Not Created", func() {
	var (
		mockCtrl     *gomock.Controller
		mockUI       *mocks.MockUI
		mockVBox     *mocks.MockVBox
		mockBuilder  *mocks.MockBuilder
		mockStopped  *mocks.MockVM
		mockFS       *mocks.MockFS
		mockNetwork  *mocks.MockNetwork
		notCreatedVM vm.NotCreated
		conf         *config.Config
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockUI = mocks.NewMockUI(mockCtrl)
		mockVBox = mocks.NewMockVBox(mockCtrl)
		mockBuilder = mocks.NewMockBuilder(mockCtrl)
		mockStopped = mocks.NewMockVM(mockCtrl)
		mockFS = mocks.NewMockFS(mockCtrl)
		mockNetwork = mocks.NewMockNetwork(mockCtrl)
		conf = &config.Config{
			DefaultCPUs: func() (int, error) { return 0, nil },
		}

		notCreatedVM = vm.NotCreated{
			VMConfig: &config.VMConfig{
				Name: "some-vm",
			},

			VBox:    mockVBox,
			UI:      mockUI,
			Builder: mockBuilder,
			FS:      mockFS,
			Config:  conf,
			Network: mockNetwork,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("Stop", func() {
		It("should print a message", func() {
			mockUI.EXPECT().Say("PCF Dev VM has not been created.")

			notCreatedVM.Stop()
		})
	})

	Describe("VerifyStartOpts", func() {
		Context("when memory is passed as an option", func() {
			Context("when the desired memory is less than the minimum", func() {
				It("should print an error", func() {
					conf.MinMemory = uint64(3000)

					Expect(notCreatedVM.VerifyStartOpts(&vm.StartOpts{
						Memory: uint64(2000),
					})).To(MatchError("PCF Dev requires at least 3000 MB of memory to run"))
				})
			})

			Context("when 'scs' is passed as a service", func() {
				It("should use the designated spring cloud min memory limit", func() {
					conf.SpringCloudMinMemory = uint64(6000)

					Expect(notCreatedVM.VerifyStartOpts(&vm.StartOpts{
						Memory:   uint64(3500),
						Services: "scs",
					})).To(MatchError("PCF Dev requires at least 6000 MB of memory to run"))
				})
			})

			Context("when 'spring-cloud-services' is passed as a service", func() {
				It("should use the designated spring cloud min memory limit", func() {
					conf.SpringCloudMinMemory = uint64(6000)

					Expect(notCreatedVM.VerifyStartOpts(&vm.StartOpts{
						Memory:   uint64(3500),
						Services: "spring-cloud-services",
					})).To(MatchError("PCF Dev requires at least 6000 MB of memory to run"))
				})
			})

			Context("when 'all' is passed as a service", func() {
				It("should use the designated spring cloud min memory limit", func() {
					conf.SpringCloudMinMemory = uint64(6000)

					Expect(notCreatedVM.VerifyStartOpts(&vm.StartOpts{
						Memory:   uint64(3500),
						Services: "all",
					})).To(MatchError("PCF Dev requires at least 6000 MB of memory to run"))
				})
			})

			Context("when a colliding IP is selected", func() {
				It("should show a warning about the colliding IP", func() {
					mockNetwork.EXPECT().HasIPCollision("192.168.33.1").Return(true, nil)
					mockUI.EXPECT().Say("Warning: the chosen PCF Dev VM IP address may be in use by another VM or device.")

					Expect(notCreatedVM.VerifyStartOpts(&vm.StartOpts{
						IP: "192.168.33.11",
					})).To(Succeed())
				})
			})

			Context("when a colliding ip is not a valid IP", func() {
				It("should return an error", func() {
					Expect(notCreatedVM.VerifyStartOpts(&vm.StartOpts{
						IP: "some-ip",
					})).To(MatchError("some-ip is not a supported IP address"))
				})
			})

			Context("when there is an error decidiing if there is a colliding IP", func() {
				It("should return an error", func() {
					mockNetwork.EXPECT().HasIPCollision("192.168.11.1").Return(false, errors.New("some-error"))

					Expect(notCreatedVM.VerifyStartOpts(&vm.StartOpts{
						IP: "192.168.11.11",
					})).To(MatchError("some-error"))
				})
			})

			Context("when non-standard domain and no IP is passed", func() {
				It("should return an error", func() {
					Expect(notCreatedVM.VerifyStartOpts(&vm.StartOpts{
						Domain: "some-bad-domain",
					})).To(MatchError("some-bad-domain is not one of the allowed PCF Dev domains"))
				})
			})

			Context("when non-standard domain and IP is passed", func() {
				It("should succeed", func() {
					mockNetwork.EXPECT().HasIPCollision("192.168.11.1").Return(false, nil)

					Expect(notCreatedVM.VerifyStartOpts(&vm.StartOpts{
						IP:     "192.168.11.11",
						Domain: "some-non-standard-domain",
					})).To(Succeed())
				})
			})

			Context("when a valid domain is passed", func() {
				It("should succeed", func() {
					Expect(notCreatedVM.VerifyStartOpts(&vm.StartOpts{
						Domain: "local.pcfdev.io",
					})).To(Succeed())
				})
			})

			Context("when docker registries are passed in 'host:port' format", func() {
				It("should succeed", func() {
					Expect(notCreatedVM.VerifyStartOpts(&vm.StartOpts{
						Registries: "some-host:some-port",
					})).To(Succeed())
				})
			})

			Context("when docker registries not passed in 'host:port' format", func() {
				It("should return an error", func() {
					Expect(notCreatedVM.VerifyStartOpts(&vm.StartOpts{
						Registries: "some-host?some-port",
					})).To(MatchError("docker registries must be passed in 'host:port' format"))
				})
			})

			Context("when the desired memory is equal to the minimum and less than free memory", func() {
				It("should succeed", func() {
					conf.FreeMemory = uint64(5000)
					conf.MinMemory = uint64(3000)

					Expect(notCreatedVM.VerifyStartOpts(&vm.StartOpts{
						Memory: uint64(3000),
					})).To(Succeed())
				})
			})

			Context("when the desired memory is greater than the minimum and less than free memory", func() {
				It("should succeed", func() {
					conf.FreeMemory = uint64(5000)
					conf.MinMemory = uint64(3000)

					Expect(notCreatedVM.VerifyStartOpts(&vm.StartOpts{
						Memory: uint64(4000),
					})).To(Succeed())
				})
			})

			Context("when desired memory is greater than free memory", func() {
				Context("when the user accepts to continue", func() {
					It("should succeed", func() {
						conf.FreeMemory = uint64(2000)

						mockUI.EXPECT().Confirm("Less than 4000 MB of free memory detected, continue (y/N): ").Return(true)

						Expect(notCreatedVM.VerifyStartOpts(&vm.StartOpts{
							Memory: uint64(4000),
						})).To(Succeed())
					})
				})
				Context("when the user declines to continue", func() {
					It("should return an error", func() {
						conf.FreeMemory = uint64(2000)

						mockUI.EXPECT().Confirm("Less than 4000 MB of free memory detected, continue (y/N): ").Return(false)

						Expect(notCreatedVM.VerifyStartOpts(&vm.StartOpts{
							Memory: uint64(4000),
						})).To(MatchError("user declined to continue, exiting"))
					})
				})
			})
		})

		Context("when scs is passed as a service", func() {
			It("should use the default memory for Spring Cloud Services", func() {
				conf.SpringCloudDefaultMemory = uint64(6000)
				conf.FreeMemory = uint64(3500)

				mockUI.EXPECT().Confirm("Less than 6000 MB of free memory detected, continue (y/N): ").Return(true)

				Expect(notCreatedVM.VerifyStartOpts(&vm.StartOpts{
					Services: "scs",
				})).To(Succeed())
			})
		})

		Context("when spring-cloud-services is passed as a service", func() {
			It("should use the default memory for Spring Cloud Services", func() {
				conf.SpringCloudDefaultMemory = uint64(6000)
				conf.FreeMemory = uint64(3500)

				mockUI.EXPECT().Confirm("Less than 6000 MB of free memory detected, continue (y/N): ").Return(true)

				Expect(notCreatedVM.VerifyStartOpts(&vm.StartOpts{
					Services: "spring-cloud-services",
				})).To(Succeed())
			})
		})

		Context("when all is passed as a service", func() {
			It("should use the default memory for Spring Cloud Services", func() {
				conf.SpringCloudDefaultMemory = uint64(6000)
				conf.FreeMemory = uint64(3500)

				mockUI.EXPECT().Confirm("Less than 6000 MB of free memory detected, continue (y/N): ").Return(true)

				Expect(notCreatedVM.VerifyStartOpts(&vm.StartOpts{
					Services: "all",
				})).To(Succeed())
			})
		})

		Context("when initial services are passed in as option", func() {
			Context("when services specifed are invalid", func() {
				It("should return an error", func() {
					Expect(notCreatedVM.VerifyStartOpts(&vm.StartOpts{
						Services: "some-bad-service,redis,mysql,some-bad-service-2",
					})).To(MatchError("invalid services specified: some-bad-service, some-bad-service-2"))
				})
			})

			Context("when valid comma separated services are specifed", func() {
				It("should succeed", func() {
					Expect(notCreatedVM.VerifyStartOpts(&vm.StartOpts{
						Services: "none,all,default,redis,mysql,rabbitmq,spring-cloud-services,scs",
					})).To(Succeed())
				})
			})

			Context("when empty string service", func() {
				It("should succeed because it is the default", func() {
					Expect(notCreatedVM.VerifyStartOpts(&vm.StartOpts{
						Services: "",
					})).To(Succeed())
				})
			})
		})

		Context("when memory is not passed as an option", func() {
			Context("when the default memory is equal to free memory", func() {
				It("should succeed", func() {
					conf.FreeMemory = uint64(3000)
					conf.DefaultMemory = uint64(3000)

					Expect(notCreatedVM.VerifyStartOpts(&vm.StartOpts{})).To(Succeed())
				})
			})

			Context("when the default memory is less than free memory", func() {
				It("should succeed", func() {
					conf.FreeMemory = uint64(5000)
					conf.DefaultMemory = uint64(3000)

					Expect(notCreatedVM.VerifyStartOpts(&vm.StartOpts{})).To(Succeed())
				})
			})

			Context("when default memory is greater than free memory", func() {
				Context("when the user accepts to continue", func() {
					It("should succeed", func() {
						conf.FreeMemory = uint64(3000)
						conf.DefaultMemory = uint64(4000)

						mockUI.EXPECT().Confirm("Less than 4000 MB of free memory detected, continue (y/N): ").Return(true)

						Expect(notCreatedVM.VerifyStartOpts(&vm.StartOpts{})).To(Succeed())
					})
				})

				Context("when the user declines to continue", func() {
					It("should return an error", func() {
						conf.FreeMemory = uint64(3000)
						conf.DefaultMemory = uint64(4000)

						mockUI.EXPECT().Confirm("Less than 4000 MB of free memory detected, continue (y/N): ").Return(false)

						Expect(notCreatedVM.VerifyStartOpts(&vm.StartOpts{})).To(MatchError("user declined to continue, exiting"))
					})
				})
			})
		})

		Context("when cores is passed as an option", func() {
			Context("when cores is a positive number", func() {
				It("should succeed", func() {
					Expect(notCreatedVM.VerifyStartOpts(&vm.StartOpts{CPUs: 4})).To(Succeed())
				})
			})

			Context("when cores is less than zero", func() {
				It("should return an error", func() {
					Expect(notCreatedVM.VerifyStartOpts(&vm.StartOpts{CPUs: -1})).To(MatchError("cannot start with less than one core"))
				})
			})
		})

		Context("when OVAPath is passed as an option", func() {
			It("should skip memory check", func() {
				conf.FreeMemory = uint64(3000)
				conf.DefaultMemory = uint64(4000)
				mockFS.EXPECT().Exists("some-ova-path").Return(true, nil)
				Expect(notCreatedVM.VerifyStartOpts(&vm.StartOpts{
					OVAPath: "some-ova-path",
				})).To(Succeed())
			})
		})

		Context("when there is no file at the path specified by OVAPath", func() {
			It("should throw an error", func() {
				mockFS.EXPECT().Exists("some-ova-path").Return(false, nil)
				Expect(notCreatedVM.VerifyStartOpts(&vm.StartOpts{
					OVAPath: "some-ova-path",
				})).To(MatchError("no file found at some-ova-path"))
			})
		})

		Context("when checking if ova exists returns an error", func() {
			It("should throw an error", func() {
				mockFS.EXPECT().Exists("some-ova-path").Return(false, errors.New("some-error"))
				Expect(notCreatedVM.VerifyStartOpts(&vm.StartOpts{
					OVAPath: "some-ova-path",
				})).To(MatchError("some-error"))
			})
		})
	})

	Describe("Start", func() {
		var home string

		BeforeEach(func() {
			var err error
			home, err = user.GetHome()
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when opts are provided", func() {
			It("should import and start the vm with given options", func() {
				startOpts := &vm.StartOpts{
					Memory:   uint64(4000),
					CPUs:     3,
					OVAPath:  "some-ova-path",
					Services: "all",
					IP:       "some-ip",
					Domain:   "some-domain",
				}
				gomock.InOrder(
					mockUI.EXPECT().Say("Allocating 4000 MB out of 8000 MB total system memory (5000 MB free)."),
					mockUI.EXPECT().Say("Importing VM..."),
					mockVBox.EXPECT().ImportVM(&config.VMConfig{
						Name:    "some-vm",
						Memory:  uint64(4000),
						CPUs:    3,
						OVAPath: "some-ova-path",
						IP:      "some-ip",
						Domain:  "some-domain",
					}),
					mockBuilder.EXPECT().VM("some-vm").Return(mockStopped, nil),
					mockStopped.EXPECT().Start(startOpts),
				)
				conf.FreeMemory = uint64(5000)
				conf.TotalMemory = uint64(8000)

				notCreatedVM.Start(startOpts)
			})
		})

		Context("when scs is passed as a service", func() {
			It("should use the spring cloud default memory", func() {
				startOpts := &vm.StartOpts{
					CPUs:     3,
					OVAPath:  "some-ova-path",
					Services: "scs",
				}
				gomock.InOrder(
					mockUI.EXPECT().Say("Allocating 6000 MB out of 8000 MB total system memory (7000 MB free)."),
					mockUI.EXPECT().Say("Importing VM..."),
					mockVBox.EXPECT().ImportVM(&config.VMConfig{
						Name:    "some-vm",
						Memory:  uint64(6000),
						CPUs:    3,
						OVAPath: "some-ova-path",
					}),
					mockBuilder.EXPECT().VM("some-vm").Return(mockStopped, nil),
					mockStopped.EXPECT().Start(startOpts),
				)
				conf.SpringCloudDefaultMemory = uint64(6000)
				conf.FreeMemory = uint64(7000)
				conf.TotalMemory = uint64(8000)

				notCreatedVM.Start(startOpts)
			})
		})

		Context("when spring-cloud-services is passed as a service", func() {
			It("should use the spring cloud default memory", func() {
				startOpts := &vm.StartOpts{
					CPUs:     3,
					OVAPath:  "some-ova-path",
					Services: "spring-cloud-services",
				}
				gomock.InOrder(
					mockUI.EXPECT().Say("Allocating 6000 MB out of 8000 MB total system memory (7000 MB free)."),
					mockUI.EXPECT().Say("Importing VM..."),
					mockVBox.EXPECT().ImportVM(&config.VMConfig{
						Name:    "some-vm",
						Memory:  uint64(6000),
						CPUs:    3,
						OVAPath: "some-ova-path",
					}),
					mockBuilder.EXPECT().VM("some-vm").Return(mockStopped, nil),
					mockStopped.EXPECT().Start(startOpts),
				)
				conf.SpringCloudDefaultMemory = uint64(6000)
				conf.FreeMemory = uint64(7000)
				conf.TotalMemory = uint64(8000)

				notCreatedVM.Start(startOpts)
			})
		})

		Context("when all is passed as a service", func() {
			It("should use the spring cloud default memory", func() {
				startOpts := &vm.StartOpts{
					CPUs:     3,
					OVAPath:  "some-ova-path",
					Services: "all",
				}
				gomock.InOrder(
					mockUI.EXPECT().Say("Allocating 6000 MB out of 8000 MB total system memory (7000 MB free)."),
					mockUI.EXPECT().Say("Importing VM..."),
					mockVBox.EXPECT().ImportVM(&config.VMConfig{
						Name:    "some-vm",
						Memory:  uint64(6000),
						CPUs:    3,
						OVAPath: "some-ova-path",
					}),
					mockBuilder.EXPECT().VM("some-vm").Return(mockStopped, nil),
					mockStopped.EXPECT().Start(startOpts),
				)
				conf.SpringCloudDefaultMemory = uint64(6000)
				conf.FreeMemory = uint64(7000)
				conf.TotalMemory = uint64(8000)

				notCreatedVM.Start(startOpts)
			})
		})

		Context("when the opts are not provided", func() {
			It("should give the VM the default memory and cpus", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Allocating 3500 MB out of 8000 MB total system memory (5000 MB free)."),
					mockUI.EXPECT().Say("Importing VM..."),
					mockVBox.EXPECT().ImportVM(&config.VMConfig{
						Name:    "some-vm",
						Memory:  uint64(3500),
						CPUs:    7,
						OVAPath: filepath.Join("some-ova-dir", "some-vm.ova"),
					}),
					mockBuilder.EXPECT().VM("some-vm").Return(mockStopped, nil),
					mockStopped.EXPECT().Start(&vm.StartOpts{}),
				)
				conf.OVADir = "some-ova-dir"
				conf.DefaultCPUs = func() (int, error) { return 7, nil }
				conf.DefaultMemory = uint64(3500)
				conf.FreeMemory = uint64(5000)
				conf.TotalMemory = uint64(8000)

				Expect(notCreatedVM.Start(&vm.StartOpts{})).To(Succeed())
			})

			Context("and checking number of CPUs returns an error", func() {
				It("should return an error", func() {
					conf.DefaultCPUs = func() (int, error) { return 0, errors.New("some-error") }
					Expect(notCreatedVM.Start(&vm.StartOpts{})).To(MatchError("some-error"))
				})
			})
		})

		Context("when there is an error importing the VM", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Allocating 3072 MB out of 0 MB total system memory (0 MB free)."),
					mockUI.EXPECT().Say("Importing VM..."),
					mockVBox.EXPECT().ImportVM(&config.VMConfig{
						Name:    "some-vm",
						Memory:  uint64(3072),
						OVAPath: filepath.Join("some-ova-dir", "some-vm.ova"),
					}).Return(errors.New("some-error")),
				)
				conf.OVADir = "some-ova-dir"

				Expect(notCreatedVM.Start(&vm.StartOpts{
					Memory: uint64(3072),
				})).To(MatchError("failed to import VM: some-error"))
			})
		})

		Context("when there is an error constructing a stopped VM", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Allocating 3072 MB out of 0 MB total system memory (0 MB free)."),
					mockUI.EXPECT().Say("Importing VM..."),
					mockVBox.EXPECT().ImportVM(&config.VMConfig{
						Name:    "some-vm",
						Memory:  uint64(3072),
						OVAPath: filepath.Join("some-ova-dir", "some-vm.ova"),
					}),
					mockBuilder.EXPECT().VM("some-vm").Return(nil, errors.New("some-error")),
				)
				conf.OVADir = "some-ova-dir"

				Expect(notCreatedVM.Start(&vm.StartOpts{
					Memory: uint64(3072),
				})).To(MatchError("failed to start VM: some-error"))
			})
		})

		Context("when there is an error starting the stopped VM", func() {
			It("should return an error", func() {
				startOpts := &vm.StartOpts{
					Memory: uint64(3072),
				}
				gomock.InOrder(
					mockUI.EXPECT().Say("Allocating 3072 MB out of 0 MB total system memory (0 MB free)."),
					mockUI.EXPECT().Say("Importing VM..."),
					mockVBox.EXPECT().ImportVM(&config.VMConfig{
						Name:    "some-vm",
						Memory:  uint64(3072),
						OVAPath: filepath.Join("some-ova-dir", "some-vm.ova"),
					}),
					mockBuilder.EXPECT().VM("some-vm").Return(mockStopped, nil),
					mockStopped.EXPECT().Start(startOpts).Return(errors.New("failed to start VM: some-error")),
				)
				conf.OVADir = "some-ova-dir"

				Expect(notCreatedVM.Start(startOpts)).To(MatchError("failed to start VM: some-error"))
			})
		})
	})

	Describe("Status", func() {
		It("should return 'Not Created'", func() {
			Expect(notCreatedVM.Status()).To(Equal("Not Created"))
		})
	})

	Describe("Suspend", func() {
		It("should say message", func() {
			mockUI.EXPECT().Say("No VM running, cannot suspend.")

			Expect(notCreatedVM.Suspend()).To(Succeed())
		})
	})

	Describe("Resume", func() {
		It("should say message", func() {
			mockUI.EXPECT().Say("No VM suspended, cannot resume.")

			Expect(notCreatedVM.Resume()).To(Succeed())
		})
	})

	Describe("GetDebugLogs", func() {
		It("should say message", func() {
			mockUI.EXPECT().Say("No VM created, cannot retrieve debug logs.")

			Expect(notCreatedVM.GetDebugLogs()).To(Succeed())
		})
	})

	Describe("Trust", func() {
		It("should say message", func() {
			mockUI.EXPECT().Say("No VM created, cannot trust VM certificates.")

			Expect(notCreatedVM.Trust(&vm.StartOpts{})).To(Succeed())
		})
	})

	Describe("Target", func() {
		It("should say message", func() {
			mockUI.EXPECT().Say("No VM created, cannot target PCF Dev.")

			Expect(notCreatedVM.Target(false)).To(Succeed())
		})
	})

	Describe("SSH", func() {
		It("should say message", func() {
			mockUI.EXPECT().Say("No VM created, cannot SSH to PCF Dev.")

			Expect(notCreatedVM.SSH()).To(Succeed())
		})
	})
})

package vm_test

import (
	"errors"

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
		notCreatedVM vm.NotCreated
		conf         *config.Config
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockUI = mocks.NewMockUI(mockCtrl)
		mockVBox = mocks.NewMockVBox(mockCtrl)
		mockBuilder = mocks.NewMockBuilder(mockCtrl)
		mockStopped = mocks.NewMockVM(mockCtrl)
		conf = &config.Config{}

		notCreatedVM = vm.NotCreated{
			VMConfig: &config.VMConfig{
				Name: "some-vm",
			},

			VBox:    mockVBox,
			UI:      mockUI,
			Builder: mockBuilder,
			Config:  conf,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("Stop", func() {
		Context("when no conflicting vm is present", func() {
			It("should print a message", func() {
				gomock.InOrder(
					mockVBox.EXPECT().ConflictingVMPresent(notCreatedVM.VMConfig).Return(false, nil),
					mockUI.EXPECT().Say("PCF Dev VM has not been created"),
				)

				notCreatedVM.Stop()
			})
		})
		Context("when conflicting vm is present", func() {
			It("should return an error", func() {
				mockVBox.EXPECT().ConflictingVMPresent(notCreatedVM.VMConfig).Return(true, nil)

				Expect(notCreatedVM.Stop()).To(MatchError("old version of PCF Dev already running"))
			})
		})
		Context("when there is an error seeing if there is an conflicting VM present", func() {
			It("should return an error", func() {
				mockVBox.EXPECT().ConflictingVMPresent(notCreatedVM.VMConfig).Return(false, errors.New("some-error"))

				Expect(notCreatedVM.Stop()).To(MatchError("failed to stop VM: some-error"))
			})
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
				gomock.InOrder(
					mockVBox.EXPECT().ConflictingVMPresent(notCreatedVM.VMConfig).Return(false, nil),
					mockUI.EXPECT().Say("Allocating 4000 MB out of 8000 MB total system memory (5000 MB free)."),
					mockUI.EXPECT().Say("Importing VM..."),
					mockVBox.EXPECT().ImportVM(&config.VMConfig{
						Name:   "some-vm",
						Memory: uint64(4000),
						CPUs:   3,
					}).Return(nil),
					mockBuilder.EXPECT().VM("some-vm").Return(mockStopped, nil),
					mockStopped.EXPECT().Start(&vm.StartOpts{}),
				)
				conf.FreeMemory = uint64(5000)
				conf.TotalMemory = uint64(8000)

				notCreatedVM.Start(&vm.StartOpts{
					Memory: uint64(4000),
					CPUs:   3,
				})
			})
		})

		Context("when the opts are not provided", func() {
			It("should give the VM the default memory and cpus", func() {
				gomock.InOrder(
					mockVBox.EXPECT().ConflictingVMPresent(notCreatedVM.VMConfig).Return(false, nil),
					mockUI.EXPECT().Say("Allocating 3500 MB out of 8000 MB total system memory (5000 MB free)."),
					mockUI.EXPECT().Say("Importing VM..."),
					mockVBox.EXPECT().ImportVM(&config.VMConfig{
						Name:   "some-vm",
						Memory: uint64(3500),
						CPUs:   7,
					}).Return(nil),
					mockBuilder.EXPECT().VM("some-vm").Return(mockStopped, nil),
					mockStopped.EXPECT().Start(&vm.StartOpts{}).Return(nil),
				)
				conf.DefaultCPUs = 7
				conf.DefaultMemory = uint64(3500)
				conf.FreeMemory = uint64(5000)
				conf.TotalMemory = uint64(8000)

				Expect(notCreatedVM.Start(&vm.StartOpts{})).To(Succeed())
			})
		})

		Context("when there is an error seeing if conflicting vms are present", func() {
			It("should return an error", func() {
				mockVBox.EXPECT().ConflictingVMPresent(notCreatedVM.VMConfig).Return(false, errors.New("some-error"))

				Expect(notCreatedVM.Start(&vm.StartOpts{})).To(MatchError("failed to start VM: some-error"))
			})
		})

		Context("when there are conflicting vms present", func() {
			It("should return an error", func() {
				mockVBox.EXPECT().ConflictingVMPresent(notCreatedVM.VMConfig).Return(true, nil)

				Expect(notCreatedVM.Start(&vm.StartOpts{})).To(MatchError("old version of PCF Dev already running"))
			})
		})

		Context("when there is an error importing the VM", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockVBox.EXPECT().ConflictingVMPresent(notCreatedVM.VMConfig).Return(false, nil),
					mockUI.EXPECT().Say("Allocating 3072 MB out of 0 MB total system memory (0 MB free)."),
					mockUI.EXPECT().Say("Importing VM..."),
					mockVBox.EXPECT().ImportVM(&config.VMConfig{Name: "some-vm", Memory: uint64(3072)}).Return(errors.New("some-error")),
				)

				Expect(notCreatedVM.Start(&vm.StartOpts{
					Memory: uint64(3072),
				})).To(MatchError("failed to import VM: some-error"))
			})
		})

		Context("when there is an error constructing a stopped VM", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockVBox.EXPECT().ConflictingVMPresent(notCreatedVM.VMConfig).Return(false, nil),
					mockUI.EXPECT().Say("Allocating 3072 MB out of 0 MB total system memory (0 MB free)."),
					mockUI.EXPECT().Say("Importing VM..."),
					mockVBox.EXPECT().ImportVM(&config.VMConfig{Name: "some-vm", Memory: uint64(3072)}).Return(nil),
					mockBuilder.EXPECT().VM("some-vm").Return(nil, errors.New("some-error")),
				)

				Expect(notCreatedVM.Start(&vm.StartOpts{
					Memory: uint64(3072),
				})).To(MatchError("failed to start VM: some-error"))
			})
		})

		Context("when there is an error starting the stopped VM", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockVBox.EXPECT().ConflictingVMPresent(notCreatedVM.VMConfig).Return(false, nil),
					mockUI.EXPECT().Say("Allocating 3072 MB out of 0 MB total system memory (0 MB free)."),
					mockUI.EXPECT().Say("Importing VM..."),
					mockVBox.EXPECT().ImportVM(&config.VMConfig{Name: "some-vm", Memory: uint64(3072)}).Return(nil),
					mockBuilder.EXPECT().VM("some-vm").Return(mockStopped, nil),
					mockStopped.EXPECT().Start(&vm.StartOpts{}).Return(errors.New("failed to start VM: some-error")),
				)

				Expect(notCreatedVM.Start(&vm.StartOpts{
					Memory: uint64(3072),
				})).To(MatchError("failed to start VM: some-error"))
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
})

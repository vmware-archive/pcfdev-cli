package plugin_test

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/pivotal-cf/pcfdev-cli/plugin"
	"github.com/pivotal-cf/pcfdev-cli/plugin/mocks"
	"github.com/pivotal-cf/pcfdev-cli/user"

	"github.com/cloudfoundry/cli/plugin/fakes"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Plugin", func() {
	var (
		mockCtrl                *gomock.Controller
		mockSSH                 *mocks.MockSSH
		mockUI                  *mocks.MockUI
		mockVBox                *mocks.MockVBox
		mockDownloader          *mocks.MockDownloader
		mockClient              *mocks.MockClient
		mockConfig              *mocks.MockConfig
		mockBuilder             *mocks.MockBuilder
		mockVM                  *mocks.MockVM
		mockRequirementsChecker *mocks.MockRequirementsChecker
		pcfdev                  *plugin.Plugin
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockSSH = mocks.NewMockSSH(mockCtrl)
		mockUI = mocks.NewMockUI(mockCtrl)
		mockVBox = mocks.NewMockVBox(mockCtrl)
		mockDownloader = mocks.NewMockDownloader(mockCtrl)
		mockClient = mocks.NewMockClient(mockCtrl)
		mockConfig = mocks.NewMockConfig(mockCtrl)
		mockBuilder = mocks.NewMockBuilder(mockCtrl)
		mockVM = mocks.NewMockVM(mockCtrl)
		mockRequirementsChecker = mocks.NewMockRequirementsChecker(mockCtrl)
		pcfdev = &plugin.Plugin{
			SSH:                 mockSSH,
			UI:                  mockUI,
			VBox:                mockVBox,
			Downloader:          mockDownloader,
			Client:              mockClient,
			Config:              mockConfig,
			Builder:             mockBuilder,
			RequirementsChecker: mockRequirementsChecker,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("Run", func() {
		var home string

		BeforeEach(func() {
			var err error
			home, err = user.GetHome()
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when it is called with the wrong number of arguments", func() {
			It("should print the usage message", func() {
				mockUI.EXPECT().Failed("Usage: %s", "cf dev download|start|status|stop|suspend|resume|destroy")
				pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev"})
			})
		})

		Context("when it is called with an invalid argument", func() {
			It("should print the usage message", func() {
				mockUI.EXPECT().Failed("'%s' is not a registered command.\nUsage: %s", "invalid", "cf dev download|start|status|stop|suspend|resume|destroy")
				pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "invalid"})
			})
		})

		Context("download", func() {
			Context("when OVA is not current", func() {
				It("should download and save the token", func() {
					gomock.InOrder(
						mockDownloader.EXPECT().IsOVACurrent().Return(false, nil),
						mockClient.EXPECT().IsEULAAccepted().Return(true, nil),
						mockUI.EXPECT().Say("Downloading VM..."),
						mockDownloader.EXPECT().Download(),
						mockConfig.EXPECT().SaveToken().Return(nil),
						mockUI.EXPECT().Say("\nVM downloaded"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "download"})
				})

				Context("when EULA check fails", func() {
					It("should print an error", func() {
						gomock.InOrder(
							mockDownloader.EXPECT().IsOVACurrent().Return(false, nil),
							mockClient.EXPECT().IsEULAAccepted().Return(false, errors.New("some-error")),
							mockUI.EXPECT().Failed("Error: some-error"),
						)

						pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "download"})

					})
				})

				Context("when downloading the OVA fails", func() {
					It("should print an error", func() {
						gomock.InOrder(
							mockDownloader.EXPECT().IsOVACurrent().Return(false, nil),
							mockClient.EXPECT().IsEULAAccepted().Return(true, nil),
							mockUI.EXPECT().Say("Downloading VM..."),
							mockDownloader.EXPECT().Download().Return(errors.New("some-error")),
							mockUI.EXPECT().Failed("Error: some-error"),
						)

						pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "download"})
					})
				})

				Context("when saving the API token fails", func() {
					It("should print an error", func() {
						gomock.InOrder(
							mockDownloader.EXPECT().IsOVACurrent().Return(false, nil),
							mockClient.EXPECT().IsEULAAccepted().Return(true, nil),
							mockUI.EXPECT().Say("Downloading VM..."),
							mockDownloader.EXPECT().Download(),
							mockConfig.EXPECT().SaveToken().Return(errors.New("some-error")),
							mockUI.EXPECT().Failed("Error: some-error"),
						)

						pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "download"})

					})
				})

				Context("when EULA has not been accepted and user accepts the EULA", func() {
					It("should download the ova", func() {
						gomock.InOrder(
							mockDownloader.EXPECT().IsOVACurrent().Return(false, nil),
							mockClient.EXPECT().IsEULAAccepted().Return(false, nil),
							mockClient.EXPECT().GetEULA().Return("some-eula", nil),
							mockUI.EXPECT().Say("some-eula"),
							mockUI.EXPECT().Confirm("Accept (yes/no):").Return(true),
							mockClient.EXPECT().AcceptEULA().Return(nil),
							mockUI.EXPECT().Say("Downloading VM..."),
							mockDownloader.EXPECT().Download(),
							mockConfig.EXPECT().SaveToken().Return(nil),
							mockUI.EXPECT().Say("\nVM downloaded"),
						)

						pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "download"})
					})
				})

				Context("when EULA has not been accepted and user denies the EULA", func() {
					It("should not accept and fail gracefully", func() {
						gomock.InOrder(
							mockDownloader.EXPECT().IsOVACurrent().Return(false, nil),
							mockClient.EXPECT().IsEULAAccepted().Return(false, nil),
							mockClient.EXPECT().GetEULA().Return("some-eula", nil),
							mockUI.EXPECT().Say("some-eula"),
							mockUI.EXPECT().Confirm("Accept (yes/no):").Return(false),
							mockUI.EXPECT().Failed("You must accept the end user license agreement to use PCF Dev."),
						)

						pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "download"})
					})
				})

				Context("when EULA has not been accepted and it fails to accept the EULA", func() {
					It("should return the error", func() {
						gomock.InOrder(
							mockDownloader.EXPECT().IsOVACurrent().Return(false, nil),
							mockClient.EXPECT().IsEULAAccepted().Return(false, nil),
							mockClient.EXPECT().GetEULA().Return("some-eula", nil),
							mockUI.EXPECT().Say("some-eula"),
							mockUI.EXPECT().Confirm("Accept (yes/no):").Return(true),
							mockClient.EXPECT().AcceptEULA().Return(errors.New("some-error")),
							mockUI.EXPECT().Failed("Error: some-error"),
						)

						pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "download"})
					})
				})

				Context("when EULA is not accepted and getting the EULA fails", func() {
					It("should print an error", func() {
						gomock.InOrder(
							mockDownloader.EXPECT().IsOVACurrent().Return(false, nil),
							mockClient.EXPECT().IsEULAAccepted().Return(false, nil),
							mockClient.EXPECT().GetEULA().Return("", errors.New("some-error")),
							mockUI.EXPECT().Failed("Error: some-error"),
						)

						pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "download"})
					})
				})
				Context("when PCFDEV_HOME is set", func() {
					var pcfdevHome string

					BeforeEach(func() {
						pcfdevHome = os.Getenv("PCFDEV_HOME")
						os.Setenv("PCFDEV_HOME", filepath.Join("some", "other", "dir"))
					})

					AfterEach(func() {
						os.Setenv("PCFDEV_HOME", pcfdevHome)
					})

					It("should download the ova to PCFDEV_HOME", func() {
						gomock.InOrder(
							mockDownloader.EXPECT().IsOVACurrent().Return(false, nil),
							mockClient.EXPECT().IsEULAAccepted().Return(true, nil),
							mockUI.EXPECT().Say("Downloading VM..."),
							mockDownloader.EXPECT().Download(),
							mockConfig.EXPECT().SaveToken().Return(nil),
							mockUI.EXPECT().Say("\nVM downloaded"),
						)

						pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "download"})
					})
				})
			})

			Context("when OVA is current", func() {
				It("should not download", func() {
					gomock.InOrder(
						mockDownloader.EXPECT().IsOVACurrent().Return(true, nil),
						mockUI.EXPECT().Say("Using existing image"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "download"})
				})
			})

			Context("when calling IsOVACurrent fails", func() {
				It("should return an error", func() {
					gomock.InOrder(
						mockDownloader.EXPECT().IsOVACurrent().Return(false, errors.New("some-error")),
						mockUI.EXPECT().Failed("Error: some-error"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "download"})
				})
			})
		})

		Describe("start", func() {
			Context("when ova is not current", func() {
				It("should download and start the ova", func() {
					gomock.InOrder(
						mockRequirementsChecker.EXPECT().Check().Return(nil),
						mockDownloader.EXPECT().IsOVACurrent().Return(false, nil),
						mockClient.EXPECT().IsEULAAccepted().Return(true, nil),
						mockUI.EXPECT().Say("Downloading VM..."),
						mockDownloader.EXPECT().Download(),
						mockConfig.EXPECT().SaveToken().Return(nil),
						mockUI.EXPECT().Say("\nVM downloaded"),
						mockConfig.EXPECT().GetVMName().Return("some-vm-name"),
						mockBuilder.EXPECT().VM("some-vm-name").Return(mockVM, nil),
						mockVM.EXPECT().Start(),
					)
					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
				})
			})

			Context("when ova is current", func() {
				Context("when PCFDEV_HOME is set", func() {
					var pcfdevHome string

					BeforeEach(func() {
						pcfdevHome = os.Getenv("PCFDEV_HOME")
						os.Setenv("PCFDEV_HOME", filepath.Join("some", "other", "dir"))
					})

					AfterEach(func() {
						os.Setenv("PCFDEV_HOME", pcfdevHome)
					})

					It("should download and start the ova in PCFDEV_HOME", func() {
						gomock.InOrder(
							mockRequirementsChecker.EXPECT().Check().Return(nil),
							mockDownloader.EXPECT().IsOVACurrent().Return(true, nil),
							mockUI.EXPECT().Say("Using existing image"),
							mockConfig.EXPECT().GetVMName().Return("some-vm-name"),
							mockBuilder.EXPECT().VM("some-vm-name").Return(mockVM, nil),
							mockVM.EXPECT().Start(),
						)
						pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
					})
				})

				It("should start without downloading", func() {
					gomock.InOrder(
						mockRequirementsChecker.EXPECT().Check().Return(nil),
						mockDownloader.EXPECT().IsOVACurrent().Return(true, nil),
						mockUI.EXPECT().Say("Using existing image"),
						mockConfig.EXPECT().GetVMName().Return("some-vm-name"),
						mockBuilder.EXPECT().VM("some-vm-name").Return(mockVM, nil),
						mockVM.EXPECT().Start(),
					)
					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})

				})

				Context("when it fails to get VM", func() {
					It("should return an error", func() {
						gomock.InOrder(
							mockRequirementsChecker.EXPECT().Check().Return(nil),
							mockDownloader.EXPECT().IsOVACurrent().Return(true, nil),
							mockUI.EXPECT().Say("Using existing image"),
							mockConfig.EXPECT().GetVMName().Return("some-vm-name"),
							mockBuilder.EXPECT().VM("some-vm-name").Return(nil, errors.New("some-error")),
							mockUI.EXPECT().Failed("Error: some-error"),
						)

						pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
					})
				})

				Context("when it fails to start VM", func() {
					It("should return an error", func() {
						gomock.InOrder(
							mockRequirementsChecker.EXPECT().Check().Return(nil),
							mockDownloader.EXPECT().IsOVACurrent().Return(true, nil),
							mockUI.EXPECT().Say("Using existing image"),
							mockConfig.EXPECT().GetVMName().Return("some-vm-name"),
							mockBuilder.EXPECT().VM("some-vm-name").Return(mockVM, nil),
							mockVM.EXPECT().Start().Return(errors.New("some-error")),
							mockUI.EXPECT().Failed("Error: some-error"),
						)

						pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
					})
				})

			})
		})

		Context("when the OVA fails to download", func() {
			It("should print an error message", func() {
				gomock.InOrder(
					mockRequirementsChecker.EXPECT().Check().Return(nil),
					mockDownloader.EXPECT().IsOVACurrent().Return(false, nil),
					mockClient.EXPECT().IsEULAAccepted().Return(true, nil),
					mockUI.EXPECT().Say("Downloading VM..."),
					mockDownloader.EXPECT().Download().Return(errors.New("some-error")),
					mockUI.EXPECT().Failed("Error: some-error"),
				)

				pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
			})
		})

		Context("when the EULA is not accepted", func() {
			It("should print the EULA", func() {
				gomock.InOrder(
					mockRequirementsChecker.EXPECT().Check().Return(nil),
					mockDownloader.EXPECT().IsOVACurrent().Return(false, nil),
					mockClient.EXPECT().IsEULAAccepted().Return(false, nil),
					mockClient.EXPECT().GetEULA().Return("some-eula", nil),
					mockUI.EXPECT().Say("some-eula"),
					mockUI.EXPECT().Confirm("Accept (yes/no):").Return(true),
					mockClient.EXPECT().AcceptEULA().Return(nil),

					mockUI.EXPECT().Say("Downloading VM..."),
					mockDownloader.EXPECT().Download(),
					mockConfig.EXPECT().SaveToken().Return(nil),
					mockUI.EXPECT().Say("\nVM downloaded"),

					mockConfig.EXPECT().GetVMName().Return("some-vm-name"),
					mockBuilder.EXPECT().VM("some-vm-name").Return(mockVM, nil),
					mockVM.EXPECT().Start(),
				)
				pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
			})
		})

		Context("when the system does not meet requirements and the user accepts to continue", func() {
			It("should print a warning and prompt for the response to continue", func() {
				gomock.InOrder(
					mockRequirementsChecker.EXPECT().Check().Return(errors.New("some-message")),
					mockUI.EXPECT().Confirm("Less than 3 GB of memory detected, continue (y/N): ").Return(true),
					mockDownloader.EXPECT().IsOVACurrent().Return(true, nil),
					mockUI.EXPECT().Say("Using existing image"),
					mockConfig.EXPECT().GetVMName().Return("some-vm-name"),
					mockBuilder.EXPECT().VM("some-vm-name").Return(mockVM, nil),
					mockVM.EXPECT().Start(),
				)

				pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
			})
		})

		Context("when the system does not meet requirements and the user declines to continue", func() {
			It("should print a warning and prompt for the response to continue", func() {
				gomock.InOrder(
					mockRequirementsChecker.EXPECT().Check().Return(errors.New("some-message")),
					mockUI.EXPECT().Confirm("Less than 3 GB of memory detected, continue (y/N): ").Return(false),
					mockUI.EXPECT().Say("Exiting..."),
				)

				pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
			})
		})
	})

	Context("stop", func() {
		It("should stop the VM", func() {
			gomock.InOrder(
				mockConfig.EXPECT().GetVMName().Return("some-vm-name"),
				mockBuilder.EXPECT().VM("some-vm-name").Return(mockVM, nil),
				mockVM.EXPECT().Stop(),
			)

			pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "stop"})
		})

		Context("when it fails to get VM", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockConfig.EXPECT().GetVMName().Return("some-vm-name"),
					mockBuilder.EXPECT().VM("some-vm-name").Return(nil, errors.New("some-error")),
					mockUI.EXPECT().Failed("Error: some-error"),
				)

				pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "stop"})
			})
		})

		Context("when it fails to stop VM", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockConfig.EXPECT().GetVMName().Return("some-vm-name"),
					mockBuilder.EXPECT().VM("some-vm-name").Return(mockVM, nil),
					mockVM.EXPECT().Stop().Return(errors.New("some-error")),
					mockUI.EXPECT().Failed("Error: some-error"),
				)

				pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "stop"})
			})
		})
	})

	Context("suspend", func() {
		It("should suspend the VM", func() {
			gomock.InOrder(
				mockConfig.EXPECT().GetVMName().Return("some-vm-name"),
				mockBuilder.EXPECT().VM("some-vm-name").Return(mockVM, nil),
				mockVM.EXPECT().Suspend(),
			)

			pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "suspend"})
		})

		Context("when it fails to get VM", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockConfig.EXPECT().GetVMName().Return("some-vm-name"),
					mockBuilder.EXPECT().VM("some-vm-name").Return(nil, errors.New("some-error")),
					mockUI.EXPECT().Failed("Error: some-error"),
				)

				pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "suspend"})
			})
		})

		Context("when it fails to suspend VM", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockConfig.EXPECT().GetVMName().Return("some-vm-name"),
					mockBuilder.EXPECT().VM("some-vm-name").Return(mockVM, nil),
					mockVM.EXPECT().Suspend().Return(errors.New("some-error")),
					mockUI.EXPECT().Failed("Error: some-error"),
				)

				pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "suspend"})
			})
		})
	})

	Context("resume", func() {
		It("should resume the VM", func() {
			gomock.InOrder(
				mockRequirementsChecker.EXPECT().Check().Return(nil),
				mockConfig.EXPECT().GetVMName().Return("some-vm-name"),
				mockBuilder.EXPECT().VM("some-vm-name").Return(mockVM, nil),
				mockVM.EXPECT().Resume(),
			)

			pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "resume"})
		})

		Context("when it fails to get VM", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockRequirementsChecker.EXPECT().Check().Return(nil),
					mockConfig.EXPECT().GetVMName().Return("some-vm-name"),
					mockBuilder.EXPECT().VM("some-vm-name").Return(nil, errors.New("some-error")),
					mockUI.EXPECT().Failed("Error: some-error"),
				)

				pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "resume"})
			})
		})

		Context("when it fails to resume VM", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockRequirementsChecker.EXPECT().Check().Return(nil),
					mockConfig.EXPECT().GetVMName().Return("some-vm-name"),
					mockBuilder.EXPECT().VM("some-vm-name").Return(mockVM, nil),
					mockVM.EXPECT().Resume().Return(errors.New("some-error")),
					mockUI.EXPECT().Failed("Error: some-error"),
				)

				pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "resume"})
			})
		})

		Context("when the system does not meet requirements and the user accepts to continue", func() {
			It("should print a warning and prompt for the response to continue", func() {
				gomock.InOrder(
					mockRequirementsChecker.EXPECT().Check().Return(errors.New("some-message")),
					mockUI.EXPECT().Confirm("Less than 3 GB of memory detected, continue (y/N): ").Return(true),
					mockConfig.EXPECT().GetVMName().Return("some-vm-name"),
					mockBuilder.EXPECT().VM("some-vm-name").Return(mockVM, nil),
					mockVM.EXPECT().Resume(),
				)

				pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "resume"})
			})
		})

		Context("when the system does not meet requirements and the user declines to continue", func() {
			It("should print a warning and prompt for the response to continue", func() {
				gomock.InOrder(
					mockRequirementsChecker.EXPECT().Check().Return(errors.New("some-message")),
					mockUI.EXPECT().Confirm("Less than 3 GB of memory detected, continue (y/N): ").Return(false),
					mockUI.EXPECT().Say("Exiting..."),
				)

				pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "resume"})
			})
		})
	})

	Context("status", func() {
		It("should return the status", func() {
			gomock.InOrder(
				mockConfig.EXPECT().GetVMName().Return("some-vm-name"),
				mockBuilder.EXPECT().VM("some-vm-name").Return(mockVM, nil),
				mockVM.EXPECT().Status(),
			)

			pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "status"})
		})

		Context("when it fails to get VM", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockConfig.EXPECT().GetVMName().Return("some-vm-name"),
					mockBuilder.EXPECT().VM("some-vm-name").Return(nil, errors.New("some-error")),
					mockUI.EXPECT().Failed("Error: some-error"),
				)

				pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "status"})
			})
		})
	})

	Context("destroy", func() {
		var mockVM2 *mocks.MockVM

		BeforeEach(func() {
			mockVM2 = mocks.NewMockVM(mockCtrl)
		})

		It("should destroy all PCF Dev VMs created by the CLI", func() {
			vms := []string{"pcfdev-0.0.0", "pcfdev-0.0.1"}
			gomock.InOrder(
				mockVBox.EXPECT().GetPCFDevVMs().Return(vms, nil),
				mockUI.EXPECT().Say("Destroying VM..."),
				mockBuilder.EXPECT().VM("pcfdev-0.0.0").Return(mockVM, nil),
				mockVM.EXPECT().Destroy().Return(nil),
				mockBuilder.EXPECT().VM("pcfdev-0.0.1").Return(mockVM2, nil),
				mockVM2.EXPECT().Destroy().Return(nil),
				mockUI.EXPECT().Say("PCF Dev VM has been destroyed"),
			)

			pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "destroy"})
		})

		Context("there are no PCF Dev VMs", func() {
			It("should send an error message", func() {
				gomock.InOrder(
					mockVBox.EXPECT().GetPCFDevVMs().Return([]string{}, nil),
					mockUI.EXPECT().Say("PCF Dev VM has not been created"),
				)

				pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "destroy"})
			})
		})

		Context("there is an error getting the PCFDev names", func() {
			It("should send an error message", func() {
				gomock.InOrder(
					mockVBox.EXPECT().GetPCFDevVMs().Return([]string{}, errors.New("some-error")),
					mockUI.EXPECT().Failed("Failed to destroy PCF Dev VM."),
				)

				pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "destroy"})
			})
		})

		Context("there is an error building the VMs", func() {
			It("should send an error message", func() {
				vms := []string{"pcfdev-0.0.0"}
				gomock.InOrder(
					mockVBox.EXPECT().GetPCFDevVMs().Return(vms, nil),
					mockUI.EXPECT().Say("Destroying VM..."),
					mockBuilder.EXPECT().VM("pcfdev-0.0.0").Return(nil, errors.New("some-error")),
					mockUI.EXPECT().Failed("Failed to destroy PCF Dev VM."),
				)

				pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "destroy"})
			})
		})

		Context("there is an error destroying the VMs", func() {
			It("should send an error message", func() {
				vms := []string{"pcfdev-0.0.0"}
				gomock.InOrder(
					mockVBox.EXPECT().GetPCFDevVMs().Return(vms, nil),
					mockUI.EXPECT().Say("Destroying VM..."),
					mockBuilder.EXPECT().VM("pcfdev-0.0.0").Return(mockVM, nil),
					mockVM.EXPECT().Destroy().Return(errors.New("some-error")),
					mockUI.EXPECT().Failed("Failed to destroy PCF Dev VM."),
				)

				pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "destroy"})
			})
		})
	})

	Context("uninstalling plugin", func() {
		It("returns immediately", func() {
			pcfdev.Run(&fakes.FakeCliConnection{}, []string{"CLI-MESSAGE-UNINSTALL"})
		})
	})
})

package plugin_test

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/pivotal-cf/pcfdev-cli/config"
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
		mockBuilder             *mocks.MockBuilder
		mockVM                  *mocks.MockVM
		mockRequirementsChecker *mocks.MockRequirementsChecker
		fakeCliConnection       *fakes.FakeCliConnection
		pcfdev                  *plugin.Plugin

		desiredMemory = uint64(3072)
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockSSH = mocks.NewMockSSH(mockCtrl)
		mockUI = mocks.NewMockUI(mockCtrl)
		mockVBox = mocks.NewMockVBox(mockCtrl)
		mockDownloader = mocks.NewMockDownloader(mockCtrl)
		mockClient = mocks.NewMockClient(mockCtrl)
		mockBuilder = mocks.NewMockBuilder(mockCtrl)
		mockVM = mocks.NewMockVM(mockCtrl)
		mockRequirementsChecker = mocks.NewMockRequirementsChecker(mockCtrl)
		fakeCliConnection = &fakes.FakeCliConnection{}
		pcfdev = &plugin.Plugin{
			SSH:        mockSSH,
			UI:         mockUI,
			VBox:       mockVBox,
			Downloader: mockDownloader,
			Client:     mockClient,
			Config: &config.Config{
				DefaultVMName: "some-vm-name",
			},
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

		Context("help", func() {
			Context("when it is called with no subcommand", func() {
				It("should print the usage message", func() {
					pcfdev.Run(fakeCliConnection, []string{"dev"})

					Expect(fakeCliConnection.CliCommandArgsForCall(0)[0]).To(Equal("help"))
					Expect(fakeCliConnection.CliCommandArgsForCall(0)[1]).To(Equal("dev"))
				})
			})

			Context("when it is called with an invalid subcommand", func() {
				It("should print the usage message", func() {
					pcfdev.Run(fakeCliConnection, []string{"dev", "some-bad-subcommand"})

					Expect(fakeCliConnection.CliCommandArgsForCall(0)[0]).To(Equal("help"))
					Expect(fakeCliConnection.CliCommandArgsForCall(0)[1]).To(Equal("dev"))
				})
			})

			Context("when it is called with help", func() {
				It("should print the usage message", func() {
					pcfdev.Run(fakeCliConnection, []string{"dev", "help"})

					Expect(fakeCliConnection.CliCommandArgsForCall(0)[0]).To(Equal("help"))
					Expect(fakeCliConnection.CliCommandArgsForCall(0)[1]).To(Equal("dev"))
				})
			})

			Context("when it is called with --help", func() {
				It("should print the usage message", func() {
					pcfdev.Run(fakeCliConnection, []string{"dev", "--help"})

					Expect(fakeCliConnection.CliCommandArgsForCall(0)[0]).To(Equal("help"))
					Expect(fakeCliConnection.CliCommandArgsForCall(0)[1]).To(Equal("dev"))
				})
			})

			Context("when printing the help text fails", func() {
				It("should print an error", func() {
					fakeCliConnection.CliCommandReturns(nil, errors.New("some-error"))
					mockUI.EXPECT().Failed("Error: some-error")

					pcfdev.Run(fakeCliConnection, []string{"dev", "help"})
				})
			})
		})

		Context("download", func() {
			Context("when OVA is not current", func() {
				It("should download the OVA", func() {
					gomock.InOrder(
						mockDownloader.EXPECT().IsOVACurrent().Return(false, nil),
						mockClient.EXPECT().IsEULAAccepted().Return(true, nil),
						mockUI.EXPECT().Say("Downloading VM..."),
						mockDownloader.EXPECT().Download(),
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
							mockUI.EXPECT().Failed("Error: you must accept the end user license agreement to use PCF Dev"),
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
						mockRequirementsChecker.EXPECT().Check(desiredMemory).Return(nil),
						mockDownloader.EXPECT().IsOVACurrent().Return(false, nil),
						mockClient.EXPECT().IsEULAAccepted().Return(true, nil),
						mockUI.EXPECT().Say("Downloading VM..."),
						mockDownloader.EXPECT().Download(),
						mockUI.EXPECT().Say("\nVM downloaded"),
						mockBuilder.EXPECT().VM("some-vm-name", &config.VMConfig{DesiredMemory: desiredMemory}).Return(mockVM, nil),
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
							mockRequirementsChecker.EXPECT().Check(desiredMemory).Return(nil),
							mockDownloader.EXPECT().IsOVACurrent().Return(true, nil),
							mockUI.EXPECT().Say("Using existing image"),
							mockBuilder.EXPECT().VM("some-vm-name", &config.VMConfig{DesiredMemory: desiredMemory}).Return(mockVM, nil),
							mockVM.EXPECT().Start(),
						)
						pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
					})
				})

				It("should start without downloading", func() {
					gomock.InOrder(
						mockRequirementsChecker.EXPECT().Check(desiredMemory).Return(nil),
						mockDownloader.EXPECT().IsOVACurrent().Return(true, nil),
						mockUI.EXPECT().Say("Using existing image"),
						mockBuilder.EXPECT().VM("some-vm-name", &config.VMConfig{DesiredMemory: desiredMemory}).Return(mockVM, nil),
						mockVM.EXPECT().Start(),
					)
					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})

				})

				Context("when it fails to get VM", func() {
					It("should return an error", func() {
						gomock.InOrder(
							mockRequirementsChecker.EXPECT().Check(desiredMemory).Return(nil),
							mockDownloader.EXPECT().IsOVACurrent().Return(true, nil),
							mockUI.EXPECT().Say("Using existing image"),
							mockBuilder.EXPECT().VM("some-vm-name", &config.VMConfig{DesiredMemory: desiredMemory}).Return(nil, errors.New("some-error")),
							mockUI.EXPECT().Failed("Error: some-error"),
						)

						pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
					})
				})

				Context("when it fails to start VM", func() {
					It("should return an error", func() {
						gomock.InOrder(
							mockRequirementsChecker.EXPECT().Check(desiredMemory).Return(nil),
							mockDownloader.EXPECT().IsOVACurrent().Return(true, nil),
							mockUI.EXPECT().Say("Using existing image"),
							mockBuilder.EXPECT().VM("some-vm-name", &config.VMConfig{DesiredMemory: desiredMemory}).Return(mockVM, nil),
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
					mockRequirementsChecker.EXPECT().Check(desiredMemory).Return(nil),
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
					mockRequirementsChecker.EXPECT().Check(desiredMemory).Return(nil),
					mockDownloader.EXPECT().IsOVACurrent().Return(false, nil),
					mockClient.EXPECT().IsEULAAccepted().Return(false, nil),
					mockClient.EXPECT().GetEULA().Return("some-eula", nil),
					mockUI.EXPECT().Say("some-eula"),
					mockUI.EXPECT().Confirm("Accept (yes/no):").Return(true),
					mockClient.EXPECT().AcceptEULA().Return(nil),

					mockUI.EXPECT().Say("Downloading VM..."),
					mockDownloader.EXPECT().Download(),
					mockUI.EXPECT().Say("\nVM downloaded"),

					mockBuilder.EXPECT().VM("some-vm-name", &config.VMConfig{DesiredMemory: desiredMemory}).Return(mockVM, nil),
					mockVM.EXPECT().Start(),
				)
				pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
			})
		})

		Context("when the system does not meet requirements and the user accepts to continue", func() {
			It("should print a warning and prompt for the response to continue", func() {
				gomock.InOrder(
					mockRequirementsChecker.EXPECT().Check(desiredMemory).Return(errors.New("some-message")),
					mockUI.EXPECT().Confirm("Less than 3 GB of memory detected, continue (y/N): ").Return(true),
					mockDownloader.EXPECT().IsOVACurrent().Return(true, nil),
					mockUI.EXPECT().Say("Using existing image"),
					mockBuilder.EXPECT().VM("some-vm-name", &config.VMConfig{DesiredMemory: desiredMemory}).Return(mockVM, nil),
					mockVM.EXPECT().Start(),
				)

				pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
			})
		})

		Context("when the system does not meet requirements and the user declines to continue", func() {
			It("should print a warning and prompt for the response to continue", func() {
				gomock.InOrder(
					mockRequirementsChecker.EXPECT().Check(desiredMemory).Return(errors.New("some-message")),
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
				mockBuilder.EXPECT().VM("some-vm-name", &config.VMConfig{}).Return(mockVM, nil),
				mockVM.EXPECT().Stop(),
			)

			pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "stop"})
		})

		Context("when it fails to get VM", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockBuilder.EXPECT().VM("some-vm-name", &config.VMConfig{}).Return(nil, errors.New("some-error")),
					mockUI.EXPECT().Failed("Error: some-error"),
				)

				pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "stop"})
			})
		})

		Context("when it fails to stop VM", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockBuilder.EXPECT().VM("some-vm-name", &config.VMConfig{}).Return(mockVM, nil),
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
				mockBuilder.EXPECT().VM("some-vm-name", &config.VMConfig{}).Return(mockVM, nil),
				mockVM.EXPECT().Suspend(),
			)

			pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "suspend"})
		})

		Context("when it fails to get VM", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockBuilder.EXPECT().VM("some-vm-name", &config.VMConfig{}).Return(nil, errors.New("some-error")),
					mockUI.EXPECT().Failed("Error: some-error"),
				)

				pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "suspend"})
			})
		})

		Context("when it fails to suspend VM", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockBuilder.EXPECT().VM("some-vm-name", &config.VMConfig{}).Return(mockVM, nil),
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
				mockBuilder.EXPECT().VM("some-vm-name").Return(mockVM, nil),
				mockVM.EXPECT().Resume(),
			)

			pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "resume"})
		})

		Context("when it fails to get VM", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockRequirementsChecker.EXPECT().Check().Return(nil),
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
				mockBuilder.EXPECT().VM("some-vm-name", &config.VMConfig{}).Return(mockVM, nil),
				mockVM.EXPECT().Status(),
			)

			pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "status"})
		})

		Context("when it fails to get VM", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockBuilder.EXPECT().VM("some-vm-name", &config.VMConfig{}).Return(nil, errors.New("some-error")),
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
				mockBuilder.EXPECT().VM("pcfdev-0.0.0", &config.VMConfig{}).Return(mockVM, nil),
				mockVM.EXPECT().Destroy().Return(nil),
				mockBuilder.EXPECT().VM("pcfdev-0.0.1", &config.VMConfig{}).Return(mockVM2, nil),
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
					mockUI.EXPECT().Failed("Error: failed to destroy VM: some-error"),
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
					mockBuilder.EXPECT().VM("pcfdev-0.0.0", &config.VMConfig{}).Return(nil, errors.New("some-error")),
					mockUI.EXPECT().Failed("Error: failed to destroy VM: some-error"),
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
					mockBuilder.EXPECT().VM("pcfdev-0.0.0", &config.VMConfig{}).Return(mockVM, nil),
					mockVM.EXPECT().Destroy().Return(errors.New("some-error")),
					mockUI.EXPECT().Failed("Error: failed to destroy VM: some-error"),
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

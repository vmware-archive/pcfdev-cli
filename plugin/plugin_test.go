package plugin_test

import (
	"errors"

	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/plugin"
	"github.com/pivotal-cf/pcfdev-cli/plugin/mocks"
	"github.com/pivotal-cf/pcfdev-cli/user"
	"github.com/pivotal-cf/pcfdev-cli/vm"

	"github.com/cloudfoundry/cli/plugin/fakes"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Plugin", func() {
	var (
		mockCtrl          *gomock.Controller
		mockSSH           *mocks.MockSSH
		mockUI            *mocks.MockUI
		mockVBox          *mocks.MockVBox
		mockDownloader    *mocks.MockDownloader
		mockClient        *mocks.MockClient
		mockBuilder       *mocks.MockBuilder
		mockVM            *mocks.MockVM
		fakeCliConnection *fakes.FakeCliConnection
		pcfdev            *plugin.Plugin
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
		fakeCliConnection = &fakes.FakeCliConnection{}
		pcfdev = &plugin.Plugin{
			SSH:        mockSSH,
			UI:         mockUI,
			VBox:       mockVBox,
			Downloader: mockDownloader,
			Client:     mockClient,
			Config: &config.Config{
				DefaultVMName: "some-vm-name",
				MinMemory:     uint64(3024),
			},
			Builder: mockBuilder,
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

			Context("when it is called with an invalid argument", func() {
				It("should print the usage message", func() {
					pcfdev.Run(fakeCliConnection, []string{"dev", "download", "-m"})

					Expect(fakeCliConnection.CliCommandArgsForCall(0)[0]).To(Equal("help"))
					Expect(fakeCliConnection.CliCommandArgsForCall(0)[1]).To(Equal("dev"))
				})
			})
		})

		Describe("start", func() {
			It("validates start options and starts the VM", func() {
				startOpts := &vm.StartOpts{
					Memory: uint64(3456),
					CPUs:   2,
				}
				gomock.InOrder(
					mockBuilder.EXPECT().VM("some-vm-name").Return(mockVM, nil),
					mockVM.EXPECT().VerifyStartOpts(startOpts).Return(nil),
					mockDownloader.EXPECT().IsOVACurrent().Return(false, nil),
					mockClient.EXPECT().IsEULAAccepted().Return(true, nil),
					mockUI.EXPECT().Say("Downloading VM..."),
					mockDownloader.EXPECT().Download(),
					mockUI.EXPECT().Say("\nVM downloaded"),
					mockVM.EXPECT().Start(startOpts),
				)
				pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start", "-m", "3456", "-c", "2"})
			})

			Context("when the user does not use the allocated memory flag", func() {
				It("should download and start the ova with the builder specified memory", func() {
					gomock.InOrder(
						mockBuilder.EXPECT().VM("some-vm-name").Return(mockVM, nil),
						mockVM.EXPECT().VerifyStartOpts(&vm.StartOpts{}).Return(nil),
						mockDownloader.EXPECT().IsOVACurrent().Return(false, nil),
						mockClient.EXPECT().IsEULAAccepted().Return(true, nil),
						mockUI.EXPECT().Say("Downloading VM..."),
						mockDownloader.EXPECT().Download(),
						mockUI.EXPECT().Say("\nVM downloaded"),
						mockVM.EXPECT().Start(&vm.StartOpts{}),
					)
					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
				})

				Context("when ova is current", func() {
					It("should start without downloading", func() {
						gomock.InOrder(
							mockBuilder.EXPECT().VM("some-vm-name").Return(mockVM, nil),
							mockVM.EXPECT().VerifyStartOpts(&vm.StartOpts{}).Return(nil),
							mockDownloader.EXPECT().IsOVACurrent().Return(true, nil),
							mockUI.EXPECT().Say("Using existing image"),
							mockVM.EXPECT().Start(&vm.StartOpts{}),
						)
						pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})

					})

					Context("when it fails to get VM", func() {
						It("should return an error", func() {
							gomock.InOrder(
								mockBuilder.EXPECT().VM("some-vm-name").Return(nil, errors.New("some-error")),
								mockUI.EXPECT().Failed("Error: some-error"),
							)

							pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
						})
					})

					Context("when verifying start options fails", func() {
						It("should return an error", func() {
							gomock.InOrder(
								mockBuilder.EXPECT().VM("some-vm-name").Return(mockVM, nil),
								mockVM.EXPECT().VerifyStartOpts(&vm.StartOpts{}).Return(errors.New("some-error")),
								mockUI.EXPECT().Failed("Error: some-error"),
							)

							pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
						})
					})

					Context("when it fails to start VM", func() {
						It("should return an error", func() {
							gomock.InOrder(
								mockBuilder.EXPECT().VM("some-vm-name").Return(mockVM, nil),
								mockVM.EXPECT().VerifyStartOpts(&vm.StartOpts{}).Return(nil),
								mockDownloader.EXPECT().IsOVACurrent().Return(true, nil),
								mockUI.EXPECT().Say("Using existing image"),
								mockVM.EXPECT().Start(&vm.StartOpts{}).Return(errors.New("some-error")),
								mockUI.EXPECT().Failed("Error: some-error"),
							)

							pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
						})
					})
				})

				Context("when ova is not current", func() {
					Context("when the OVA fails to download", func() {
						It("should print an error message", func() {
							gomock.InOrder(
								mockBuilder.EXPECT().VM("some-vm-name").Return(mockVM, nil),
								mockVM.EXPECT().VerifyStartOpts(&vm.StartOpts{}).Return(nil),
								mockDownloader.EXPECT().IsOVACurrent().Return(false, nil),
								mockClient.EXPECT().IsEULAAccepted().Return(true, nil),
								mockUI.EXPECT().Say("Downloading VM..."),
								mockDownloader.EXPECT().Download().Return(errors.New("some-error")),
								mockUI.EXPECT().Failed("Error: some-error"),
							)

							pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
						})
					})

					Context("when the EULA is not already accepted", func() {
						It("should print the EULA", func() {
							gomock.InOrder(
								mockBuilder.EXPECT().VM("some-vm-name").Return(mockVM, nil),
								mockVM.EXPECT().VerifyStartOpts(&vm.StartOpts{}).Return(nil),
								mockDownloader.EXPECT().IsOVACurrent().Return(false, nil),
								mockClient.EXPECT().IsEULAAccepted().Return(false, nil),
								mockClient.EXPECT().GetEULA().Return("some-eula", nil),
								mockUI.EXPECT().Say("some-eula"),
								mockUI.EXPECT().Confirm("Accept (yes/no):").Return(true),
								mockClient.EXPECT().AcceptEULA().Return(nil),

								mockUI.EXPECT().Say("Downloading VM..."),
								mockDownloader.EXPECT().Download(),
								mockUI.EXPECT().Say("\nVM downloaded"),

								mockVM.EXPECT().Start(&vm.StartOpts{}),
							)
							pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
						})
					})
				})
			})
		})
	})

	Context("stop", func() {
		It("should stop the VM", func() {
			gomock.InOrder(
				mockBuilder.EXPECT().VM("some-vm-name").Return(mockVM, nil),
				mockVM.EXPECT().Stop(),
			)

			pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "stop"})
		})

		Context("when it fails to get VM", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockBuilder.EXPECT().VM("some-vm-name").Return(nil, errors.New("some-error")),
					mockUI.EXPECT().Failed("Error: some-error"),
				)

				pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "stop"})
			})
		})

		Context("when it fails to stop VM", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockBuilder.EXPECT().VM("some-vm-name").Return(mockVM, nil),
					mockVM.EXPECT().Stop().Return(errors.New("some-error")),
					mockUI.EXPECT().Failed("Error: some-error"),
				)

				pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "stop"})
			})
		})

		Context("when it is called with an invalid argument", func() {
			It("should print the usage message", func() {
				pcfdev.Run(fakeCliConnection, []string{"dev", "stop", "-m"})

				Expect(fakeCliConnection.CliCommandArgsForCall(0)[0]).To(Equal("help"))
				Expect(fakeCliConnection.CliCommandArgsForCall(0)[1]).To(Equal("dev"))
			})
		})
	})

	Context("suspend", func() {
		It("should suspend the VM", func() {
			gomock.InOrder(
				mockBuilder.EXPECT().VM("some-vm-name").Return(mockVM, nil),
				mockVM.EXPECT().Suspend(),
			)

			pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "suspend"})
		})

		Context("when it fails to get VM", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockBuilder.EXPECT().VM("some-vm-name").Return(nil, errors.New("some-error")),
					mockUI.EXPECT().Failed("Error: some-error"),
				)

				pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "suspend"})
			})
		})

		Context("when it fails to suspend VM", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockBuilder.EXPECT().VM("some-vm-name").Return(mockVM, nil),
					mockVM.EXPECT().Suspend().Return(errors.New("some-error")),
					mockUI.EXPECT().Failed("Error: some-error"),
				)

				pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "suspend"})
			})
		})

		Context("when it is called with an invalid argument", func() {
			It("should print the usage message", func() {
				pcfdev.Run(fakeCliConnection, []string{"dev", "suspend", "-m"})

				Expect(fakeCliConnection.CliCommandArgsForCall(0)[0]).To(Equal("help"))
				Expect(fakeCliConnection.CliCommandArgsForCall(0)[1]).To(Equal("dev"))
			})
		})
	})

	Context("resume", func() {
		It("should resume the VM", func() {
			gomock.InOrder(
				mockBuilder.EXPECT().VM("some-vm-name").Return(mockVM, nil),
				mockVM.EXPECT().Resume(),
			)

			pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "resume"})
		})

		Context("when it fails to get VM", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockBuilder.EXPECT().VM("some-vm-name").Return(nil, errors.New("some-error")),
					mockUI.EXPECT().Failed("Error: some-error"),
				)

				pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "resume"})
			})
		})

		Context("when it fails to resume VM", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockBuilder.EXPECT().VM("some-vm-name").Return(mockVM, nil),
					mockVM.EXPECT().Resume().Return(errors.New("some-error")),
					mockUI.EXPECT().Failed("Error: some-error"),
				)

				pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "resume"})
			})
		})

		Context("when it is called with an invalid argument", func() {
			It("should print the usage message", func() {
				pcfdev.Run(fakeCliConnection, []string{"dev", "resume", "-m"})

				Expect(fakeCliConnection.CliCommandArgsForCall(0)[0]).To(Equal("help"))
				Expect(fakeCliConnection.CliCommandArgsForCall(0)[1]).To(Equal("dev"))
			})
		})
	})

	Context("status", func() {
		It("should return the status", func() {
			gomock.InOrder(
				mockBuilder.EXPECT().VM("some-vm-name").Return(mockVM, nil),
				mockVM.EXPECT().Status().Return("some-status"),
				mockUI.EXPECT().Say("some-status"),
			)

			pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "status"})
		})

		Context("when it fails to get VM", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockBuilder.EXPECT().VM("some-vm-name").Return(nil, errors.New("some-error")),
					mockUI.EXPECT().Failed("Error: some-error"),
				)

				pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "status"})
			})
		})

		Context("when it is called with an invalid argument", func() {
			It("should print the usage message", func() {
				pcfdev.Run(fakeCliConnection, []string{"dev", "status", "-m"})

				Expect(fakeCliConnection.CliCommandArgsForCall(0)[0]).To(Equal("help"))
				Expect(fakeCliConnection.CliCommandArgsForCall(0)[1]).To(Equal("dev"))
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
					mockBuilder.EXPECT().VM("pcfdev-0.0.0").Return(nil, errors.New("some-error")),
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
					mockBuilder.EXPECT().VM("pcfdev-0.0.0").Return(mockVM, nil),
					mockVM.EXPECT().Destroy().Return(errors.New("some-error")),
					mockUI.EXPECT().Failed("Error: failed to destroy VM: some-error"),
				)

				pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "destroy"})
			})
		})

		Context("when it is called with an invalid argument", func() {
			It("should print the usage message", func() {
				pcfdev.Run(fakeCliConnection, []string{"dev", "destroy", "-m"})

				Expect(fakeCliConnection.CliCommandArgsForCall(0)[0]).To(Equal("help"))
				Expect(fakeCliConnection.CliCommandArgsForCall(0)[1]).To(Equal("dev"))
			})
		})
	})

	Context("uninstalling plugin", func() {
		It("returns immediately", func() {
			pcfdev.Run(&fakes.FakeCliConnection{}, []string{"CLI-MESSAGE-UNINSTALL"})
		})
	})
})

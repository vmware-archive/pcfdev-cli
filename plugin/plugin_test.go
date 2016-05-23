package plugin_test

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/pivotal-cf/pcfdev-cli/plugin"
	"github.com/pivotal-cf/pcfdev-cli/plugin/mocks"
	"github.com/pivotal-cf/pcfdev-cli/user"
	"github.com/pivotal-cf/pcfdev-cli/vbox"

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
		mockRequirementsChecker *mocks.MockRequirementsChecker
		mockClient              *mocks.MockClient
		pcfdev                  *plugin.Plugin
		vm                      *vbox.VM
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockSSH = mocks.NewMockSSH(mockCtrl)
		mockUI = mocks.NewMockUI(mockCtrl)
		mockVBox = mocks.NewMockVBox(mockCtrl)
		mockDownloader = mocks.NewMockDownloader(mockCtrl)
		mockClient = mocks.NewMockClient(mockCtrl)
		mockRequirementsChecker = mocks.NewMockRequirementsChecker(mockCtrl)
		pcfdev = &plugin.Plugin{
			SSH:                 mockSSH,
			UI:                  mockUI,
			VBox:                mockVBox,
			Downloader:          mockDownloader,
			RequirementsChecker: mockRequirementsChecker,
			Client:              mockClient,
			VMName:              "some-vm-name",
		}
		vm = &vbox.VM{
			IP:      "some-ip",
			SSHPort: "some-port",
			Domain:  "some-domain",
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
				mockUI.EXPECT().Failed("Usage: %s", "cf dev download|start|status|stop|destroy")
				pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev"})
			})
		})

		Context("when it is called with an invalid argument", func() {
			It("should print the usage message", func() {
				mockUI.EXPECT().Failed("'%s' is not a registered command.\nUsage: %s", "invalid", "cf dev download|start|status|stop|destroy")
				pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "invalid"})
			})
		})

		Context("download", func() {
			It("should cleanup old OVAs and download the new OVA", func() {
				gomock.InOrder(
					mockClient.EXPECT().IsEULAAccepted().Return(true, nil),
					mockUI.EXPECT().Say("Downloading VM..."),
					mockDownloader.EXPECT().Download(filepath.Join(home, ".pcfdev", "some-vm-name.ova")),
					mockUI.EXPECT().Say("\nVM downloaded"),
				)

				pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "download"})
			})

			Context("when downloading the OVA fails", func() {
				It("should print an error", func() {
					gomock.InOrder(
						mockClient.EXPECT().IsEULAAccepted().Return(true, nil),
						mockUI.EXPECT().Say("Downloading VM..."),
						mockDownloader.EXPECT().Download(filepath.Join(home, ".pcfdev", "some-vm-name.ova")).Return(errors.New("some-error")),
						mockUI.EXPECT().Failed("Error: some-error"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "download"})
				})
			})

			Context("when EULA check fails", func() {
				It("should print an error", func() {
					gomock.InOrder(
						mockClient.EXPECT().IsEULAAccepted().Return(false, errors.New("some-error")),
						mockUI.EXPECT().Failed("Error: some-error"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "download"})

				})
			})

			Context("when EULA has not been accepted and user accepts the EULA", func() {
				It("should download the ova", func() {
					gomock.InOrder(
						mockClient.EXPECT().IsEULAAccepted().Return(false, nil),
						mockClient.EXPECT().GetEULA().Return("some-eula", nil),
						mockUI.EXPECT().Say("some-eula"),
						mockUI.EXPECT().Confirm("Accept (yes/no):").Return(true),
						mockClient.EXPECT().AcceptEULA().Return(nil),
						mockUI.EXPECT().Say("Downloading VM..."),
						mockDownloader.EXPECT().Download(filepath.Join(home, ".pcfdev", "some-vm-name.ova")),
						mockUI.EXPECT().Say("\nVM downloaded"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "download"})
				})
			})

			Context("when EULA has not been accepted and user denies the EULA", func() {
				It("should not accept and fail gracefully", func() {
					gomock.InOrder(
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
						mockClient.EXPECT().IsEULAAccepted().Return(true, nil),
						mockUI.EXPECT().Say("Downloading VM..."),
						mockDownloader.EXPECT().Download(filepath.Join("some", "other", "dir", ".pcfdev", "some-vm-name.ova")),
						mockUI.EXPECT().Say("\nVM downloaded"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "download"})
				})
			})
		})

		Describe("start", func() {
			Context("when the VM has not been created", func() {
				It("should download and start the ova", func() {
					gomock.InOrder(
						mockRequirementsChecker.EXPECT().Check().Return(nil),
						mockVBox.EXPECT().Status("some-vm-name").Return(vbox.StatusNotCreated, nil),
						mockVBox.EXPECT().ConflictingVMPresent("some-vm-name").Return(false, nil),
						mockClient.EXPECT().IsEULAAccepted().Return(true, nil),
						mockUI.EXPECT().Say("Downloading VM..."),
						mockDownloader.EXPECT().Download(filepath.Join(home, ".pcfdev", "some-vm-name.ova")),
						mockUI.EXPECT().Say("\nVM downloaded"),

						mockUI.EXPECT().Say("Importing VM..."),
						mockVBox.EXPECT().ImportVM(filepath.Join(home, ".pcfdev", "some-vm-name.ova"), "some-vm-name").Return(nil),
						mockUI.EXPECT().Say("PCF Dev is now imported to Virtualbox"),
						mockUI.EXPECT().Say("Starting VM..."),
						mockVBox.EXPECT().StartVM("some-vm-name").Return(vm, nil),
						mockUI.EXPECT().Say("Provisioning VM..."),
						mockSSH.EXPECT().RunSSHCommand("sudo /var/pcfdev/run some-domain some-ip '$2a$04$EpJtIJ8w6hfCwbKYBkn3t.GCY18Pk6s7yN66y37fSJlLuDuMkdHtS'", "some-port", 2*time.Minute, os.Stdout, os.Stderr),
						mockUI.EXPECT().Say("PCF Dev is now running"),
					)
					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
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

					It("should download and start the ova in PCFDEV_HOME", func() {
						gomock.InOrder(
							mockRequirementsChecker.EXPECT().Check().Return(nil),
							mockVBox.EXPECT().Status("some-vm-name").Return(vbox.StatusNotCreated, nil),
							mockVBox.EXPECT().ConflictingVMPresent("some-vm-name").Return(false, nil),
							mockClient.EXPECT().IsEULAAccepted().Return(true, nil),
							mockUI.EXPECT().Say("Downloading VM..."),
							mockDownloader.EXPECT().Download(filepath.Join("some", "other", "dir", ".pcfdev", "some-vm-name.ova")),
							mockUI.EXPECT().Say("\nVM downloaded"),

							mockUI.EXPECT().Say("Importing VM..."),
							mockVBox.EXPECT().ImportVM(filepath.Join("some", "other", "dir", ".pcfdev", "some-vm-name.ova"), "some-vm-name").Return(nil),
							mockUI.EXPECT().Say("PCF Dev is now imported to Virtualbox"),
							mockUI.EXPECT().Say("Starting VM..."),
							mockVBox.EXPECT().StartVM("some-vm-name").Return(vm, nil),
							mockUI.EXPECT().Say("Provisioning VM..."),
							mockSSH.EXPECT().RunSSHCommand("sudo /var/pcfdev/run some-domain some-ip '$2a$04$EpJtIJ8w6hfCwbKYBkn3t.GCY18Pk6s7yN66y37fSJlLuDuMkdHtS'", "some-port", 2*time.Minute, os.Stdout, os.Stderr),
							mockUI.EXPECT().Say("PCF Dev is now running"),
						)
						pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
					})
				})
			})

			Context("when the VM is stopped", func() {
				It("should start and provision the VM", func() {
					gomock.InOrder(
						mockRequirementsChecker.EXPECT().Check().Return(nil),
						mockVBox.EXPECT().Status("some-vm-name").Return(vbox.StatusStopped, nil),
						mockUI.EXPECT().Say("Starting VM..."),
						mockVBox.EXPECT().StartVM("some-vm-name").Return(vm, nil),
						mockUI.EXPECT().Say("Provisioning VM..."),
						mockSSH.EXPECT().RunSSHCommand("sudo /var/pcfdev/run some-domain some-ip '$2a$04$EpJtIJ8w6hfCwbKYBkn3t.GCY18Pk6s7yN66y37fSJlLuDuMkdHtS'", "some-port", 2*time.Minute, os.Stdout, os.Stderr),
						mockUI.EXPECT().Say("PCF Dev is now running"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
				})
			})

			Context("when the VM is already running", func() {
				It("should print a message and exit", func() {
					gomock.InOrder(
						mockRequirementsChecker.EXPECT().Check().Return(nil),
						mockVBox.EXPECT().Status("some-vm-name").Return(vbox.StatusRunning, nil),
						mockUI.EXPECT().Say("PCF Dev is running"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
				})
			})

			Context("when the OVA fails to download", func() {
				It("should print an error message", func() {
					gomock.InOrder(
						mockRequirementsChecker.EXPECT().Check().Return(nil),
						mockVBox.EXPECT().Status("some-vm-name").Return(vbox.StatusNotCreated, nil),
						mockVBox.EXPECT().ConflictingVMPresent("some-vm-name").Return(false, nil),
						mockClient.EXPECT().IsEULAAccepted().Return(true, nil),
						mockUI.EXPECT().Say("Downloading VM..."),
						mockDownloader.EXPECT().Download(filepath.Join(home, ".pcfdev", "some-vm-name.ova")).Return(errors.New("some-error")),
						mockUI.EXPECT().Failed("Error: some-error"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
				})
			})

			Context("when the VM fails to start", func() {
				It("should print an error message", func() {
					gomock.InOrder(
						mockRequirementsChecker.EXPECT().Check().Return(nil),
						mockVBox.EXPECT().Status("some-vm-name").Return(vbox.StatusStopped, nil),
						mockUI.EXPECT().Say("Starting VM..."),
						mockVBox.EXPECT().StartVM("some-vm-name").Return(nil, errors.New("some-error")),
						mockUI.EXPECT().Failed("Error: failed to start VM: some-error"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
				})
			})

			Context("when the VM fails to provision", func() {
				It("should print an error message", func() {
					err := errors.New("some-error")
					gomock.InOrder(
						mockRequirementsChecker.EXPECT().Check().Return(nil),
						mockVBox.EXPECT().Status("some-vm-name").Return(vbox.StatusStopped, nil),
						mockUI.EXPECT().Say("Starting VM..."),
						mockVBox.EXPECT().StartVM("some-vm-name").Return(vm, nil),
						mockUI.EXPECT().Say("Provisioning VM..."),
						mockSSH.EXPECT().RunSSHCommand("sudo /var/pcfdev/run some-domain some-ip '$2a$04$EpJtIJ8w6hfCwbKYBkn3t.GCY18Pk6s7yN66y37fSJlLuDuMkdHtS'", "some-port", 2*time.Minute, os.Stdout, os.Stderr).Return(err),
						mockUI.EXPECT().Failed("Error: failed to provision VM: some-error"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
				})
			})

			Context("when the VM status query fails", func() {
				It("should print an error message", func() {
					expectedError := errors.New("some-error")
					gomock.InOrder(
						mockRequirementsChecker.EXPECT().Check().Return(nil),
						mockVBox.EXPECT().Status("some-vm-name").Return("", expectedError),
						mockUI.EXPECT().Failed("Error: failed to get VM status: some-error"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
				})
			})

			Context("when VM importing fails", func() {
				It("should print an error message", func() {
					expectedError := errors.New("some-error")
					gomock.InOrder(
						mockRequirementsChecker.EXPECT().Check().Return(nil),
						mockVBox.EXPECT().Status("some-vm-name").Return(vbox.StatusNotCreated, nil),
						mockVBox.EXPECT().ConflictingVMPresent("some-vm-name").Return(false, nil),
						mockClient.EXPECT().IsEULAAccepted().Return(true, nil),
						mockUI.EXPECT().Say("Downloading VM..."),
						mockDownloader.EXPECT().Download(filepath.Join(home, ".pcfdev", "some-vm-name.ova")),
						mockUI.EXPECT().Say("\nVM downloaded"),
						mockUI.EXPECT().Say("Importing VM..."),

						mockVBox.EXPECT().ImportVM(filepath.Join(home, ".pcfdev", "some-vm-name.ova"), "some-vm-name").Return(expectedError),
						mockUI.EXPECT().Failed("Error: failed to import VM: some-error"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
				})
			})

			Context("when the VM doesn't exist, but a conflicting VM is present", func() {
				It("should print an error message", func() {
					gomock.InOrder(
						mockRequirementsChecker.EXPECT().Check().Return(nil),
						mockVBox.EXPECT().Status("some-vm-name").Return(vbox.StatusNotCreated, nil),
						mockVBox.EXPECT().ConflictingVMPresent("some-vm-name").Return(true, nil),
						mockUI.EXPECT().Failed("Error: old version of PCF Dev detected, you must run `cf dev destroy` to continue."),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
				})
			})

			Context("when checking for conflicting VMs fails", func() {
				It("should print an error message", func() {
					gomock.InOrder(
						mockRequirementsChecker.EXPECT().Check().Return(nil),
						mockVBox.EXPECT().Status("some-vm-name").Return(vbox.StatusNotCreated, nil),
						mockVBox.EXPECT().ConflictingVMPresent("some-vm-name").Return(false, errors.New("some-error")),
						mockUI.EXPECT().Failed("Error: some-error"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
				})
			})

			Context("when the system does not meet requirements", func() {
				It("should print an error message", func() {
					gomock.InOrder(
						mockRequirementsChecker.EXPECT().Check().Return(errors.New("some-message")),
						mockUI.EXPECT().Failed("Error: could not start PCF Dev: some-message"),
					)
					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
				})
			})

			Context("when the EULA is not accepted", func() {
				It("should print the EULA", func() {
					gomock.InOrder(
						mockRequirementsChecker.EXPECT().Check().Return(nil),
						mockVBox.EXPECT().Status("some-vm-name").Return(vbox.StatusNotCreated, nil),
						mockVBox.EXPECT().ConflictingVMPresent("some-vm-name").Return(false, nil),
						mockClient.EXPECT().IsEULAAccepted().Return(false, nil),
						mockClient.EXPECT().GetEULA().Return("some-eula", nil),
						mockUI.EXPECT().Say("some-eula"),
						mockUI.EXPECT().Confirm("Accept (yes/no):").Return(true),
						mockClient.EXPECT().AcceptEULA().Return(nil),

						mockUI.EXPECT().Say("Downloading VM..."),
						mockDownloader.EXPECT().Download(filepath.Join(home, ".pcfdev", "some-vm-name.ova")),
						mockUI.EXPECT().Say("\nVM downloaded"),

						mockUI.EXPECT().Say("Importing VM..."),
						mockVBox.EXPECT().ImportVM(filepath.Join(home, ".pcfdev", "some-vm-name.ova"), "some-vm-name").Return(nil),
						mockUI.EXPECT().Say("PCF Dev is now imported to Virtualbox"),
						mockUI.EXPECT().Say("Starting VM..."),
						mockVBox.EXPECT().StartVM("some-vm-name").Return(vm, nil),
						mockUI.EXPECT().Say("Provisioning VM..."),
						mockSSH.EXPECT().RunSSHCommand("sudo /var/pcfdev/run some-domain some-ip '$2a$04$EpJtIJ8w6hfCwbKYBkn3t.GCY18Pk6s7yN66y37fSJlLuDuMkdHtS'", "some-port", 2*time.Minute, os.Stdout, os.Stderr),
						mockUI.EXPECT().Say("PCF Dev is now running"),
					)
					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
				})
			})
		})

		Context("stop", func() {
			It("should stop the vm", func() {
				gomock.InOrder(
					mockVBox.EXPECT().Status("some-vm-name").Return(vbox.StatusStopped, nil),
					mockUI.EXPECT().Say("PCF Dev is stopped"),
				)

				pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "stop"})
			})

			Context("when VM is running", func() {
				It("should stop the vm", func() {
					gomock.InOrder(
						mockVBox.EXPECT().Status("some-vm-name").Return(vbox.StatusRunning, nil),
						mockUI.EXPECT().Say("Stopping VM..."),
						mockVBox.EXPECT().StopVM("some-vm-name").Return(nil),
						mockUI.EXPECT().Say("PCF Dev is now stopped"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "stop"})
				})

				Context("Vbox fails to stop VM", func() {
					It("should print an error", func() {
						err := errors.New("some-error")
						gomock.InOrder(
							mockVBox.EXPECT().Status("some-vm-name").Return(vbox.StatusRunning, nil),
							mockUI.EXPECT().Say("Stopping VM..."),
							mockVBox.EXPECT().StopVM("some-vm-name").Return(err),
							mockUI.EXPECT().Failed("Error: failed to stop VM: some-error"),
						)

						pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "stop"})
					})
				})
			})

			Context("when VM is not created", func() {
				Context("when a conflicting VM is running", func() {
					It("should print an error", func() {
						gomock.InOrder(
							mockVBox.EXPECT().Status("some-vm-name").Return(vbox.StatusNotCreated, nil),
							mockVBox.EXPECT().ConflictingVMPresent("some-vm-name").Return(true, nil),
							mockUI.EXPECT().Failed("Error: Old version of PCF Dev detected. You must run `cf dev destroy` to continue."),
						)

						pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "stop"})
					})
				})

				Context("when no conflicting VMs are running", func() {
					It("should send an error message", func() {
						gomock.InOrder(
							mockVBox.EXPECT().Status("some-vm-name").Return(vbox.StatusNotCreated, nil),
							mockVBox.EXPECT().ConflictingVMPresent("some-vm-name").Return(false, nil),
							mockUI.EXPECT().Say("PCF Dev VM has not been created"),
						)

						pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "stop"})
					})
				})

				Context("when checking for conflicting VMs fails", func() {
					It("should print an error", func() {
						gomock.InOrder(
							mockVBox.EXPECT().Status("some-vm-name").Return(vbox.StatusNotCreated, nil),
							mockVBox.EXPECT().ConflictingVMPresent("some-vm-name").Return(false, errors.New("some-error")),
							mockUI.EXPECT().Failed("Error: some-error"),
						)

						pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "stop"})
					})
				})
			})
		})

		Context("status", func() {
			Context("VBox VM is running", func() {
				It("should return the status Running", func() {
					gomock.InOrder(
						mockVBox.EXPECT().Status("some-vm-name").Return(vbox.StatusRunning, nil),
						mockUI.EXPECT().Say("Running"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "status"})
				})
			})

			Context("VBox VM is stopped", func() {
				It("should return the status Stopped", func() {
					gomock.InOrder(
						mockVBox.EXPECT().Status("some-vm-name").Return(vbox.StatusStopped, nil),
						mockUI.EXPECT().Say("Stopped"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "status"})
				})
			})

			Context("VBox VM is not created (i.e. not imported to VBox)", func() {
				It("should return the status Not created", func() {
					gomock.InOrder(
						mockVBox.EXPECT().Status("some-vm-name").Return(vbox.StatusNotCreated, nil),
						mockUI.EXPECT().Say("Not created"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "status"})
				})
			})
		})

		Context("destroy", func() {
			It("should destroy all PCF Dev VMs created by the CLI", func() {
				vms := []string{"pcfdev-0.0.0", "pcfdev-0.0.1"}
				gomock.InOrder(
					mockVBox.EXPECT().GetPCFDevVMs().Return(vms, nil),
					mockUI.EXPECT().Say("Destroying VM..."),
					mockVBox.EXPECT().DestroyVMs(vms).Return(nil),
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
						mockUI.EXPECT().Failed("Error: failed to query VM: some-error"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "destroy"})
				})
			})

			Context("there is an error destroying the VMs", func() {
				It("should send an error message", func() {
					vms := []string{"pcfdev-0.0.0", "pcfdev-0.0.1"}
					gomock.InOrder(
						mockVBox.EXPECT().GetPCFDevVMs().Return(vms, nil),
						mockUI.EXPECT().Say("Destroying VM..."),
						mockVBox.EXPECT().DestroyVMs(vms).Return(errors.New("some-error")),
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
})

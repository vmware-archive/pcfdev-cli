package plugin_test

import (
	"errors"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/pivotal-cf/pcfdev-cli/pivnet"
	"github.com/pivotal-cf/pcfdev-cli/plugin"
	"github.com/pivotal-cf/pcfdev-cli/plugin/mocks"
	"github.com/pivotal-cf/pcfdev-cli/vbox"

	"github.com/cloudfoundry/cli/plugin/fakes"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
)

var _ = Describe("Plugin", func() {
	var (
		mockCtrl                *gomock.Controller
		mockClient              *mocks.MockClient
		mockSSH                 *mocks.MockSSH
		mockUI                  *mocks.MockUI
		mockVBox                *mocks.MockVBox
		mockFS                  *mocks.MockFS
		mockConfig              *mocks.MockConfig
		mockRequirementsChecker *mocks.MockRequirementsChecker
		pcfdev                  *plugin.Plugin
		vm                      *vbox.VM
	)

	const (
		vmName      = "pcfdev-2016-03-29_1728"
		expectedMD5 = "d31706e2dea302d461a1a695a4558b2a"
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = mocks.NewMockClient(mockCtrl)
		mockSSH = mocks.NewMockSSH(mockCtrl)
		mockUI = mocks.NewMockUI(mockCtrl)
		mockVBox = mocks.NewMockVBox(mockCtrl)
		mockFS = mocks.NewMockFS(mockCtrl)
		mockConfig = mocks.NewMockConfig(mockCtrl)
		mockRequirementsChecker = mocks.NewMockRequirementsChecker(mockCtrl)
		pcfdev = &plugin.Plugin{
			PivnetClient:        mockClient,
			SSH:                 mockSSH,
			UI:                  mockUI,
			VBox:                mockVBox,
			FS:                  mockFS,
			Config:              mockConfig,
			RequirementsChecker: mockRequirementsChecker,
			VMName:              vmName,
			ExpectedMD5:         expectedMD5,
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
			home = os.Getenv("HOME")
			os.Setenv("HOME", "/some/dir")
		})

		AfterEach(func() {
			os.Setenv("HOME", home)
		})

		Context("when it is called with the wrong number of arguments", func() {
			It("should print the usage message", func() {
				mockUI.EXPECT().Failed("Usage: %s", "cf dev download|start|status|stop|destroy")
				pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev"})
			})
		})

		Context("download", func() {
			Context("when ova does not exist", func() {
				It("should download the OVA", func() {
					readCloser := &pivnet.DownloadReader{ReadCloser: ioutil.NopCloser(strings.NewReader("some-ova-contents"))}
					gomock.InOrder(
						mockFS.EXPECT().CreateDir("/some/dir/.pcfdev").Return(nil),
						mockFS.EXPECT().Exists("/some/dir/.pcfdev/pcfdev.ova").Return(false, nil),
						mockConfig.EXPECT().GetToken().Return("some-token"),
						mockUI.EXPECT().Say("Downloading VM..."),
						mockClient.EXPECT().DownloadOVA("some-token").Return(readCloser, nil),
						mockFS.EXPECT().Write("/some/dir/.pcfdev/pcfdev.ova", readCloser).Return(nil),
						mockUI.EXPECT().Say("\nFinished downloading VM"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "download"})
				})
			})

			Context("when PCFDEV_HOME is set", func() {
				var pcfdevHome string

				BeforeEach(func() {
					pcfdevHome = os.Getenv("PCFDEV_HOME")
					os.Setenv("PCFDEV_HOME", "/some/other/dir")
				})

				AfterEach(func() {
					os.Setenv("PCFDEV_HOME", pcfdevHome)
				})

				It("should download the ova to PCFDEV_HOME", func() {
					readCloser := &pivnet.DownloadReader{ReadCloser: ioutil.NopCloser(strings.NewReader("some-ova-contents"))}
					gomock.InOrder(
						mockFS.EXPECT().CreateDir("/some/other/dir").Return(nil),
						mockFS.EXPECT().Exists("/some/other/dir/pcfdev.ova").Return(false, nil),
						mockConfig.EXPECT().GetToken().Return("some-token"),
						mockUI.EXPECT().Say("Downloading VM..."),
						mockClient.EXPECT().DownloadOVA("some-token").Return(readCloser, nil),
						mockFS.EXPECT().Write("/some/other/dir/pcfdev.ova", readCloser).Return(nil),
						mockUI.EXPECT().Say("\nFinished downloading VM"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "download"})
				})
			})

			Context("when ova exists and is up to date", func() {
				It("should not download the OVA", func() {
					gomock.InOrder(
						mockFS.EXPECT().CreateDir("/some/dir/.pcfdev").Return(nil),
						mockFS.EXPECT().Exists("/some/dir/.pcfdev/pcfdev.ova").Return(true, nil),
						mockFS.EXPECT().MD5("/some/dir/.pcfdev/pcfdev.ova").Return(expectedMD5, nil),
						mockUI.EXPECT().Say("VM already downloaded"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "download"})
				})
			})

			Context("when ova exists and is old", func() {
				It("should download the OVA", func() {
					readCloser := &pivnet.DownloadReader{ReadCloser: ioutil.NopCloser(strings.NewReader("some-ova-contents"))}
					gomock.InOrder(
						mockFS.EXPECT().CreateDir("/some/dir/.pcfdev").Return(nil),
						mockFS.EXPECT().Exists("/some/dir/.pcfdev/pcfdev.ova").Return(true, nil),
						mockFS.EXPECT().MD5("/some/dir/.pcfdev/pcfdev.ova").Return("some-old-shasum", nil),
						mockVBox.EXPECT().Status(vmName).Return(vbox.StatusNotCreated, nil),
						mockUI.EXPECT().Say("Upgrading your locally stored version of PCF Dev..."),
						mockFS.EXPECT().RemoveFile("/some/dir/.pcfdev/pcfdev.ova").Return(nil),
						mockConfig.EXPECT().GetToken().Return("some-token"),
						mockUI.EXPECT().Say("Downloading VM..."),
						mockClient.EXPECT().DownloadOVA("some-token").Return(readCloser, nil),
						mockFS.EXPECT().Write("/some/dir/.pcfdev/pcfdev.ova", readCloser).Return(nil),
						mockUI.EXPECT().Say("\nFinished downloading VM"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "download"})
				})

				Context("when removing old ova fails", func() {
					It("should return an error", func() {
						gomock.InOrder(
							mockFS.EXPECT().CreateDir("/some/dir/.pcfdev").Return(nil),
							mockFS.EXPECT().Exists("/some/dir/.pcfdev/pcfdev.ova").Return(true, nil),
							mockFS.EXPECT().MD5("/some/dir/.pcfdev/pcfdev.ova").Return("some-bad-md5", nil),
							mockVBox.EXPECT().Status(vmName).Return(vbox.StatusNotCreated, nil),
							mockUI.EXPECT().Say("Upgrading your locally stored version of PCF Dev..."),
							mockFS.EXPECT().RemoveFile("/some/dir/.pcfdev/pcfdev.ova").Return(errors.New("some-error")),
							mockUI.EXPECT().Failed("failed to remove old machine image /some/dir/.pcfdev/pcfdev.ova"),
						)

						pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "download"})
					})
				})

				Context("when querying for vm status fails", func() {
					It("should return an error", func() {
						gomock.InOrder(
							mockFS.EXPECT().CreateDir("/some/dir/.pcfdev").Return(nil),
							mockFS.EXPECT().Exists("/some/dir/.pcfdev/pcfdev.ova").Return(true, nil),
							mockFS.EXPECT().MD5("/some/dir/.pcfdev/pcfdev.ova").Return("some-bad-md5", nil),
							mockVBox.EXPECT().Status(vmName).Return("", errors.New("some-error")),
							mockUI.EXPECT().Failed("failed to get VM status: some-error"),
						)

						pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "download"})
					})
				})
			})

			Context("when ova shasum fails to compute", func() {
				It("should return an error", func() {
					gomock.InOrder(
						mockFS.EXPECT().CreateDir("/some/dir/.pcfdev").Return(nil),
						mockFS.EXPECT().Exists("/some/dir/.pcfdev/pcfdev.ova").Return(true, nil),
						mockFS.EXPECT().MD5("/some/dir/.pcfdev/pcfdev.ova").Return("", errors.New("some-error")),
						mockUI.EXPECT().Failed("failed to compute checksum of /some/dir/.pcfdev/pcfdev.ova"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "download"})
				})
			})
		})

		Context("start", func() {
			Context("VM has not been created and pcfdev.ova has not been downloaded", func() {
				It("should start without downloading the ova", func() {
					readCloser := &pivnet.DownloadReader{ReadCloser: ioutil.NopCloser(strings.NewReader("some-ova-contents"))}
					gomock.InOrder(
						mockRequirementsChecker.EXPECT().Check().Return(nil),
						mockFS.EXPECT().CreateDir("/some/dir/.pcfdev").Return(nil),
						mockFS.EXPECT().Exists("/some/dir/.pcfdev/pcfdev.ova").Return(false, nil),
						mockConfig.EXPECT().GetToken().Return("some-token"),
						mockUI.EXPECT().Say("Downloading VM..."),
						mockClient.EXPECT().DownloadOVA("some-token").Return(readCloser, nil),
						mockFS.EXPECT().Write("/some/dir/.pcfdev/pcfdev.ova", readCloser).Return(nil),
						mockUI.EXPECT().Say("\nFinished downloading VM"),
						mockVBox.EXPECT().Status(vmName).Return(vbox.StatusNotCreated, nil),
						mockUI.EXPECT().Say("Importing VM..."),
						mockVBox.EXPECT().ImportVM("/some/dir/.pcfdev/pcfdev.ova", vmName).Return(nil),
						mockUI.EXPECT().Say("PCF Dev is now imported to Virtualbox"),
						mockUI.EXPECT().Say("Starting VM..."),
						mockVBox.EXPECT().StartVM(vmName).Return(vm, nil),
						mockUI.EXPECT().Say("Provisioning VM..."),
						mockSSH.EXPECT().RunSSHCommand("sudo /var/pcfdev/run some-domain some-ip '$2a$04$EpJtIJ8w6hfCwbKYBkn3t.GCY18Pk6s7yN66y37fSJlLuDuMkdHtS'", "some-port", 2*time.Minute, os.Stdout, os.Stderr),
						mockUI.EXPECT().Say("PCF Dev is now running"),
					)
					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
				})
			})

			Context("VM has not been created and pcfdev.ova has been downloaded", func() {
				It("should start without downloading the ova", func() {
					gomock.InOrder(
						mockRequirementsChecker.EXPECT().Check().Return(nil),
						mockFS.EXPECT().CreateDir("/some/dir/.pcfdev").Return(nil),
						mockFS.EXPECT().Exists("/some/dir/.pcfdev/pcfdev.ova").Return(true, nil),
						mockFS.EXPECT().MD5("/some/dir/.pcfdev/pcfdev.ova").Return(expectedMD5, nil),
						mockUI.EXPECT().Say("VM already downloaded"),
						mockVBox.EXPECT().Status(vmName).Return(vbox.StatusNotCreated, nil),
						mockUI.EXPECT().Say("Importing VM..."),
						mockVBox.EXPECT().ImportVM("/some/dir/.pcfdev/pcfdev.ova", vmName).Return(nil),
						mockUI.EXPECT().Say("PCF Dev is now imported to Virtualbox"),
						mockUI.EXPECT().Say("Starting VM..."),
						mockVBox.EXPECT().StartVM(vmName).Return(vm, nil),
						mockUI.EXPECT().Say("Provisioning VM..."),
						mockSSH.EXPECT().RunSSHCommand("sudo /var/pcfdev/run some-domain some-ip '$2a$04$EpJtIJ8w6hfCwbKYBkn3t.GCY18Pk6s7yN66y37fSJlLuDuMkdHtS'", "some-port", 2*time.Minute, os.Stdout, os.Stderr),
						mockUI.EXPECT().Say("PCF Dev is now running"),
					)
					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
				})
			})

			Context("VM is stopped", func() {
				It("should start and provision the VM", func() {
					readCloser := &pivnet.DownloadReader{ReadCloser: ioutil.NopCloser(strings.NewReader("some-ova-contents"))}
					gomock.InOrder(
						mockRequirementsChecker.EXPECT().Check().Return(nil),
						mockFS.EXPECT().CreateDir("/some/dir/.pcfdev").Return(nil),
						mockFS.EXPECT().Exists("/some/dir/.pcfdev/pcfdev.ova").Return(false, nil),
						mockConfig.EXPECT().GetToken().Return("some-token"),
						mockUI.EXPECT().Say("Downloading VM..."),
						mockClient.EXPECT().DownloadOVA("some-token").Return(readCloser, nil),
						mockFS.EXPECT().Write("/some/dir/.pcfdev/pcfdev.ova", readCloser).Return(nil),
						mockUI.EXPECT().Say("\nFinished downloading VM"),

						mockVBox.EXPECT().Status(vmName).Return(vbox.StatusStopped, nil),
						mockUI.EXPECT().Say("Starting VM..."),
						mockVBox.EXPECT().StartVM(vmName).Return(vm, nil),
						mockUI.EXPECT().Say("Provisioning VM..."),
						mockSSH.EXPECT().RunSSHCommand("sudo /var/pcfdev/run some-domain some-ip '$2a$04$EpJtIJ8w6hfCwbKYBkn3t.GCY18Pk6s7yN66y37fSJlLuDuMkdHtS'", "some-port", 2*time.Minute, os.Stdout, os.Stderr),
						mockUI.EXPECT().Say("PCF Dev is now running"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
				})
			})

			Context("VM is already running", func() {
				It("prints a message and exits", func() {
					gomock.InOrder(
						mockRequirementsChecker.EXPECT().Check().Return(nil),
						mockFS.EXPECT().CreateDir("/some/dir/.pcfdev").Return(nil),
						mockFS.EXPECT().Exists("/some/dir/.pcfdev/pcfdev.ova").Return(true, nil),
						mockFS.EXPECT().MD5("/some/dir/.pcfdev/pcfdev.ova").Return(expectedMD5, nil),
						mockUI.EXPECT().Say("VM already downloaded"),
						mockVBox.EXPECT().Status(vmName).Return(vbox.StatusRunning, nil),
						mockUI.EXPECT().Say("PCF Dev is running"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
				})
			})

			Context("fails to create .pcfdev dir", func() {
				It("prints an error", func() {
					err := errors.New("some-error")
					gomock.InOrder(
						mockRequirementsChecker.EXPECT().Check().Return(nil),
						mockFS.EXPECT().CreateDir("/some/dir/.pcfdev").Return(err),
						mockUI.EXPECT().Failed("some-error"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
				})
			})

			Context("fails to check if pcfdev.ova exists", func() {
				It("prints an error", func() {
					err := errors.New("some-error")

					gomock.InOrder(
						mockRequirementsChecker.EXPECT().Check().Return(nil),
						mockFS.EXPECT().CreateDir("/some/dir/.pcfdev").Return(nil),
						mockFS.EXPECT().Exists("/some/dir/.pcfdev/pcfdev.ova").Return(false, err),
						mockUI.EXPECT().Failed("some-error"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
				})
			})

			Context("OVA fails to download", func() {
				It("prints an error", func() {
					err := errors.New("some-error")
					gomock.InOrder(
						mockRequirementsChecker.EXPECT().Check().Return(nil),
						mockFS.EXPECT().CreateDir("/some/dir/.pcfdev").Return(nil),
						mockFS.EXPECT().Exists("/some/dir/.pcfdev/pcfdev.ova").Return(false, nil),
						mockConfig.EXPECT().GetToken().Return("some-token"),
						mockUI.EXPECT().Say("Downloading VM..."),
						mockClient.EXPECT().DownloadOVA("some-token").Return(nil, err),
						mockUI.EXPECT().Failed("Error: some-error"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
				})
			})

			Context("VM fails to start", func() {
				It("prints an error", func() {
					gomock.InOrder(
						mockRequirementsChecker.EXPECT().Check().Return(nil),
						mockFS.EXPECT().CreateDir("/some/dir/.pcfdev").Return(nil),
						mockFS.EXPECT().Exists("/some/dir/.pcfdev/pcfdev.ova").Return(true, nil),
						mockFS.EXPECT().MD5("/some/dir/.pcfdev/pcfdev.ova").Return(expectedMD5, nil),
						mockUI.EXPECT().Say("VM already downloaded"),
						mockVBox.EXPECT().Status(vmName).Return(vbox.StatusStopped, nil),
						mockUI.EXPECT().Say("Starting VM..."),
						mockVBox.EXPECT().StartVM(vmName).Return(nil, errors.New("some-error")),
						mockUI.EXPECT().Failed("failed to start VM: some-error"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
				})
			})

			Context("when the VM fails to provision", func() {
				It("should print an error", func() {
					err := errors.New("some-error")
					gomock.InOrder(
						mockRequirementsChecker.EXPECT().Check().Return(nil),
						mockFS.EXPECT().CreateDir("/some/dir/.pcfdev").Return(nil),
						mockFS.EXPECT().Exists("/some/dir/.pcfdev/pcfdev.ova").Return(true, nil),
						mockFS.EXPECT().MD5("/some/dir/.pcfdev/pcfdev.ova").Return(expectedMD5, nil),
						mockUI.EXPECT().Say("VM already downloaded"),
						mockVBox.EXPECT().Status(vmName).Return(vbox.StatusStopped, nil),
						mockUI.EXPECT().Say("Starting VM..."),
						mockVBox.EXPECT().StartVM(vmName).Return(vm, nil),
						mockUI.EXPECT().Say("Provisioning VM..."),
						mockSSH.EXPECT().RunSSHCommand("sudo /var/pcfdev/run some-domain some-ip '$2a$04$EpJtIJ8w6hfCwbKYBkn3t.GCY18Pk6s7yN66y37fSJlLuDuMkdHtS'", "some-port", 2*time.Minute, os.Stdout, os.Stderr).Return(err),
						mockUI.EXPECT().Failed("failed to provision VM: some-error"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
				})
			})

			Context("VM status query fails", func() {
				It("prints an error", func() {
					expectedError := errors.New("some-error")
					gomock.InOrder(
						mockRequirementsChecker.EXPECT().Check().Return(nil),
						mockFS.EXPECT().CreateDir("/some/dir/.pcfdev").Return(nil),
						mockFS.EXPECT().Exists("/some/dir/.pcfdev/pcfdev.ova").Return(true, nil),
						mockFS.EXPECT().MD5("/some/dir/.pcfdev/pcfdev.ova").Return(expectedMD5, nil),
						mockUI.EXPECT().Say("VM already downloaded"),
						mockVBox.EXPECT().Status(vmName).Return("", expectedError),
						mockUI.EXPECT().Failed("failed to get VM status: some-error"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
				})
			})

			Context("VM importing fails", func() {
				It("prints an error", func() {
					expectedError := errors.New("some-error")
					gomock.InOrder(
						mockRequirementsChecker.EXPECT().Check().Return(nil),
						mockFS.EXPECT().CreateDir("/some/dir/.pcfdev").Return(nil),
						mockFS.EXPECT().Exists("/some/dir/.pcfdev/pcfdev.ova").Return(true, nil),
						mockFS.EXPECT().MD5("/some/dir/.pcfdev/pcfdev.ova").Return(expectedMD5, nil),
						mockUI.EXPECT().Say("VM already downloaded"),
						mockVBox.EXPECT().Status(vmName).Return(vbox.StatusNotCreated, nil),
						mockUI.EXPECT().Say("Importing VM..."),

						mockVBox.EXPECT().ImportVM("/some/dir/.pcfdev/pcfdev.ova", vmName).Return(expectedError),
						mockUI.EXPECT().Failed("failed to import VM: some-error"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
				})
			})

			Context("when the vm exists and is old and is stopped", func() {
				It("prints an error", func() {
					gomock.InOrder(
						mockRequirementsChecker.EXPECT().Check().Return(nil),
						mockFS.EXPECT().CreateDir("/some/dir/.pcfdev").Return(nil),
						mockFS.EXPECT().Exists("/some/dir/.pcfdev/pcfdev.ova").Return(true, nil),
						mockFS.EXPECT().MD5("/some/dir/.pcfdev/pcfdev.ova").Return("some-bad-md5", nil),
						mockVBox.EXPECT().Status(vmName).Return(vbox.StatusStopped, nil),
						mockUI.EXPECT().Failed("Old version of PCF Dev detected. You must run `cf dev destroy` to continue."),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
				})
			})

			Context("when the vm exists and is old and there is an error querying for vm status", func() {
				It("should return an error", func() {
					gomock.InOrder(
						mockRequirementsChecker.EXPECT().Check().Return(nil),
						mockFS.EXPECT().CreateDir("/some/dir/.pcfdev").Return(nil),
						mockFS.EXPECT().Exists("/some/dir/.pcfdev/pcfdev.ova").Return(true, nil),
						mockFS.EXPECT().MD5("/some/dir/.pcfdev/pcfdev.ova").Return("some-bad-md5", nil),
						mockVBox.EXPECT().Status(vmName).Return("", errors.New("some-error")),
						mockUI.EXPECT().Failed("failed to get VM status: some-error"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
				})
			})

			Context("when the system does not meet requirements", func() {
				It("should print an error message", func() {
					gomock.InOrder(
						mockRequirementsChecker.EXPECT().Check().Return(errors.New("some-message")),
						mockUI.EXPECT().Failed("Could not start PCF Dev: some-message"),
					)
					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
				})
			})
		})

		Context("stop", func() {
			It("should stop the vm", func() {
				gomock.InOrder(
					mockVBox.EXPECT().Status(vmName).Return(vbox.StatusStopped, nil),
					mockUI.EXPECT().Say("PCF Dev is stopped"),
				)

				pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "stop"})
			})

			Context("when VM is running", func() {
				It("should stop the vm", func() {
					gomock.InOrder(
						mockVBox.EXPECT().Status(vmName).Return(vbox.StatusRunning, nil),
						mockUI.EXPECT().Say("Stopping VM..."),
						mockVBox.EXPECT().StopVM(vmName).Return(nil),
						mockUI.EXPECT().Say("PCF Dev is now stopped"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "stop"})
				})

				Context("Vbox fails to stop VM", func() {
					It("should print an error", func() {
						err := errors.New("some-error")
						gomock.InOrder(
							mockVBox.EXPECT().Status(vmName).Return(vbox.StatusRunning, nil),
							mockUI.EXPECT().Say("Stopping VM..."),
							mockVBox.EXPECT().StopVM(vmName).Return(err),
							mockUI.EXPECT().Failed("failed to stop VM: some-error"),
						)

						pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "stop"})
					})
				})
			})

			Context("when VM is not created", func() {
				Context("when a conflicting VM is running", func() {
					It("should print an error", func() {
						gomock.InOrder(
							mockVBox.EXPECT().Status(vmName).Return(vbox.StatusNotCreated, nil),
							mockVBox.EXPECT().ConflictingVMPresent(vmName).Return(true, nil),
							mockUI.EXPECT().Failed("Old version of PCF Dev detected. You must run `cf dev destroy` to continue."),
						)

						pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "stop"})
					})
				})

				Context("when no conflicting VMs are running", func() {
					It("should send an error message", func() {
						gomock.InOrder(
							mockVBox.EXPECT().Status(vmName).Return(vbox.StatusNotCreated, nil),
							mockVBox.EXPECT().ConflictingVMPresent(vmName).Return(false, nil),
							mockUI.EXPECT().Say("PCF Dev VM has not been created"),
						)

						pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "stop"})
					})
				})

				Context("when checking for conflicting VMs fails", func() {
					It("should print an error", func() {
						gomock.InOrder(
							mockVBox.EXPECT().Status(vmName).Return(vbox.StatusNotCreated, nil),
							mockVBox.EXPECT().ConflictingVMPresent(vmName).Return(false, errors.New("some-error")),
							mockUI.EXPECT().Failed("some-error"),
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
						mockVBox.EXPECT().Status(vmName).Return(vbox.StatusRunning, nil),
						mockUI.EXPECT().Say("Running"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "status"})
				})
			})

			Context("VBox VM is stopped", func() {
				It("should return the status Stopped", func() {
					gomock.InOrder(
						mockVBox.EXPECT().Status(vmName).Return(vbox.StatusStopped, nil),
						mockUI.EXPECT().Say("Stopped"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "status"})
				})
			})

			Context("VBox VM is not created (i.e. not imported to VBox)", func() {
				It("should return the status Not created", func() {
					gomock.InOrder(
						mockVBox.EXPECT().Status(vmName).Return(vbox.StatusNotCreated, nil),
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
						mockUI.EXPECT().Failed("failed to query VM: some-error"),
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
						mockUI.EXPECT().Failed("failed to destroy VM: some-error"),
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

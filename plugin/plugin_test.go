package plugin_test

import (
	"errors"
	"io/ioutil"
	"os"
	"strings"

	"github.com/pivotal-cf/pcfdev-cli/plugin"
	"github.com/pivotal-cf/pcfdev-cli/plugin/mocks"
	"github.com/pivotal-cf/pcfdev-cli/vbox"

	"github.com/cloudfoundry/cli/plugin/fakes"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
)

var _ = Describe("Plugin", func() {
	var (
		mockCtrl   *gomock.Controller
		mockClient *mocks.MockClient
		mockSSH    *mocks.MockSSH
		mockUI     *mocks.MockUI
		mockVBox   *mocks.MockVBox
		mockFS     *mocks.MockFS
		pcfdev     *plugin.Plugin
		vm         *vbox.VM
	)

	const vmName = "pcfdev-2016-03-29_1728"

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = mocks.NewMockClient(mockCtrl)
		mockSSH = mocks.NewMockSSH(mockCtrl)
		mockUI = mocks.NewMockUI(mockCtrl)
		mockVBox = mocks.NewMockVBox(mockCtrl)
		mockFS = mocks.NewMockFS(mockCtrl)
		pcfdev = &plugin.Plugin{
			PivnetClient: mockClient,
			SSH:          mockSSH,
			UI:           mockUI,
			VBox:         mockVBox,
			FS:           mockFS,
		}
		vm = &vbox.VM{
			IP:      "some-ip",
			SSHPort: "some-port",
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

		Context("wrong number of arguments", func() {
			It("prints the usage message", func() {
				mockUI.EXPECT().Failed("Usage: %s", "cf dev download|start|status|stop|destroy")
				pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev"})
			})
		})

		Context("download", func() {
			Context("when ova does not exist", func() {
				It("should download the VM", func() {
					readCloser := ioutil.NopCloser(strings.NewReader("some-ova-contents"))
					gomock.InOrder(
						mockFS.EXPECT().CreateDir("/some/dir/.pcfdev").Return(nil),
						mockFS.EXPECT().Exists("/some/dir/.pcfdev/pcfdev.ova").Return(false, nil),
						mockUI.EXPECT().Say("Downloading VM..."),
						mockClient.EXPECT().DownloadOVA().Return(readCloser, nil),
						mockFS.EXPECT().Write("/some/dir/.pcfdev/pcfdev.ova", readCloser).Return(nil),
						mockUI.EXPECT().Say("Finished downloading VM"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "download"})
				})
			})

			Context("when ova already exists", func() {
				It("should not download the VM", func() {
					gomock.InOrder(
						mockFS.EXPECT().CreateDir("/some/dir/.pcfdev").Return(nil),
						mockFS.EXPECT().Exists("/some/dir/.pcfdev/pcfdev.ova").Return(true, nil),
						mockUI.EXPECT().Say("VM already downloaded"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "download"})
				})
			})
		})

		Context("start", func() {
			Context("VM has not been created and pcfdev.ova has not been downloaded", func() {
				It("should start without downloading the ova", func() {
					readCloser := ioutil.NopCloser(strings.NewReader("some-ova-contents"))
					gomock.InOrder(
						mockVBox.EXPECT().Status(vmName).Return(vbox.StatusNotCreated, nil),
						mockFS.EXPECT().CreateDir("/some/dir/.pcfdev").Return(nil),
						mockFS.EXPECT().Exists("/some/dir/.pcfdev/pcfdev.ova").Return(false, nil),
						mockUI.EXPECT().Say("Downloading VM..."),
						mockClient.EXPECT().DownloadOVA().Return(readCloser, nil),
						mockFS.EXPECT().Write("/some/dir/.pcfdev/pcfdev.ova", readCloser).Return(nil),
						mockUI.EXPECT().Say("Finished downloading VM"),
						mockUI.EXPECT().Say("Importing VM..."),
						mockVBox.EXPECT().ImportVM("/some/dir/.pcfdev/pcfdev.ova", vmName).Return(nil),
						mockUI.EXPECT().Say("PCF Dev is now imported to Virtualbox"),
						mockUI.EXPECT().Say("Starting VM..."),
						mockVBox.EXPECT().StartVM(vmName).Return(vm, nil),
						mockUI.EXPECT().Say("Provisioning VM..."),
						mockSSH.EXPECT().RunSSHCommand("sudo /var/pcfdev/run local.pcfdev.io some-ip", "some-port"),
						mockUI.EXPECT().Say("PCF Dev is now running"),
					)
					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
				})
			})

			Context("VM has not been created and pcfdev.ova has been downloaded", func() {
				It("should start without downloading the ova", func() {
					gomock.InOrder(
						mockVBox.EXPECT().Status(vmName).Return(vbox.StatusNotCreated, nil),
						mockFS.EXPECT().CreateDir("/some/dir/.pcfdev").Return(nil),
						mockFS.EXPECT().Exists("/some/dir/.pcfdev/pcfdev.ova").Return(true, nil),
						mockUI.EXPECT().Say("VM already downloaded"),
						mockUI.EXPECT().Say("Importing VM..."),
						mockVBox.EXPECT().ImportVM("/some/dir/.pcfdev/pcfdev.ova", vmName).Return(nil),
						mockUI.EXPECT().Say("PCF Dev is now imported to Virtualbox"),
						mockUI.EXPECT().Say("Starting VM..."),
						mockVBox.EXPECT().StartVM(vmName).Return(vm, nil),
						mockUI.EXPECT().Say("Provisioning VM..."),
						mockSSH.EXPECT().RunSSHCommand("sudo /var/pcfdev/run local.pcfdev.io some-ip", "some-port"),
						mockUI.EXPECT().Say("PCF Dev is now running"),
					)
					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
				})
			})

			Context("VM is stopped", func() {
				It("should start and provision the VM", func() {
					readCloser := ioutil.NopCloser(strings.NewReader("some-ova-contents"))
					gomock.InOrder(
						mockVBox.EXPECT().Status(vmName).Return(vbox.StatusStopped, nil),
						mockFS.EXPECT().CreateDir("/some/dir/.pcfdev").Return(nil),
						mockFS.EXPECT().Exists("/some/dir/.pcfdev/pcfdev.ova").Return(false, nil),
						mockUI.EXPECT().Say("Downloading VM..."),
						mockClient.EXPECT().DownloadOVA().Return(readCloser, nil),
						mockFS.EXPECT().Write("/some/dir/.pcfdev/pcfdev.ova", readCloser).Return(nil),
						mockUI.EXPECT().Say("Finished downloading VM"),

						mockUI.EXPECT().Say("Starting VM..."),
						mockVBox.EXPECT().StartVM(vmName).Return(vm, nil),
						mockUI.EXPECT().Say("Provisioning VM..."),
						mockSSH.EXPECT().RunSSHCommand("sudo /var/pcfdev/run local.pcfdev.io some-ip", "some-port"),
						mockUI.EXPECT().Say("PCF Dev is now running"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
				})
			})

			Context("VM is already running", func() {
				It("prints a message and exits", func() {
					gomock.InOrder(
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
						mockVBox.EXPECT().Status(vmName).Return(vbox.StatusNotCreated, nil),
						mockFS.EXPECT().CreateDir("/some/dir/.pcfdev").Return(err),
						mockUI.EXPECT().Failed("failed to fetch VM: some-error"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
				})
			})

			Context("fails to check if pcfdev.ova exists", func() {
				It("prints an error", func() {
					err := errors.New("some-error")

					gomock.InOrder(
						mockVBox.EXPECT().Status(vmName).Return(vbox.StatusNotCreated, nil),
						mockFS.EXPECT().CreateDir("/some/dir/.pcfdev").Return(nil),
						mockFS.EXPECT().Exists("/some/dir/.pcfdev/pcfdev.ova").Return(false, err),
						mockUI.EXPECT().Failed("failed to fetch VM: some-error"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
				})
			})

			Context("OVA fails to download", func() {
				It("prints an error", func() {
					err := errors.New("some-error")
					gomock.InOrder(
						mockVBox.EXPECT().Status(vmName).Return(vbox.StatusNotCreated, nil),
						mockFS.EXPECT().CreateDir("/some/dir/.pcfdev").Return(nil),
						mockFS.EXPECT().Exists("/some/dir/.pcfdev/pcfdev.ova").Return(false, nil),
						mockUI.EXPECT().Say("Downloading VM..."),
						mockClient.EXPECT().DownloadOVA().Return(nil, err),
						mockUI.EXPECT().Failed("failed to fetch VM: some-error"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
				})
			})

			Context("VM fails to start", func() {
				It("prints an error", func() {
					gomock.InOrder(
						mockVBox.EXPECT().Status(vmName).Return(vbox.StatusStopped, nil),
						mockFS.EXPECT().CreateDir("/some/dir/.pcfdev").Return(nil),
						mockFS.EXPECT().Exists("/some/dir/.pcfdev/pcfdev.ova").Return(true, nil),
						mockUI.EXPECT().Say("VM already downloaded"),
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
						mockVBox.EXPECT().Status(vmName).Return(vbox.StatusStopped, nil),
						mockFS.EXPECT().CreateDir("/some/dir/.pcfdev").Return(nil),
						mockFS.EXPECT().Exists("/some/dir/.pcfdev/pcfdev.ova").Return(true, nil),
						mockUI.EXPECT().Say("VM already downloaded"),
						mockUI.EXPECT().Say("Starting VM..."),
						mockVBox.EXPECT().StartVM(vmName).Return(vm, nil),
						mockUI.EXPECT().Say("Provisioning VM..."),
						mockSSH.EXPECT().RunSSHCommand("sudo /var/pcfdev/run local.pcfdev.io some-ip", "some-port").Return(err),
						mockUI.EXPECT().Failed("failed to provision VM: some-error"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
				})
			})

			Context("VM status query fails", func() {
				It("prints an error", func() {
					expectedError := errors.New("some-error")
					gomock.InOrder(
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
						mockVBox.EXPECT().Status(vmName).Return(vbox.StatusNotCreated, nil),
						mockFS.EXPECT().CreateDir("/some/dir/.pcfdev").Return(nil),
						mockFS.EXPECT().Exists("/some/dir/.pcfdev/pcfdev.ova").Return(true, nil),
						mockUI.EXPECT().Say("VM already downloaded"),
						mockUI.EXPECT().Say("Importing VM..."),

						mockVBox.EXPECT().ImportVM("/some/dir/.pcfdev/pcfdev.ova", vmName).Return(expectedError),
						mockUI.EXPECT().Failed("failed to import VM: some-error"),
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

			Context("VM is running", func() {
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
				It("should return the status Not Created", func() {
					gomock.InOrder(
						mockVBox.EXPECT().Status(vmName).Return(vbox.StatusNotCreated, nil),
						mockUI.EXPECT().Say("Not Created"),
					)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "status"})
				})
			})
		})

		Context("destroy", func() {
			It("should destroy the vm", func() {
				gomock.InOrder(
					mockVBox.EXPECT().Status(vmName).Return(vbox.StatusStopped, nil),
					mockUI.EXPECT().Say("Destroying VM..."),
					mockVBox.EXPECT().DestroyVM(vmName).Return(nil),
					mockUI.EXPECT().Say("PCF Dev VM has been destroyed"),
				)

				pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "destroy"})
			})

			Context("there is no VM", func() {
				It("should send an error message", func() {
					gomock.InOrder(
						mockVBox.EXPECT().Status(vmName).Return(vbox.StatusNotCreated, nil),
						mockUI.EXPECT().Say("PCF Dev VM has not been created"),
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

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
	)

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
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("Run", func() {
		var home string
		Context("wrong number of arguments", func() {
			It("prints the usage message", func() {
				mockUI.EXPECT().Failed("Usage: %s", "cf dev start|stop")
				pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev"})
			})
		})
		Context("start", func() {
			BeforeEach(func() {
				home = os.Getenv("HOME")
				os.Setenv("HOME", "/some/dir")
			})
			AfterEach(func() {
				os.Setenv("HOME", home)
			})
			It("should start and provision the VM", func() {
				readCloser := ioutil.NopCloser(strings.NewReader("some-ova-contents"))
				vm := vbox.VM{
					IP:      "some-ip",
					SSHPort: "some-port",
				}
				mockVBox.EXPECT().IsVMRunning("pcfdev-2016-03-29_1728").Return(false)
				mockFS.EXPECT().CreateDir("/some/dir/.pcfdev").Return(nil)
				mockFS.EXPECT().Exists("/some/dir/.pcfdev/pcfdev.ova").Return(false, nil)
				mockUI.EXPECT().Say("Downloading OVA...")
				mockClient.EXPECT().DownloadOVA().Return(readCloser, nil)
				mockFS.EXPECT().Write("/some/dir/.pcfdev/pcfdev.ova", readCloser).Return(nil)
				mockUI.EXPECT().Say("Finished downloading OVA")

				mockUI.EXPECT().Say("Starting VM...")
				mockVBox.EXPECT().StartVM(gomock.Any(), "pcfdev-2016-03-29_1728").Return(&vm, nil)
				mockUI.EXPECT().Say("Provisioning VM...")
				mockSSH.EXPECT().RunSSHCommand("sudo /var/pcfdev/run local.pcfdev.io some-ip", "some-port")
				mockUI.EXPECT().Say("PCFDev is now running")

				pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
			})

			Context("fails to create .pcfdev dir", func() {
				It("prints an error", func() {
					mockVBox.EXPECT().IsVMRunning("pcfdev-2016-03-29_1728").Return(false)
					err := errors.New("some-error")
					mockFS.EXPECT().CreateDir("/some/dir/.pcfdev").Return(err)
					mockUI.EXPECT().Failed("failed to fetch OVA: %s", err)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
				})
			})

			Context("fails to check if pcfdev.ova exists", func() {
				It("prints an error", func() {
					mockVBox.EXPECT().IsVMRunning("pcfdev-2016-03-29_1728").Return(false)
					mockFS.EXPECT().CreateDir("/some/dir/.pcfdev").Return(nil)
					err := errors.New("some-error")
					mockFS.EXPECT().Exists("/some/dir/.pcfdev/pcfdev.ova").Return(false, err)
					mockUI.EXPECT().Failed("failed to fetch OVA: %s", err)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
				})
			})

			Context("pcfdev.ova already exists", func() {
				It("should start without downloading the ova", func() {
					vm := vbox.VM{
						IP:      "some-ip",
						SSHPort: "some-port",
					}
					mockVBox.EXPECT().IsVMRunning("pcfdev-2016-03-29_1728").Return(false)
					mockFS.EXPECT().CreateDir("/some/dir/.pcfdev").Return(nil)
					mockFS.EXPECT().Exists("/some/dir/.pcfdev/pcfdev.ova").Return(true, nil)
					mockUI.EXPECT().Say("pcfdev.ova already downloaded")
					mockUI.EXPECT().Say("Starting VM...")
					mockVBox.EXPECT().StartVM(gomock.Any(), "pcfdev-2016-03-29_1728").Return(&vm, nil)
					mockUI.EXPECT().Say("Provisioning VM...")
					mockSSH.EXPECT().RunSSHCommand("sudo /var/pcfdev/run local.pcfdev.io some-ip", "some-port")
					mockUI.EXPECT().Say("PCFDev is now running")

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
				})
			})

			Context("VM is already running", func() {
				It("prints a message and exits", func() {
					mockVBox.EXPECT().IsVMRunning("pcfdev-2016-03-29_1728").Return(true)
					mockUI.EXPECT().Say("PCFDev is already running")

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
				})
			})

			Context("OVA fails to download", func() {
				It("prints an error", func() {
					mockVBox.EXPECT().IsVMRunning("pcfdev-2016-03-29_1728").Return(false)
					mockFS.EXPECT().CreateDir("/some/dir/.pcfdev").Return(nil)
					mockFS.EXPECT().Exists("/some/dir/.pcfdev/pcfdev.ova").Return(false, nil)
					err := errors.New("some-error")
					mockUI.EXPECT().Say("Downloading OVA...")
					mockClient.EXPECT().DownloadOVA().Return(nil, err)
					mockUI.EXPECT().Failed("failed to fetch OVA: %s", err)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
				})
			})

			Context("VM fails to start", func() {
				It("prints an error", func() {
					mockVBox.EXPECT().IsVMRunning("pcfdev-2016-03-29_1728").Return(false)
					mockFS.EXPECT().CreateDir("/some/dir/.pcfdev").Return(nil)
					mockFS.EXPECT().Exists("/some/dir/.pcfdev/pcfdev.ova").Return(false, nil)
					readCloser := ioutil.NopCloser(strings.NewReader("some-ova-contents"))
					mockUI.EXPECT().Say("Downloading OVA...")
					mockClient.EXPECT().DownloadOVA().Return(readCloser, nil)
					mockFS.EXPECT().Write("/some/dir/.pcfdev/pcfdev.ova", readCloser).Return(nil)
					mockUI.EXPECT().Say("Finished downloading OVA")
					err := errors.New("some-error")
					mockUI.EXPECT().Say("Starting VM...")
					mockVBox.EXPECT().StartVM(gomock.Any(), "pcfdev-2016-03-29_1728").Return(nil, err)
					mockUI.EXPECT().Failed("failed to start VM: %s", err)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
				})
			})

			Context("VM fails to provision", func() {
				It("prints an error", func() {
					mockVBox.EXPECT().IsVMRunning("pcfdev-2016-03-29_1728").Return(false)
					mockFS.EXPECT().CreateDir("/some/dir/.pcfdev").Return(nil)
					mockFS.EXPECT().Exists("/some/dir/.pcfdev/pcfdev.ova").Return(false, nil)
					readCloser := ioutil.NopCloser(strings.NewReader("some-ova-contents"))
					mockUI.EXPECT().Say("Downloading OVA...")
					mockClient.EXPECT().DownloadOVA().Return(readCloser, nil)
					mockFS.EXPECT().Write("/some/dir/.pcfdev/pcfdev.ova", readCloser).Return(nil)
					mockUI.EXPECT().Say("Finished downloading OVA")
					vm := vbox.VM{
						IP:      "some-ip",
						SSHPort: "some-port",
					}
					mockUI.EXPECT().Say("Starting VM...")
					mockVBox.EXPECT().StartVM(gomock.Any(), "pcfdev-2016-03-29_1728").Return(&vm, nil)
					err := errors.New("some-error")
					mockUI.EXPECT().Say("Provisioning VM...")
					mockSSH.EXPECT().RunSSHCommand("sudo /var/pcfdev/run local.pcfdev.io some-ip", "some-port").Return(err)
					mockUI.EXPECT().Failed("failed to provision VM: %s", err)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "start"})
				})
			})
		})
		Context("stop", func() {
			It("should stop the vm", func() {
				mockUI.EXPECT().Say("Stopping VM...")
				mockVBox.EXPECT().StopVM("pcfdev-2016-03-29_1728").Return(nil)
				mockUI.EXPECT().Say("PCFDev is now stopped")

				pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "stop"})
			})
			Context("Vbox fails to stop VM", func() {
				It("should print an error", func() {
					err := errors.New("some-error")
					mockUI.EXPECT().Say("Stopping VM...")
					mockVBox.EXPECT().StopVM("pcfdev-2016-03-29_1728").Return(err)
					mockUI.EXPECT().Failed("failed to stop VM: %s", err)

					pcfdev.Run(&fakes.FakeCliConnection{}, []string{"dev", "stop"})
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

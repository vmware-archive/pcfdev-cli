package vm_test

import (
	"errors"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/vm"
	"github.com/pivotal-cf/pcfdev-cli/vm/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("LogFetcher", func() {
	var (
		mockCtrl   *gomock.Controller
		mockUI     *mocks.MockUI
		mockSSH    *mocks.MockSSH
		mockFS     *mocks.MockFS
		mockDriver *mocks.MockDriver
		logFetcher vm.LogFetcher
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockUI = mocks.NewMockUI(mockCtrl)
		mockSSH = mocks.NewMockSSH(mockCtrl)
		mockFS = mocks.NewMockFS(mockCtrl)
		mockDriver = mocks.NewMockDriver(mockCtrl)
		logFetcher = &vm.ConcreteLogFetcher{
			UI:     mockUI,
			SSH:    mockSSH,
			FS:     mockFS,
			Driver: mockDriver,

			VMConfig: &config.VMConfig{
				SSHPort: "some-port",
				Name:    "some-vm-name",
			},
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("#GetDebugLogs", func() {
		It("should say a message", func() {
			gomock.InOrder(
				mockFS.EXPECT().TempDir().Return("some-temp-dir", nil),
				mockSSH.EXPECT().GetSSHOutput("sudo cat /var/pcfdev/provision.log", "127.0.0.1", "some-port", 20*time.Second).Return("some-pcfdev-provision-log", nil),
				mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "provision.log"), strings.NewReader("some-pcfdev-provision-log")),
				mockSSH.EXPECT().GetSSHOutput("sudo cat /var/pcfdev/reset.log", "127.0.0.1", "some-port", 20*time.Second).Return("some-pcfdev-reset-log", nil),
				mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "reset.log"), strings.NewReader("some-pcfdev-reset-log")),
				mockSSH.EXPECT().GetSSHOutput("sudo cat /var/log/kern.log", "127.0.0.1", "some-port", 20*time.Second).Return("some-kern-log", nil),
				mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "kern.log"), strings.NewReader("some-kern-log")),
				mockSSH.EXPECT().GetSSHOutput("sudo cat /var/log/dmesg", "127.0.0.1", "some-port", 20*time.Second).Return("some-dmesg-log", nil),
				mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "dmesg"), strings.NewReader("some-dmesg-log")),
				mockSSH.EXPECT().GetSSHOutput("ifconfig", "127.0.0.1", "some-port", 20*time.Second).Return("some-ifconfig-log", nil),
				mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "ifconfig"), strings.NewReader("some-ifconfig-log")),
				mockSSH.EXPECT().GetSSHOutput("route -n", "127.0.0.1", "some-port", 20*time.Second).Return("some-routes-log", nil),
				mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "routes"), strings.NewReader("some-routes-log")),

				mockDriver.EXPECT().VBoxManage("list", "vms", "--long").Return([]byte("some-vm-list"), nil),
				mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "vm-list"), strings.NewReader("some-vm-list")),
				mockDriver.EXPECT().VBoxManage("showvminfo", "some-vm-name").Return([]byte("some-vm-info"), nil),
				mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "vm-info"), strings.NewReader("some-vm-info")),
				mockDriver.EXPECT().VBoxManage("list", "hostonlyifs", "--long").Return([]byte("some-vm-hostonlyifs"), nil),
				mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "vm-hostonlyifs"), strings.NewReader("some-vm-hostonlyifs")),

				mockFS.EXPECT().Compress(
					"pcfdev-debug",
					".",
					[]string{
						filepath.Join("some-temp-dir", "provision.log"),
						filepath.Join("some-temp-dir", "reset.log"),
						filepath.Join("some-temp-dir", "kern.log"),
						filepath.Join("some-temp-dir", "dmesg"),
						filepath.Join("some-temp-dir", "ifconfig"),
						filepath.Join("some-temp-dir", "routes"),
						filepath.Join("some-temp-dir", "vm-list"),
						filepath.Join("some-temp-dir", "vm-info"),
						filepath.Join("some-temp-dir", "vm-hostonlyifs"),
					}),

				mockUI.EXPECT().Say("Debug logs written to pcfdev-debug.tgz. While some scrubbing has taken place, please remove any remaining sensitive information from these logs before sharing."),
			)

			Expect(logFetcher.FetchLogs()).To(Succeed())
		})

		Context("when there is sensitive information", func() {
			It("should remove the sensitive information", func() {
				gomock.InOrder(
					mockFS.EXPECT().TempDir().Return("some-temp-dir", nil),
					mockSSH.EXPECT().GetSSHOutput("sudo cat /var/pcfdev/provision.log", "127.0.0.1", "some-port", 20*time.Second).Return("http://some-private-domain.com", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "provision.log"), strings.NewReader("<redacted uri>")),
					mockSSH.EXPECT().GetSSHOutput("sudo cat /var/pcfdev/reset.log", "127.0.0.1", "some-port", 20*time.Second).Return("some-pcfdev-reset-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "reset.log"), strings.NewReader("some-pcfdev-reset-log")),
					mockSSH.EXPECT().GetSSHOutput("sudo cat /var/log/kern.log", "127.0.0.1", "some-port", 20*time.Second).Return("some-kern-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "kern.log"), strings.NewReader("some-kern-log")),
					mockSSH.EXPECT().GetSSHOutput("sudo cat /var/log/dmesg", "127.0.0.1", "some-port", 20*time.Second).Return("some-dmesg-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "dmesg"), strings.NewReader("some-dmesg-log")),
					mockSSH.EXPECT().GetSSHOutput("ifconfig", "127.0.0.1", "some-port", 20*time.Second).Return("some-ifconfig-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "ifconfig"), strings.NewReader("some-ifconfig-log")),
					mockSSH.EXPECT().GetSSHOutput("route -n", "127.0.0.1", "some-port", 20*time.Second).Return("http://some-private-domain.com", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "routes"), strings.NewReader("http://some-private-domain.com")),

					mockDriver.EXPECT().VBoxManage("list", "vms", "--long").Return([]byte("http://some-private-domain.com"), nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "vm-list"), strings.NewReader("http://some-private-domain.com")),
					mockDriver.EXPECT().VBoxManage("showvminfo", "some-vm-name").Return([]byte("some-vm-info"), nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "vm-info"), strings.NewReader("some-vm-info")),
					mockDriver.EXPECT().VBoxManage("list", "hostonlyifs", "--long").Return([]byte("http://some-private-domain.com"), nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "vm-hostonlyifs"), strings.NewReader("http://some-private-domain.com")),

					mockFS.EXPECT().Compress(
						"pcfdev-debug",
						".",
						[]string{
							filepath.Join("some-temp-dir", "provision.log"),
							filepath.Join("some-temp-dir", "reset.log"),
							filepath.Join("some-temp-dir", "kern.log"),
							filepath.Join("some-temp-dir", "dmesg"),
							filepath.Join("some-temp-dir", "ifconfig"),
							filepath.Join("some-temp-dir", "routes"),
							filepath.Join("some-temp-dir", "vm-list"),
							filepath.Join("some-temp-dir", "vm-info"),
							filepath.Join("some-temp-dir", "vm-hostonlyifs"),
						}),

					mockUI.EXPECT().Say("Debug logs written to pcfdev-debug.tgz. While some scrubbing has taken place, please remove any remaining sensitive information from these logs before sharing."),
				)

				Expect(logFetcher.FetchLogs()).To(Succeed())
			})
		})

		Context("when there is an error creating a temporary directory", func() {
			It("should return the error", func() {
				mockFS.EXPECT().TempDir().Return("", errors.New("some-error"))

				Expect(logFetcher.FetchLogs()).To(MatchError("some-error"))
			})
		})

		Context("when there is an error getting ssh output of vm log file", func() {
			It("should return the error", func() {
				gomock.InOrder(
					mockFS.EXPECT().TempDir().Return("some-temp-dir", nil),
					mockSSH.EXPECT().GetSSHOutput("sudo cat /var/pcfdev/provision.log", "127.0.0.1", "some-port", 20*time.Second).Return("", errors.New("some-error")),
				)

				Expect(logFetcher.FetchLogs()).To(MatchError("some-error"))
			})
		})

		Context("when there is an error writing the temporary file for the vm log file", func() {
			It("should return the error", func() {
				gomock.InOrder(
					mockFS.EXPECT().TempDir().Return("some-temp-dir", nil),
					mockSSH.EXPECT().GetSSHOutput("sudo cat /var/pcfdev/provision.log", "127.0.0.1", "some-port", 20*time.Second).Return("some-pcfdev-provision-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "provision.log"), strings.NewReader("some-pcfdev-provision-log")).Return(errors.New("some-error")),
				)

				Expect(logFetcher.FetchLogs()).To(MatchError("some-error"))
			})
		})

		Context("when there is an error getting output of logging shell command", func() {
			It("should return the error", func() {
				gomock.InOrder(
					mockFS.EXPECT().TempDir().Return("some-temp-dir", nil),
					mockSSH.EXPECT().GetSSHOutput("sudo cat /var/pcfdev/provision.log", "127.0.0.1", "some-port", 20*time.Second).Return("some-pcfdev-provision-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "provision.log"), strings.NewReader("some-pcfdev-provision-log")),
					mockSSH.EXPECT().GetSSHOutput("sudo cat /var/pcfdev/reset.log", "127.0.0.1", "some-port", 20*time.Second).Return("some-pcfdev-reset-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "reset.log"), strings.NewReader("some-pcfdev-reset-log")),
					mockSSH.EXPECT().GetSSHOutput("sudo cat /var/log/kern.log", "127.0.0.1", "some-port", 20*time.Second).Return("some-kern-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "kern.log"), strings.NewReader("some-kern-log")),
					mockSSH.EXPECT().GetSSHOutput("sudo cat /var/log/dmesg", "127.0.0.1", "some-port", 20*time.Second).Return("some-dmesg-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "dmesg"), strings.NewReader("some-dmesg-log")),
					mockSSH.EXPECT().GetSSHOutput("ifconfig", "127.0.0.1", "some-port", 20*time.Second).Return("some-ifconfig-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "ifconfig"), strings.NewReader("some-ifconfig-log")),
					mockSSH.EXPECT().GetSSHOutput("route -n", "127.0.0.1", "some-port", 20*time.Second).Return("some-routes-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "routes"), strings.NewReader("some-routes-log")),

					mockDriver.EXPECT().VBoxManage("list", "vms", "--long").Return(nil, errors.New("some-error")),
				)

				Expect(logFetcher.FetchLogs()).To(MatchError("some-error"))
			})
		})

		Context("when there is an error writing the temporary file for the logging shell command", func() {
			It("should return the error", func() {
				gomock.InOrder(
					mockFS.EXPECT().TempDir().Return("some-temp-dir", nil),
					mockSSH.EXPECT().GetSSHOutput("sudo cat /var/pcfdev/provision.log", "127.0.0.1", "some-port", 20*time.Second).Return("some-pcfdev-provision-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "provision.log"), strings.NewReader("some-pcfdev-provision-log")),
					mockSSH.EXPECT().GetSSHOutput("sudo cat /var/pcfdev/reset.log", "127.0.0.1", "some-port", 20*time.Second).Return("some-pcfdev-reset-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "reset.log"), strings.NewReader("some-pcfdev-reset-log")),
					mockSSH.EXPECT().GetSSHOutput("sudo cat /var/log/kern.log", "127.0.0.1", "some-port", 20*time.Second).Return("some-kern-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "kern.log"), strings.NewReader("some-kern-log")),
					mockSSH.EXPECT().GetSSHOutput("sudo cat /var/log/dmesg", "127.0.0.1", "some-port", 20*time.Second).Return("some-dmesg-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "dmesg"), strings.NewReader("some-dmesg-log")),
					mockSSH.EXPECT().GetSSHOutput("ifconfig", "127.0.0.1", "some-port", 20*time.Second).Return("some-ifconfig-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "ifconfig"), strings.NewReader("some-ifconfig-log")),
					mockSSH.EXPECT().GetSSHOutput("route -n", "127.0.0.1", "some-port", 20*time.Second).Return("some-routes-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "routes"), strings.NewReader("some-routes-log")),

					mockDriver.EXPECT().VBoxManage("list", "vms", "--long").Return([]byte("some-vm-list"), nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "vm-list"), strings.NewReader("some-vm-list")).Return(errors.New("some-error")),
				)

				Expect(logFetcher.FetchLogs()).To(MatchError("some-error"))
			})
		})

		Context("when there is an error compressing a tar ball of the log files", func() {
			It("should return the error", func() {
				gomock.InOrder(
					mockFS.EXPECT().TempDir().Return("some-temp-dir", nil),
					mockSSH.EXPECT().GetSSHOutput("sudo cat /var/pcfdev/provision.log", "127.0.0.1", "some-port", 20*time.Second).Return("some-pcfdev-provision-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "provision.log"), strings.NewReader("some-pcfdev-provision-log")),
					mockSSH.EXPECT().GetSSHOutput("sudo cat /var/pcfdev/reset.log", "127.0.0.1", "some-port", 20*time.Second).Return("some-pcfdev-reset-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "reset.log"), strings.NewReader("some-pcfdev-reset-log")),
					mockSSH.EXPECT().GetSSHOutput("sudo cat /var/log/kern.log", "127.0.0.1", "some-port", 20*time.Second).Return("some-kern-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "kern.log"), strings.NewReader("some-kern-log")),
					mockSSH.EXPECT().GetSSHOutput("sudo cat /var/log/dmesg", "127.0.0.1", "some-port", 20*time.Second).Return("some-dmesg-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "dmesg"), strings.NewReader("some-dmesg-log")),
					mockSSH.EXPECT().GetSSHOutput("ifconfig", "127.0.0.1", "some-port", 20*time.Second).Return("some-ifconfig-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "ifconfig"), strings.NewReader("some-ifconfig-log")),
					mockSSH.EXPECT().GetSSHOutput("route -n", "127.0.0.1", "some-port", 20*time.Second).Return("some-routes-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "routes"), strings.NewReader("some-routes-log")),

					mockDriver.EXPECT().VBoxManage("list", "vms", "--long").Return([]byte("some-vm-list"), nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "vm-list"), strings.NewReader("some-vm-list")),
					mockDriver.EXPECT().VBoxManage("showvminfo", "some-vm-name").Return([]byte("some-vm-info"), nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "vm-info"), strings.NewReader("some-vm-info")),
					mockDriver.EXPECT().VBoxManage("list", "hostonlyifs", "--long").Return([]byte("some-vm-hostonlyifs"), nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "vm-hostonlyifs"), strings.NewReader("some-vm-hostonlyifs")),

					mockFS.EXPECT().Compress(
						"pcfdev-debug",
						".",
						[]string{
							filepath.Join("some-temp-dir", "provision.log"),
							filepath.Join("some-temp-dir", "reset.log"),
							filepath.Join("some-temp-dir", "kern.log"),
							filepath.Join("some-temp-dir", "dmesg"),
							filepath.Join("some-temp-dir", "ifconfig"),
							filepath.Join("some-temp-dir", "routes"),
							filepath.Join("some-temp-dir", "vm-list"),
							filepath.Join("some-temp-dir", "vm-info"),
							filepath.Join("some-temp-dir", "vm-hostonlyifs"),
						}).Return(errors.New("some-error")),
				)

				Expect(logFetcher.FetchLogs()).To(MatchError("some-error"))
			})
		})
	})
})

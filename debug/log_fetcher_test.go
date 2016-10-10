package debug_test

import (
	"errors"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/debug"
	"github.com/pivotal-cf/pcfdev-cli/debug/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("LogFetcher", func() {
	var (
		mockCtrl   *gomock.Controller
		mockSSH    *mocks.MockSSH
		mockFS     *mocks.MockFS
		mockDriver *mocks.MockDriver
		logFetcher *debug.LogFetcher
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockSSH = mocks.NewMockSSH(mockCtrl)
		mockFS = mocks.NewMockFS(mockCtrl)
		mockDriver = mocks.NewMockDriver(mockCtrl)
		logFetcher = &debug.LogFetcher{
			SSH:    mockSSH,
			FS:     mockFS,
			Driver: mockDriver,

			VMConfig: &config.VMConfig{
				SSHPort: "some-port",
				Name:    "some-vm-name",
			},

			Config: &config.Config{
				PrivateKeyPath: "some-private-key-path",
			},
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("#GetDebugLogs", func() {
		It("should say a message", func() {
			gomock.InOrder(
				mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
				mockFS.EXPECT().TempDir().Return("some-temp-dir", nil),
				mockSSH.EXPECT().GetSSHOutput("sudo cat /var/pcfdev/provision.log", "127.0.0.1", "some-port", []byte("some-private-key"), 20*time.Second).Return("some-pcfdev-provision-log", nil),
				mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "provision.log"), strings.NewReader("some-pcfdev-provision-log"), false),
				mockSSH.EXPECT().GetSSHOutput("sudo cat /var/pcfdev/reset.log", "127.0.0.1", "some-port", []byte("some-private-key"), 20*time.Second).Return("some-pcfdev-reset-log", nil),
				mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "reset.log"), strings.NewReader("some-pcfdev-reset-log"), false),
				mockSSH.EXPECT().GetSSHOutput("sudo cat /var/log/kern.log", "127.0.0.1", "some-port", []byte("some-private-key"), 20*time.Second).Return("some-kern-log", nil),
				mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "kern.log"), strings.NewReader("some-kern-log"), false),
				mockSSH.EXPECT().GetSSHOutput("sudo cat /var/log/dmesg", "127.0.0.1", "some-port", []byte("some-private-key"), 20*time.Second).Return("some-dmesg-log", nil),
				mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "dmesg"), strings.NewReader("some-dmesg-log"), false),
				mockSSH.EXPECT().GetSSHOutput("ifconfig", "127.0.0.1", "some-port", []byte("some-private-key"), 20*time.Second).Return("some-ifconfig-log", nil),
				mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "ifconfig"), strings.NewReader("some-ifconfig-log"), false),
				mockSSH.EXPECT().GetSSHOutput("route -n", "127.0.0.1", "some-port", []byte("some-private-key"), 20*time.Second).Return("some-routes-log", nil),
				mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "routes"), strings.NewReader("some-routes-log"), false),

				mockDriver.EXPECT().VBoxManage("list", "vms", "--long").Return([]byte("some-vm-list"), nil),
				mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "vm-list"), strings.NewReader("some-vm-list"), false),
				mockDriver.EXPECT().VBoxManage("showvminfo", "some-vm-name").Return([]byte("some-vm-info"), nil),
				mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "vm-info"), strings.NewReader("some-vm-info"), false),
				mockDriver.EXPECT().VBoxManage("list", "hostonlyifs", "--long").Return([]byte("some-vm-hostonlyifs"), nil),
				mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "vm-hostonlyifs"), strings.NewReader("some-vm-hostonlyifs"), false),

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
			)

			Expect(logFetcher.FetchLogs()).To(Succeed())
		})

		Context("when there is sensitive information", func() {
			It("should remove the sensitive information", func() {
				gomock.InOrder(
					mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
					mockFS.EXPECT().TempDir().Return("some-temp-dir", nil),
					mockSSH.EXPECT().GetSSHOutput("sudo cat /var/pcfdev/provision.log", "127.0.0.1", "some-port", []byte("some-private-key"), 20*time.Second).Return("http://some-private-domain.com", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "provision.log"), strings.NewReader("<redacted uri>"), false),
					mockSSH.EXPECT().GetSSHOutput("sudo cat /var/pcfdev/reset.log", "127.0.0.1", "some-port", []byte("some-private-key"), 20*time.Second).Return("some-pcfdev-reset-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "reset.log"), strings.NewReader("some-pcfdev-reset-log"), false),
					mockSSH.EXPECT().GetSSHOutput("sudo cat /var/log/kern.log", "127.0.0.1", "some-port", []byte("some-private-key"), 20*time.Second).Return("some-kern-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "kern.log"), strings.NewReader("some-kern-log"), false),
					mockSSH.EXPECT().GetSSHOutput("sudo cat /var/log/dmesg", "127.0.0.1", "some-port", []byte("some-private-key"), 20*time.Second).Return("some-dmesg-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "dmesg"), strings.NewReader("some-dmesg-log"), false),
					mockSSH.EXPECT().GetSSHOutput("ifconfig", "127.0.0.1", "some-port", []byte("some-private-key"), 20*time.Second).Return("some-ifconfig-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "ifconfig"), strings.NewReader("some-ifconfig-log"), false),
					mockSSH.EXPECT().GetSSHOutput("route -n", "127.0.0.1", "some-port", []byte("some-private-key"), 20*time.Second).Return("http://some-private-domain.com", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "routes"), strings.NewReader("http://some-private-domain.com"), false),

					mockDriver.EXPECT().VBoxManage("list", "vms", "--long").Return([]byte("http://some-private-domain.com"), nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "vm-list"), strings.NewReader("http://some-private-domain.com"), false),
					mockDriver.EXPECT().VBoxManage("showvminfo", "some-vm-name").Return([]byte("some-vm-info"), nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "vm-info"), strings.NewReader("some-vm-info"), false),
					mockDriver.EXPECT().VBoxManage("list", "hostonlyifs", "--long").Return([]byte("http://some-private-domain.com"), nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "vm-hostonlyifs"), strings.NewReader("http://some-private-domain.com"), false),

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
				)

				Expect(logFetcher.FetchLogs()).To(Succeed())
			})
		})

		Context("when there is an error creating a temporary directory", func() {
			It("should return the error", func() {
				gomock.InOrder(
					mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
					mockFS.EXPECT().TempDir().Return("", errors.New("some-error")),
				)

				Expect(logFetcher.FetchLogs()).To(MatchError("some-error"))
			})
		})

		Context("when there is an error getting ssh output of vm log file", func() {
			It("should return the error", func() {
				gomock.InOrder(
					mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
					mockFS.EXPECT().TempDir().Return("some-temp-dir", nil),
					mockSSH.EXPECT().GetSSHOutput("sudo cat /var/pcfdev/provision.log", "127.0.0.1", "some-port", []byte("some-private-key"), 20*time.Second).Return("", errors.New("some-error")),
				)

				Expect(logFetcher.FetchLogs()).To(MatchError("some-error"))
			})
		})

		Context("when there is an error writing the temporary file for the vm log file", func() {
			It("should return the error", func() {
				gomock.InOrder(
					mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
					mockFS.EXPECT().TempDir().Return("some-temp-dir", nil),
					mockSSH.EXPECT().GetSSHOutput("sudo cat /var/pcfdev/provision.log", "127.0.0.1", "some-port", []byte("some-private-key"), 20*time.Second).Return("some-pcfdev-provision-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "provision.log"), strings.NewReader("some-pcfdev-provision-log"), false).Return(errors.New("some-error")),
				)

				Expect(logFetcher.FetchLogs()).To(MatchError("some-error"))
			})
		})

		Context("when there is an error getting output of logging shell command", func() {
			It("should return the error", func() {
				gomock.InOrder(
					mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
					mockFS.EXPECT().TempDir().Return("some-temp-dir", nil),
					mockSSH.EXPECT().GetSSHOutput("sudo cat /var/pcfdev/provision.log", "127.0.0.1", "some-port", []byte("some-private-key"), 20*time.Second).Return("some-pcfdev-provision-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "provision.log"), strings.NewReader("some-pcfdev-provision-log"), false),
					mockSSH.EXPECT().GetSSHOutput("sudo cat /var/pcfdev/reset.log", "127.0.0.1", "some-port", []byte("some-private-key"), 20*time.Second).Return("some-pcfdev-reset-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "reset.log"), strings.NewReader("some-pcfdev-reset-log"), false),
					mockSSH.EXPECT().GetSSHOutput("sudo cat /var/log/kern.log", "127.0.0.1", "some-port", []byte("some-private-key"), 20*time.Second).Return("some-kern-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "kern.log"), strings.NewReader("some-kern-log"), false),
					mockSSH.EXPECT().GetSSHOutput("sudo cat /var/log/dmesg", "127.0.0.1", "some-port", []byte("some-private-key"), 20*time.Second).Return("some-dmesg-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "dmesg"), strings.NewReader("some-dmesg-log"), false),
					mockSSH.EXPECT().GetSSHOutput("ifconfig", "127.0.0.1", "some-port", []byte("some-private-key"), 20*time.Second).Return("some-ifconfig-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "ifconfig"), strings.NewReader("some-ifconfig-log"), false),
					mockSSH.EXPECT().GetSSHOutput("route -n", "127.0.0.1", "some-port", []byte("some-private-key"), 20*time.Second).Return("some-routes-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "routes"), strings.NewReader("some-routes-log"), false),

					mockDriver.EXPECT().VBoxManage("list", "vms", "--long").Return(nil, errors.New("some-error")),
				)

				Expect(logFetcher.FetchLogs()).To(MatchError("some-error"))
			})
		})

		Context("when there is an error writing the temporary file for the logging shell command", func() {
			It("should return the error", func() {
				gomock.InOrder(
					mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
					mockFS.EXPECT().TempDir().Return("some-temp-dir", nil),
					mockSSH.EXPECT().GetSSHOutput("sudo cat /var/pcfdev/provision.log", "127.0.0.1", "some-port", []byte("some-private-key"), 20*time.Second).Return("some-pcfdev-provision-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "provision.log"), strings.NewReader("some-pcfdev-provision-log"), false),
					mockSSH.EXPECT().GetSSHOutput("sudo cat /var/pcfdev/reset.log", "127.0.0.1", "some-port", []byte("some-private-key"), 20*time.Second).Return("some-pcfdev-reset-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "reset.log"), strings.NewReader("some-pcfdev-reset-log"), false),
					mockSSH.EXPECT().GetSSHOutput("sudo cat /var/log/kern.log", "127.0.0.1", "some-port", []byte("some-private-key"), 20*time.Second).Return("some-kern-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "kern.log"), strings.NewReader("some-kern-log"), false),
					mockSSH.EXPECT().GetSSHOutput("sudo cat /var/log/dmesg", "127.0.0.1", "some-port", []byte("some-private-key"), 20*time.Second).Return("some-dmesg-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "dmesg"), strings.NewReader("some-dmesg-log"), false),
					mockSSH.EXPECT().GetSSHOutput("ifconfig", "127.0.0.1", "some-port", []byte("some-private-key"), 20*time.Second).Return("some-ifconfig-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "ifconfig"), strings.NewReader("some-ifconfig-log"), false),
					mockSSH.EXPECT().GetSSHOutput("route -n", "127.0.0.1", "some-port", []byte("some-private-key"), 20*time.Second).Return("some-routes-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "routes"), strings.NewReader("some-routes-log"), false),

					mockDriver.EXPECT().VBoxManage("list", "vms", "--long").Return([]byte("some-vm-list"), nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "vm-list"), strings.NewReader("some-vm-list"), false).Return(errors.New("some-error")),
				)

				Expect(logFetcher.FetchLogs()).To(MatchError("some-error"))
			})
		})

		Context("when there is an error compressing a tar ball of the log files", func() {
			It("should return the error", func() {
				gomock.InOrder(
					mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
					mockFS.EXPECT().TempDir().Return("some-temp-dir", nil),
					mockSSH.EXPECT().GetSSHOutput("sudo cat /var/pcfdev/provision.log", "127.0.0.1", "some-port", []byte("some-private-key"), 20*time.Second).Return("some-pcfdev-provision-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "provision.log"), strings.NewReader("some-pcfdev-provision-log"), false),
					mockSSH.EXPECT().GetSSHOutput("sudo cat /var/pcfdev/reset.log", "127.0.0.1", "some-port", []byte("some-private-key"), 20*time.Second).Return("some-pcfdev-reset-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "reset.log"), strings.NewReader("some-pcfdev-reset-log"), false),
					mockSSH.EXPECT().GetSSHOutput("sudo cat /var/log/kern.log", "127.0.0.1", "some-port", []byte("some-private-key"), 20*time.Second).Return("some-kern-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "kern.log"), strings.NewReader("some-kern-log"), false),
					mockSSH.EXPECT().GetSSHOutput("sudo cat /var/log/dmesg", "127.0.0.1", "some-port", []byte("some-private-key"), 20*time.Second).Return("some-dmesg-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "dmesg"), strings.NewReader("some-dmesg-log"), false),
					mockSSH.EXPECT().GetSSHOutput("ifconfig", "127.0.0.1", "some-port", []byte("some-private-key"), 20*time.Second).Return("some-ifconfig-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "ifconfig"), strings.NewReader("some-ifconfig-log"), false),
					mockSSH.EXPECT().GetSSHOutput("route -n", "127.0.0.1", "some-port", []byte("some-private-key"), 20*time.Second).Return("some-routes-log", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "routes"), strings.NewReader("some-routes-log"), false),

					mockDriver.EXPECT().VBoxManage("list", "vms", "--long").Return([]byte("some-vm-list"), nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "vm-list"), strings.NewReader("some-vm-list"), false),
					mockDriver.EXPECT().VBoxManage("showvminfo", "some-vm-name").Return([]byte("some-vm-info"), nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "vm-info"), strings.NewReader("some-vm-info"), false),
					mockDriver.EXPECT().VBoxManage("list", "hostonlyifs", "--long").Return([]byte("some-vm-hostonlyifs"), nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "vm-hostonlyifs"), strings.NewReader("some-vm-hostonlyifs"), false),

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

		Context("when there is an error reading the private key", func() {
			It("should return the error", func() {
				mockFS.EXPECT().Read("some-private-key-path").Return(nil, errors.New("some-error"))

				Expect(logFetcher.FetchLogs()).To(MatchError("some-error"))
			})
		})
	})
})

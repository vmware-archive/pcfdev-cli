package vm_test

import (
	"errors"
	"os"
	"time"

	"github.com/golang/mock/gomock"
	conf "github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/ssh"
	"github.com/pivotal-cf/pcfdev-cli/vm"
	"github.com/pivotal-cf/pcfdev-cli/vm/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Unprovisioned", func() {
	var (
		mockCtrl       *gomock.Controller
		mockFS         *mocks.MockFS
		mockUI         *mocks.MockUI
		mockVBox       *mocks.MockVBox
		mockSSH        *mocks.MockSSH
		mockClient     *mocks.MockClient
		mockLogFetcher *mocks.MockLogFetcher
		mockHelpText   *mocks.MockHelpText
		unprovisioned  vm.Unprovisioned
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockVBox = mocks.NewMockVBox(mockCtrl)
		mockUI = mocks.NewMockUI(mockCtrl)
		mockFS = mocks.NewMockFS(mockCtrl)
		mockSSH = mocks.NewMockSSH(mockCtrl)
		mockClient = mocks.NewMockClient(mockCtrl)
		mockLogFetcher = mocks.NewMockLogFetcher(mockCtrl)
		mockHelpText = mocks.NewMockHelpText(mockCtrl)

		unprovisioned = vm.Unprovisioned{
			UI:         mockUI,
			VBox:       mockVBox,
			FS:         mockFS,
			SSHClient:  mockSSH,
			LogFetcher: mockLogFetcher,
			HelpText:   mockHelpText,
			Client:     mockClient,
			Config: &conf.Config{
				PrivateKeyPath: "some-private-key-path",
			},
			VMConfig: &conf.VMConfig{
				Name:    "some-vm",
				Domain:  "some-domain",
				IP:      "some-ip",
				SSHPort: "some-port",
			},
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("Stop", func() {
		It("should stop the VM", func() {
			gomock.InOrder(
				mockUI.EXPECT().Say("Stopping VM..."),
				mockVBox.EXPECT().StopVM(unprovisioned.VMConfig),
				mockUI.EXPECT().Say("PCF Dev is now stopped."),
			)

			Expect(unprovisioned.Stop()).To(Succeed())
		})
	})

	Describe("VerifyStartOpts", func() {
		It("should say a message", func() {
			Expect(unprovisioned.VerifyStartOpts(
				&vm.StartOpts{},
			)).To(MatchError("PCF Dev is in an invalid state. Please run 'cf dev destroy' or 'cf dev stop'"))
		})
	})

	Describe("Start", func() {
		It("should start vm", func() {
			Expect(unprovisioned.Start(&vm.StartOpts{})).To(MatchError("PCF Dev is in an invalid state. Please run 'cf dev destroy' or 'cf dev stop'"))
		})
	})

	Describe("Provision", func() {
		It("should provision the VM", func() {
			sshAddresses := []ssh.SSHAddress{
				{IP: "127.0.0.1", Port: "some-port"},
				{IP: "some-ip", Port: "22"},
			}
			gomock.InOrder(
				mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
				mockSSH.EXPECT().RunSSHCommand(
					"if [ -e /var/pcfdev/provision-options.json ]; then exit 0; else exit 1; fi",
					sshAddresses,
					[]byte("some-private-key"),
					30*time.Second,
					os.Stdout,
					os.Stderr,
				),
				mockSSH.EXPECT().GetSSHOutput(
					"cat /var/pcfdev/provision-options.json",
					sshAddresses,
					[]byte("some-private-key"),
					30*time.Second,
				).Return(`{"domain":"some-domain","ip":"some-ip","services":"some-service,some-other-service","registries":["some-registry","some-other-registry"],"provider":"some-provider"}`, nil),
				mockUI.EXPECT().Say("Provisioning VM..."),
				mockSSH.EXPECT().RunSSHCommand(
					`sudo -H /var/pcfdev/provision "some-domain" "some-ip" "some-service,some-other-service" "some-registry,some-other-registry" "some-provider"`,
					sshAddresses,
					[]byte("some-private-key"),
					5*time.Minute,
					os.Stdout,
					os.Stderr,
				),
				mockHelpText.EXPECT().Print("some-domain", false),
			)

			Expect(unprovisioned.Provision(&vm.StartOpts{})).To(Succeed())
		})

		Context("when the user passes in a master password", func() {
			It("should provision the VM after replacing the secrets", func() {
				sshAddresses := []ssh.SSHAddress{
					{IP: "127.0.0.1", Port: "some-port"},
					{IP: "some-ip", Port: "22"},
				}
				gomock.InOrder(
					mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
					mockClient.EXPECT().ReplaceSecrets("some-ip", "some-master-password", []byte("some-private-key")),
					mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
					mockSSH.EXPECT().RunSSHCommand(
						"if [ -e /var/pcfdev/provision-options.json ]; then exit 0; else exit 1; fi",
						sshAddresses,
						[]byte("some-private-key"),
						30*time.Second,
						os.Stdout,
						os.Stderr,
					),
					mockSSH.EXPECT().GetSSHOutput(
						"cat /var/pcfdev/provision-options.json",
						sshAddresses,
						[]byte("some-private-key"),
						30*time.Second,
					).Return(`{"domain":"some-domain","ip":"some-ip","services":"some-service,some-other-service","registries":["some-registry","some-other-registry"],"provider":"some-provider"}`, nil),
					mockUI.EXPECT().Say("Provisioning VM..."),
					mockSSH.EXPECT().RunSSHCommand(
						`sudo -H /var/pcfdev/provision "some-domain" "some-ip" "some-service,some-other-service" "some-registry,some-other-registry" "some-provider"`,
						sshAddresses,
						[]byte("some-private-key"),
						5*time.Minute,
						os.Stdout,
						os.Stderr,
					),
					mockHelpText.EXPECT().Print("some-domain", false),
				)

				Expect(unprovisioned.Provision(&vm.StartOpts{MasterPassword: "some-master-password"})).To(Succeed())
			})
		})

		Context("when the user passes in a master password and there is an error", func() {
			It("should return the error", func() {
				gomock.InOrder(
					mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
					mockClient.EXPECT().ReplaceSecrets("some-ip", "some-master-password", []byte("some-private-key")).Return(errors.New("some-error")),
				)

				Expect(unprovisioned.Provision(&vm.StartOpts{MasterPassword: "some-master-password"})).To(MatchError("some-error"))
			})
		})

		Context("when the user passes in a master password and there is an error reading the private key", func() {
			It("should return the error", func() {
				mockFS.EXPECT().Read("some-private-key-path").Return(nil, errors.New("some-error"))

				Expect(unprovisioned.Provision(&vm.StartOpts{MasterPassword: "some-master-password"})).To(MatchError("some-error"))
			})
		})

		Context("when the VM is autotargeted", func() {
			It("should provision the VM", func() {
				sshAddresses := []ssh.SSHAddress{
					{IP: "127.0.0.1", Port: "some-port"},
					{IP: "some-ip", Port: "22"},
				}
				gomock.InOrder(

					mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
					mockSSH.EXPECT().RunSSHCommand("if [ -e /var/pcfdev/provision-options.json ]; then exit 0; else exit 1; fi",
						sshAddresses,
						[]byte("some-private-key"),
						30*time.Second,
						os.Stdout,
						os.Stderr,
					),
					mockSSH.EXPECT().GetSSHOutput(
						"cat /var/pcfdev/provision-options.json",
						sshAddresses,
						[]byte("some-private-key"),
						30*time.Second,
					).Return(`{"domain":"some-domain","ip":"some-ip","services":"some-service,some-other-service","registries":["some-registry","some-other-registry"],"provider":"some-provider"}`, nil),
					mockUI.EXPECT().Say("Provisioning VM..."),
					mockSSH.EXPECT().RunSSHCommand(
						`sudo -H /var/pcfdev/provision "some-domain" "some-ip" "some-service,some-other-service" "some-registry,some-other-registry" "some-provider"`,
						sshAddresses,
						[]byte("some-private-key"),
						5*time.Minute,
						os.Stdout,
						os.Stderr,
					),
					mockHelpText.EXPECT().Print("some-domain", true),
				)

				Expect(unprovisioned.Provision(&vm.StartOpts{Target: true})).To(Succeed())
			})
		})

		Context("when reading the private key fails", func() {
			It("should return an error", func() {
				mockFS.EXPECT().Read("some-private-key-path").Return(nil, errors.New("some-error"))

				Expect(unprovisioned.Provision(&vm.StartOpts{})).To(MatchError("some-error"))
			})
		})

		Context("when there is an error finding the provision config", func() {
			It("should return an error", func() {
				sshAddresses := []ssh.SSHAddress{
					{IP: "127.0.0.1", Port: "some-port"},
					{IP: "some-ip", Port: "22"},
				}
				gomock.InOrder(
					mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
					mockSSH.EXPECT().RunSSHCommand("if [ -e /var/pcfdev/provision-options.json ]; then exit 0; else exit 1; fi",
						sshAddresses,
						[]byte("some-private-key"),
						30*time.Second,
						os.Stdout,
						os.Stderr).Return(errors.New("some-error")),
				)

				Expect(unprovisioned.Provision(&vm.StartOpts{})).To(MatchError("failed to provision VM: missing provision configuration"))
			})
		})

		Context("when there is an error reading the provision config", func() {
			It("should return an error", func() {
				sshAddresses := []ssh.SSHAddress{
					{IP: "127.0.0.1", Port: "some-port"},
					{IP: "some-ip", Port: "22"},
				}
				gomock.InOrder(
					mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
					mockSSH.EXPECT().RunSSHCommand("if [ -e /var/pcfdev/provision-options.json ]; then exit 0; else exit 1; fi",
						sshAddresses,
						[]byte("some-private-key"),
						30*time.Second,
						os.Stdout,
						os.Stderr),
					mockSSH.EXPECT().GetSSHOutput("cat /var/pcfdev/provision-options.json", sshAddresses, []byte("some-private-key"), 30*time.Second).Return("", errors.New("some-error")),
				)

				Expect(unprovisioned.Provision(&vm.StartOpts{})).To(MatchError("failed to provision VM: some-error"))
			})
		})

		Context("when there is an error parsing the provision config", func() {
			It("should return an error", func() {
				sshAddresses := []ssh.SSHAddress{
					{IP: "127.0.0.1", Port: "some-port"},
					{IP: "some-ip", Port: "22"},
				}
				gomock.InOrder(
					mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
					mockSSH.EXPECT().RunSSHCommand("if [ -e /var/pcfdev/provision-options.json ]; then exit 0; else exit 1; fi",
						sshAddresses,
						[]byte("some-private-key"),
						30*time.Second,
						os.Stdout,
						os.Stderr),
					mockSSH.EXPECT().GetSSHOutput("cat /var/pcfdev/provision-options.json", sshAddresses, []byte("some-private-key"), 30*time.Second).Return("{some-bad-json}", nil),
				)

				Expect(unprovisioned.Provision(&vm.StartOpts{})).To(MatchError(ContainSubstring(`failed to provision VM: invalid character 's'`)))
			})
		})
	})

	Describe("Status", func() {
		It("should say a message", func() {
			Expect(unprovisioned.Status()).To(Equal("PCF Dev is in an invalid state. Please run 'cf dev destroy' or 'cf dev stop'"))
		})
	})

	Describe("Suspend", func() {
		It("should return an error", func() {
			Expect(unprovisioned.Suspend()).To(MatchError("PCF Dev is in an invalid state. Please run 'cf dev destroy' or 'cf dev stop'"))
		})
	})

	Describe("Resume", func() {
		It("should return an error", func() {
			Expect(unprovisioned.Resume()).To(MatchError("PCF Dev is in an invalid state. Please run 'cf dev destroy' or 'cf dev stop'"))
		})
	})

	Describe("Trust", func() {
		It("should return an error", func() {
			Expect(unprovisioned.Trust(&vm.StartOpts{})).To(MatchError("PCF Dev is in an invalid state. Please run 'cf dev destroy' or 'cf dev stop'"))
		})
	})

	Describe("Target", func() {
		It("should return an error", func() {
			Expect(unprovisioned.Target(false)).To(MatchError("PCF Dev is in an invalid state. Please run 'cf dev destroy' or 'cf dev stop'"))
		})
	})

	Describe("SSH", func() {
		It("should execute ssh on the client", func() {
			addresses := []ssh.SSHAddress{
				{IP: "127.0.0.1", Port: "some-port"},
				{IP: "some-ip", Port: "22"},
			}
			gomock.InOrder(
				mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
				mockSSH.EXPECT().StartSSHSession(addresses, []byte("some-private-key"), 5*time.Minute, os.Stdin, os.Stdout, os.Stderr),
			)

			Expect(unprovisioned.SSH()).To(Succeed())
		})

		Context("when reading the private key fails", func() {
			It("should return an error", func() {
				mockFS.EXPECT().Read("some-private-key-path").Return(nil, errors.New("some-error"))

				Expect(unprovisioned.SSH()).To(MatchError("some-error"))
			})
		})

		Context("when executing ssh fails", func() {
			It("should return an error", func() {
				addresses := []ssh.SSHAddress{
					{IP: "127.0.0.1", Port: "some-port"},
					{IP: "some-ip", Port: "22"},
				}
				gomock.InOrder(
					mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
					mockSSH.EXPECT().StartSSHSession(addresses, []byte("some-private-key"), 5*time.Minute, os.Stdin, os.Stdout, os.Stderr).Return(errors.New("some-error")),
				)

				Expect(unprovisioned.SSH()).To(MatchError("some-error"))
			})
		})
	})

	Describe("GetDebugLogs", func() {
		It("should succeed", func() {
			mockLogFetcher.EXPECT().FetchLogs()

			Expect(unprovisioned.GetDebugLogs()).To(Succeed())
		})

		Context("when fetching logs fails", func() {
			It("should return the error", func() {
				mockLogFetcher.EXPECT().FetchLogs().Return(errors.New("some-error"))

				Expect(unprovisioned.GetDebugLogs()).To(MatchError("failed to retrieve logs: some-error"))
			})
		})
	})

})

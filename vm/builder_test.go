package vm_test

import (
	"errors"
	"github.com/golang/mock/gomock"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/vbox"
	"github.com/pivotal-cf/pcfdev-cli/vm"
	"github.com/pivotal-cf/pcfdev-cli/vm/mocks"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Builder", func() {
	Describe("#VM", func() {
		var (
			mockCtrl   *gomock.Controller
			mockVBox   *mocks.MockVBox
			mockFS     *mocks.MockFS
			mockSSH    *mocks.MockSSH
			mockClient *mocks.MockClient
			builder    *vm.VBoxBuilder
			conf       *config.Config
		)

		BeforeEach(func() {
			mockCtrl = gomock.NewController(GinkgoT())
			mockVBox = mocks.NewMockVBox(mockCtrl)
			mockFS = mocks.NewMockFS(mockCtrl)
			mockSSH = mocks.NewMockSSH(mockCtrl)
			mockClient = mocks.NewMockClient(mockCtrl)
			conf = &config.Config{
				MinMemory:      100,
				MaxMemory:      200,
				VMDir:          "some-vm-dir",
				PrivateKeyPath: "some-private-key-path",
			}

			builder = &vm.VBoxBuilder{
				VBox:   mockVBox,
				FS:     mockFS,
				SSH:    mockSSH,
				Client: mockClient,
				Config: conf,
			}
		})

		AfterEach(func() {
			mockCtrl.Finish()
		})

		Context("when vm is not created", func() {
			It("should return a not created VM", func() {
				gomock.InOrder(
					mockVBox.EXPECT().VMStatus("some-vm").Return(vbox.StatusNotCreated, nil),
					mockFS.EXPECT().Exists(filepath.Join("some-vm-dir", "some-vm")).Return(false, nil),
				)

				notCreatedVM, err := builder.VM("some-vm")
				Expect(err).NotTo(HaveOccurred())

				switch u := notCreatedVM.(type) {
				case *vm.NotCreated:
					Expect(u.VMConfig.Name).To(Equal("some-vm"))
					Expect(u.Network).NotTo(BeNil())
				default:
					Fail("wrong type")
				}
			})
			Context("when the disk exists", func() {
				It("should return an invalid vm", func() {
					gomock.InOrder(
						mockVBox.EXPECT().VMStatus("some-vm").Return(vbox.StatusNotCreated, nil),
						mockFS.EXPECT().Exists(filepath.Join("some-vm-dir", "some-vm")).Return(true, nil),
					)

					invalidVM, err := builder.VM("some-vm")
					Expect(err).NotTo(HaveOccurred())

					switch u := invalidVM.(type) {
					case *vm.Invalid:
						Expect(u.Err).NotTo(BeNil())
					default:
						Fail("wrong type")
					}
				})
			})

			Context("when the disk does not exist", func() {
				It("should return an invalid vm", func() {
					gomock.InOrder(
						mockVBox.EXPECT().VMStatus("some-vm").Return(vbox.StatusNotCreated, nil),
						mockFS.EXPECT().Exists(filepath.Join("some-vm-dir", "some-vm")).Return(false, errors.New("some-error")),
					)

					_, err := builder.VM("some-vm")
					Expect(err).To(MatchError("some-error"))
				})
			})
		})

		Context("when vm is created", func() {
			Context("when vm is stopped", func() {
				It("should return a stopped vm", func() {
					expectedVMConfig := &config.VMConfig{}
					gomock.InOrder(
						mockVBox.EXPECT().VMStatus("some-vm").Return(vbox.StatusStopped, nil),
						mockVBox.EXPECT().VMConfig("some-vm").Return(expectedVMConfig, nil),
					)

					stoppedVM, err := builder.VM("some-vm")
					Expect(err).NotTo(HaveOccurred())

					switch u := stoppedVM.(type) {
					case *vm.Stopped:
						Expect(u.VMConfig).To(BeIdenticalTo(expectedVMConfig))
						Expect(u.SSHClient).NotTo(BeNil())
						Expect(u.VBox).NotTo(BeNil())
						Expect(u.UI).NotTo(BeNil())
					default:
						Fail("wrong type")
					}
				})
			})

			Context("when there is an error getting the vm config", func() {
				It("should return an invalid vm", func() {
					gomock.InOrder(
						mockVBox.EXPECT().VMStatus("some-vm").Return(vbox.StatusStopped, nil),
						mockVBox.EXPECT().VMConfig("some-vm").Return(nil, errors.New("some-error")),
					)

					invalidVM, err := builder.VM("some-vm")
					Expect(err).NotTo(HaveOccurred())

					switch u := invalidVM.(type) {
					case *vm.Invalid:
						Expect(u.Err).NotTo(BeNil())
					default:
						Fail("wrong type")
					}
				})
			})

			Context("when vm is running and healthcheck passes", func() {
				It("should return a running vm", func() {
					expectedVMConfig := &config.VMConfig{
						IP:      "192.168.11.11",
						Memory:  uint64(3456),
						SSHPort: "some-port",
						Domain:  "local.pcfdev.io",
					}

					gomock.InOrder(
						mockVBox.EXPECT().VMStatus("some-vm").Return(vbox.StatusRunning, nil),
						mockVBox.EXPECT().VMConfig("some-vm").Return(expectedVMConfig, nil),
						mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
						mockClient.EXPECT().Status("192.168.11.11", []byte("some-private-key")).Return("Running", nil),
					)

					runningVM, err := builder.VM("some-vm")
					Expect(err).NotTo(HaveOccurred())

					switch u := runningVM.(type) {
					case *vm.Running:
						Expect(u.Config).To(BeIdenticalTo(conf))
						Expect(u.VMConfig).To(BeIdenticalTo(expectedVMConfig))
						Expect(u.VBox).NotTo(BeNil())
						Expect(u.UI).NotTo(BeNil())
						Expect(u.SSHClient).NotTo(BeNil())
						Expect(u.FS).NotTo(BeNil())
						Expect(u.LogFetcher).NotTo(BeNil())
						Expect(u.Builder).NotTo(BeNil())
						Expect(u.CertStore).NotTo(BeNil())
						Expect(u.CmdRunner).NotTo(BeNil())
						Expect(u.HelpText).NotTo(BeNil())
					default:
						Fail("wrong type")
					}
				})
			})

			Context("when vm is running and healthcheck fails on regular ssh port", func() {
				It("should return a unprovisioned vm", func() {
					expectedVMConfig := &config.VMConfig{
						IP:      "192.168.11.11",
						Memory:  uint64(3456),
						SSHPort: "some-port",
						Domain:  "local.pcfdev.io",
					}

					gomock.InOrder(
						mockVBox.EXPECT().VMStatus("some-vm").Return(vbox.StatusRunning, nil),
						mockVBox.EXPECT().VMConfig("some-vm").Return(expectedVMConfig, nil),
						mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
						mockClient.EXPECT().Status("192.168.11.11", []byte("some-private-key")).Return("Unprovisioned", nil),
					)

					unprovisionedVM, err := builder.VM("some-vm")
					Expect(err).NotTo(HaveOccurred())

					switch u := unprovisionedVM.(type) {
					case *vm.Unprovisioned:
						Expect(u.UI).NotTo(BeNil())
						Expect(u.VBox).NotTo(BeNil())
						Expect(u.LogFetcher).NotTo(BeNil())
						Expect(u.VMConfig).To(BeIdenticalTo(expectedVMConfig))
						Expect(u.Client).To(BeIdenticalTo(builder.Client))
					default:
						Fail("wrong type")
					}
				})
			})

			Context("when vm is running but the client returns an unexpected status", func() {
				It("should return a unprovisioned vm", func() {
					expectedVMConfig := &config.VMConfig{
						IP:      "192.168.11.11",
						Memory:  uint64(3456),
						SSHPort: "some-port",
						Domain:  "local.pcfdev.io",
					}

					gomock.InOrder(
						mockVBox.EXPECT().VMStatus("some-vm").Return(vbox.StatusRunning, nil),
						mockVBox.EXPECT().VMConfig("some-vm").Return(expectedVMConfig, nil),
						mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
						mockClient.EXPECT().Status("192.168.11.11", []byte("some-private-key")).Return("some-unexpected-status", nil),
					)

					invalidVM, err := builder.VM("some-vm")
					Expect(err).NotTo(HaveOccurred())

					switch u := invalidVM.(type) {
					case *vm.Invalid:
						Expect(u.Err).NotTo(BeNil())
					default:
						Fail("wrong type")
					}
				})
			})

			Context("when getting private key returns an error", func() {
				It("should return an invalid vm", func() {
					expectedVMConfig := &config.VMConfig{
						IP:      "192.168.11.11",
						Memory:  uint64(3456),
						SSHPort: "some-port",
						Domain:  "local.pcfdev.io",
					}

					gomock.InOrder(
						mockVBox.EXPECT().VMStatus("some-vm").Return(vbox.StatusRunning, nil),
						mockVBox.EXPECT().VMConfig("some-vm").Return(expectedVMConfig, nil),
						mockFS.EXPECT().Read("some-private-key-path").Return(nil, errors.New("some-error")),
					)

					invalidVM, err := builder.VM("some-vm")
					Expect(err).NotTo(HaveOccurred())

					switch u := invalidVM.(type) {
					case *vm.Invalid:
						Expect(u.Err).To(MatchError("unable to read private key"))
					default:
						Fail("wrong type")
					}
				})
			})

			Context("when vm is running but the client returns an error", func() {
				It("should return an unprovisioned vm", func() {
					expectedVMConfig := &config.VMConfig{
						IP:      "192.168.11.11",
						Memory:  uint64(3456),
						SSHPort: "some-port",
						Domain:  "local.pcfdev.io",
					}

					gomock.InOrder(
						mockVBox.EXPECT().VMStatus("some-vm").Return(vbox.StatusRunning, nil),
						mockVBox.EXPECT().VMConfig("some-vm").Return(expectedVMConfig, nil),
						mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
						mockClient.EXPECT().Status("192.168.11.11", []byte("some-private-key")).Return("", errors.New("some-error")),
					)

					unprovisionedVM, err := builder.VM("some-vm")
					Expect(err).NotTo(HaveOccurred())

					switch u := unprovisionedVM.(type) {
					case *vm.Unprovisioned:
						Expect(u.UI).NotTo(BeNil())
						Expect(u.VBox).NotTo(BeNil())
						Expect(u.LogFetcher).NotTo(BeNil())
						Expect(u.VMConfig).To(BeIdenticalTo(expectedVMConfig))
						Expect(u.Client).To(BeIdenticalTo(builder.Client))
					default:
						Fail("wrong type")
					}
				})
			})

			Context("when vm is suspended in memory", func() {
				It("should return a paused vm", func() {
					expectedVMConfig := &config.VMConfig{
						IP:      "192.168.11.11",
						Memory:  uint64(3456),
						SSHPort: "some-port",
						Domain:  "local.pcfdev.io",
					}
					gomock.InOrder(
						mockVBox.EXPECT().VMStatus("some-vm").Return(vbox.StatusPaused, nil),
						mockVBox.EXPECT().VMConfig("some-vm").Return(expectedVMConfig, nil),
					)

					pausedVM, err := builder.VM("some-vm")
					Expect(err).NotTo(HaveOccurred())

					switch u := pausedVM.(type) {
					case *vm.Paused:
						Expect(u.VMConfig).To(BeIdenticalTo(expectedVMConfig))
						Expect(u.Config).To(BeIdenticalTo(conf))
						Expect(u.VBox).NotTo(BeNil())
						Expect(u.UI).NotTo(BeNil())
						Expect(u.FS).NotTo(BeNil())
					default:
						Fail("wrong type")
					}
				})
			})

			Context("when vm is suspended to disk", func() {
				It("should return a suspended vm", func() {
					expectedVMConfig := &config.VMConfig{
						IP:      "192.168.11.11",
						Memory:  uint64(3456),
						SSHPort: "some-port",
						Domain:  "local.pcfdev.io",
					}
					gomock.InOrder(
						mockVBox.EXPECT().VMStatus("some-vm").Return(vbox.StatusSaved, nil),
						mockVBox.EXPECT().VMConfig("some-vm").Return(expectedVMConfig, nil),
					)

					savedVM, err := builder.VM("some-vm")
					Expect(err).NotTo(HaveOccurred())

					switch u := savedVM.(type) {
					case *vm.Saved:
						Expect(u.VMConfig).To(BeIdenticalTo(expectedVMConfig))
						Expect(u.VBox).NotTo(BeNil())
						Expect(u.UI).NotTo(BeNil())
						Expect(u.Config).To(BeIdenticalTo(conf))
						Expect(u.FS).NotTo(BeNil())
					default:
						Fail("wrong type")
					}
				})
			})

			Context("when vm state is something unexpected", func() {
				It("should return an error", func() {
					expectedVMConfig := &config.VMConfig{
						IP:      "192.168.11.11",
						Memory:  uint64(3456),
						SSHPort: "some-port",
						Domain:  "local.pcfdev.io",
					}
					gomock.InOrder(
						mockVBox.EXPECT().VMStatus("some-vm").Return(vbox.StatusUnknown, nil),
						mockVBox.EXPECT().VMConfig("some-vm").Return(expectedVMConfig, nil),
					)

					invalidVM, err := builder.VM("some-vm")
					Expect(err).NotTo(HaveOccurred())

					switch u := invalidVM.(type) {
					case *vm.Invalid:
						Expect(u.Err).NotTo(BeNil())
					default:
						Fail("wrong type")
					}
				})
			})

			Context("when there is an error retrieving vm status", func() {
				It("should return an error", func() {
					mockVBox.EXPECT().VMStatus("some-vm").Return("", errors.New("some-error"))

					_, err := builder.VM("some-vm")
					Expect(err).To(MatchError("some-error"))
				})
			})

		})
	})
})

package vm_test

import (
	"errors"
	"path/filepath"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/vbox"
	"github.com/pivotal-cf/pcfdev-cli/vm"
	"github.com/pivotal-cf/pcfdev-cli/vm/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Builder", func() {
	Describe("#VM", func() {
		var (
			mockCtrl *gomock.Controller
			mockVBox *mocks.MockVBox
			mockFS   *mocks.MockFS
			mockSSH  *mocks.MockSSH
			builder  vm.Builder
		)

		BeforeEach(func() {
			mockCtrl = gomock.NewController(GinkgoT())
			mockVBox = mocks.NewMockVBox(mockCtrl)
			mockFS = mocks.NewMockFS(mockCtrl)
			mockSSH = mocks.NewMockSSH(mockCtrl)

			builder = &vm.VBoxBuilder{
				VBox: mockVBox,
				FS:   mockFS,
				SSH:  mockSSH,
				Config: &config.Config{
					MinMemory: 100,
					MaxMemory: 200,
					VMDir:     "some-vm-dir",
				},
			}
		})

		AfterEach(func() {
			mockCtrl.Finish()
		})

		Context("when vm is not created", func() {
			It("should return a not created VM", func() {
				gomock.InOrder(
					mockVBox.EXPECT().VMExists("some-vm").Return(false, nil),
					mockFS.EXPECT().Exists(filepath.Join("some-vm-dir", "some-vm")).Return(false, nil),
				)

				notCreatedVM, err := builder.VM("some-vm")
				Expect(err).NotTo(HaveOccurred())

				switch u := notCreatedVM.(type) {
				case *vm.NotCreated:
					Expect(u.VMConfig.Name).To(Equal("some-vm"))
				default:
					Fail("wrong type")
				}
			})
			Context("when the disk exists", func() {
				It("should return an invalid vm", func() {
					gomock.InOrder(
						mockVBox.EXPECT().VMExists("some-vm").Return(false, nil),
						mockFS.EXPECT().Exists(filepath.Join("some-vm-dir", "some-vm")).Return(true, nil),
					)

					invalidVM, err := builder.VM("some-vm")
					Expect(err).NotTo(HaveOccurred())

					switch u := invalidVM.(type) {
					case *vm.Invalid:
						Expect(u.Err).NotTo(BeNil())
						Expect(u.UI).NotTo(BeNil())
					default:
						Fail("wrong type")
					}
				})
			})

			Context("when the disk exists", func() {
				It("should return an invalid vm", func() {
					gomock.InOrder(
						mockVBox.EXPECT().VMExists("some-vm").Return(false, nil),
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
						mockVBox.EXPECT().VMExists("some-vm").Return(true, nil),
						mockVBox.EXPECT().VMConfig("some-vm").Return(expectedVMConfig, nil),
						mockVBox.EXPECT().VMState("some-vm").Return(vbox.StateStopped, nil),
					)

					stoppedVM, err := builder.VM("some-vm")
					Expect(err).NotTo(HaveOccurred())

					switch u := stoppedVM.(type) {
					case *vm.Stopped:
						Expect(u.VMConfig).To(BeIdenticalTo(expectedVMConfig))
						Expect(u.SSH).NotTo(BeNil())
						Expect(u.VBox).NotTo(BeNil())
						Expect(u.UI).NotTo(BeNil())
					default:
						Fail("wrong type")
					}
				})
			})

			Context("when vm is aborted", func() {
				It("should return a stopped vm", func() {
					expectedVMConfig := &config.VMConfig{}
					gomock.InOrder(
						mockVBox.EXPECT().VMExists("some-vm").Return(true, nil),
						mockVBox.EXPECT().VMConfig("some-vm").Return(expectedVMConfig, nil),
						mockVBox.EXPECT().VMState("some-vm").Return(vbox.StateAborted, nil),
					)

					abortedVM, err := builder.VM("some-vm")
					Expect(err).NotTo(HaveOccurred())

					switch u := abortedVM.(type) {
					case *vm.Stopped:
						Expect(u.VMConfig).To(BeIdenticalTo(expectedVMConfig))
						Expect(u.SSH).NotTo(BeNil())
						Expect(u.VBox).NotTo(BeNil())
						Expect(u.UI).NotTo(BeNil())
					default:
						Fail("wrong type")
					}
				})
			})

			Context("when there is an error seeing if vm exists", func() {
				It("should return an error", func() {
					mockVBox.EXPECT().VMExists("some-vm").Return(false, errors.New("some-error"))
					_, err := builder.VM("some-vm")
					Expect(err).To(MatchError("some-error"))
				})
			})

			Context("when there is an error getting the vm config", func() {
				It("should return an invalid vm", func() {
					gomock.InOrder(
						mockVBox.EXPECT().VMExists("some-vm").Return(true, nil),
						mockVBox.EXPECT().VMConfig("some-vm").Return(nil, errors.New("some-error")),
					)

					invalidVM, err := builder.VM("some-vm")
					Expect(err).NotTo(HaveOccurred())

					switch u := invalidVM.(type) {
					case *vm.Invalid:
						Expect(u.Err).NotTo(BeNil())
						Expect(u.UI).NotTo(BeNil())
					default:
						Fail("wrong type")
					}
				})
			})

			Context("when vm is running and healthcheck passes on regular ssh port", func() {
				It("should return a running vm", func() {
					healthCheckCommand := "sudo /var/pcfdev/health-check"
					expectedVMConfig := &config.VMConfig{
						IP:      "192.168.11.11",
						Memory:  uint64(3456),
						SSHPort: "some-port",
						Domain:  "local.pcfdev.io",
					}

					gomock.InOrder(
						mockVBox.EXPECT().VMExists("some-vm").Return(true, nil),
						mockVBox.EXPECT().VMConfig("some-vm").Return(expectedVMConfig, nil),
						mockVBox.EXPECT().VMState("some-vm").Return(vbox.StateRunning, nil),
					)
					mockSSH.EXPECT().GetSSHOutput(healthCheckCommand, "192.168.11.11", "22", 20*time.Second).AnyTimes().Do(
						func(string, string, string, time.Duration) { time.Sleep(time.Minute) },
					)
					mockSSH.EXPECT().GetSSHOutput(healthCheckCommand, "127.0.0.1", "some-port", 20*time.Second).Return("ok\n", nil)

					runningVM, err := builder.VM("some-vm")
					Expect(err).NotTo(HaveOccurred())

					switch u := runningVM.(type) {
					case *vm.Running:
						Expect(u.VMConfig).To(BeIdenticalTo(expectedVMConfig))
						Expect(u.VBox).NotTo(BeNil())
						Expect(u.UI).NotTo(BeNil())
						Expect(u.SSH).NotTo(BeNil())
						Expect(u.Builder).NotTo(BeNil())
					default:
						Fail("wrong type")
					}
				})
			})

			Context("when vm is running and healthcheck passes on forwarded ssh port", func() {
				It("should return a running vm", func() {
					healthCheckCommand := "sudo /var/pcfdev/health-check"
					expectedVMConfig := &config.VMConfig{
						IP:      "192.168.11.11",
						Memory:  uint64(3456),
						SSHPort: "some-port",
						Domain:  "local.pcfdev.io",
					}

					gomock.InOrder(
						mockVBox.EXPECT().VMExists("some-vm").Return(true, nil),
						mockVBox.EXPECT().VMConfig("some-vm").Return(expectedVMConfig, nil),
						mockVBox.EXPECT().VMState("some-vm").Return(vbox.StateRunning, nil),
					)
					mockSSH.EXPECT().GetSSHOutput(healthCheckCommand, "127.0.0.1", "some-port", 20*time.Second).AnyTimes().Do(
						func(string, string, string, time.Duration) { time.Sleep(time.Minute) },
					)
					mockSSH.EXPECT().GetSSHOutput(healthCheckCommand, "192.168.11.11", "22", 20*time.Second).Return("ok\n", nil)

					runningVM, err := builder.VM("some-vm")
					Expect(err).NotTo(HaveOccurred())

					switch u := runningVM.(type) {
					case *vm.Running:
						Expect(u.VMConfig).To(BeIdenticalTo(expectedVMConfig))
						Expect(u.VBox).NotTo(BeNil())
						Expect(u.UI).NotTo(BeNil())
					default:
						Fail("wrong type")
					}
				})
			})

			Context("when vm is running and healthcheck fails on regular ssh port", func() {
				It("should return a unprovisioned vm", func() {
					healthCheckCommand := "sudo /var/pcfdev/health-check"
					expectedVMConfig := &config.VMConfig{
						IP:      "192.168.11.11",
						Memory:  uint64(3456),
						SSHPort: "some-port",
						Domain:  "local.pcfdev.io",
					}

					mockVBox.EXPECT().VMExists("some-vm").Return(true, nil)
					mockVBox.EXPECT().VMConfig("some-vm").Return(expectedVMConfig, nil)
					mockVBox.EXPECT().VMState("some-vm").Return(vbox.StateRunning, nil)
					mockSSH.EXPECT().GetSSHOutput(healthCheckCommand, "127.0.0.1", "some-port", 20*time.Second).AnyTimes().Do(
						func(string, string, string, time.Duration) { time.Sleep(time.Minute) },
					)
					mockSSH.EXPECT().GetSSHOutput(healthCheckCommand, "192.168.11.11", "22", 20*time.Second).Return("", nil)

					unprovisionedVM, err := builder.VM("some-vm")
					Expect(err).NotTo(HaveOccurred())

					switch u := unprovisionedVM.(type) {
					case *vm.Unprovisioned:
						Expect(u.UI).NotTo(BeNil())
						Expect(u.VBox).NotTo(BeNil())
						Expect(u.VMConfig).To(BeIdenticalTo(expectedVMConfig))
					default:
						Fail("wrong type")
					}
				})
			})

			Context("when vm is running and healthcheck fails on forwarded port", func() {
				It("should return a unprovisioned vm", func() {
					healthCheckCommand := "sudo /var/pcfdev/health-check"
					expectedVMConfig := &config.VMConfig{
						IP:      "192.168.11.11",
						Memory:  uint64(3456),
						SSHPort: "some-port",
						Domain:  "local.pcfdev.io",
					}
					mockVBox.EXPECT().VMExists("some-vm").Return(true, nil)
					mockVBox.EXPECT().VMConfig("some-vm").Return(expectedVMConfig, nil)
					mockVBox.EXPECT().VMState("some-vm").Return(vbox.StateRunning, nil)
					mockSSH.EXPECT().GetSSHOutput(healthCheckCommand, "192.168.11.11", "22", 20*time.Second).AnyTimes().Do(
						func(string, string, string, time.Duration) { time.Sleep(time.Minute) },
					)
					mockSSH.EXPECT().GetSSHOutput(healthCheckCommand, "127.0.0.1", "some-port", 20*time.Second).Return("", nil)

					unprovisionedVM, err := builder.VM("some-vm")
					Expect(err).NotTo(HaveOccurred())

					switch u := unprovisionedVM.(type) {
					case *vm.Unprovisioned:
						Expect(u.UI).NotTo(BeNil())
						Expect(u.VBox).NotTo(BeNil())
						Expect(u.VMConfig).To(BeIdenticalTo(expectedVMConfig))
					default:
						Fail("wrong type")
					}
				})
			})

			Context("when vm is saved", func() {
				It("should return a suspended vm", func() {
					expectedVMConfig := &config.VMConfig{
						IP:      "192.168.11.11",
						Memory:  uint64(3456),
						SSHPort: "some-port",
						Domain:  "local.pcfdev.io",
					}
					gomock.InOrder(
						mockVBox.EXPECT().VMExists("some-vm").Return(true, nil),
						mockVBox.EXPECT().VMConfig("some-vm").Return(expectedVMConfig, nil),
						mockVBox.EXPECT().VMState("some-vm").Return(vbox.StateSaved, nil),
					)

					suspendedVM, err := builder.VM("some-vm")
					Expect(err).NotTo(HaveOccurred())

					switch u := suspendedVM.(type) {
					case *vm.Suspended:
						Expect(u.VMConfig).To(BeIdenticalTo(expectedVMConfig))
						Expect(u.VBox).NotTo(BeNil())
						Expect(u.UI).NotTo(BeNil())
					default:
						Fail("wrong type")
					}
				})
			})

			Context("when vm is paused", func() {
				It("should return a suspended vm", func() {
					expectedVMConfig := &config.VMConfig{
						IP:      "192.168.11.11",
						Memory:  uint64(3456),
						SSHPort: "some-port",
						Domain:  "local.pcfdev.io",
					}
					gomock.InOrder(
						mockVBox.EXPECT().VMExists("some-vm").Return(true, nil),
						mockVBox.EXPECT().VMConfig("some-vm").Return(expectedVMConfig, nil),
						mockVBox.EXPECT().VMState("some-vm").Return(vbox.StatePaused, nil),
					)

					suspendedVM, err := builder.VM("some-vm")
					Expect(err).NotTo(HaveOccurred())

					switch u := suspendedVM.(type) {
					case *vm.Suspended:
						Expect(u.VMConfig).To(BeIdenticalTo(expectedVMConfig))
						Expect(u.VBox).NotTo(BeNil())
						Expect(u.UI).NotTo(BeNil())
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
						mockVBox.EXPECT().VMExists("some-vm").Return(true, nil),
						mockVBox.EXPECT().VMConfig("some-vm").Return(expectedVMConfig, nil),
						mockVBox.EXPECT().VMState("some-vm").Return("some-unexpected-state", nil),
					)

					vm, err := builder.VM("some-vm")
					Expect(err).To(MatchError("failed to handle VM state 'some-unexpected-state'"))
					Expect(vm).To(BeNil())
				})
			})
		})
	})
})

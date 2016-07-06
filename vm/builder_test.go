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
			mockCtrl   *gomock.Controller
			mockDriver *mocks.MockDriver
			mockFS     *mocks.MockFS
			mockSSH    *mocks.MockSSH
			builder    vm.Builder
		)

		BeforeEach(func() {
			mockCtrl = gomock.NewController(GinkgoT())
			mockDriver = mocks.NewMockDriver(mockCtrl)
			mockFS = mocks.NewMockFS(mockCtrl)
			mockSSH = mocks.NewMockSSH(mockCtrl)

			builder = &vm.VBoxBuilder{
				Driver: mockDriver,
				FS:     mockFS,
				SSH:    mockSSH,
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
					mockDriver.EXPECT().VMExists("some-vm").Return(false, nil),
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
						mockDriver.EXPECT().VMExists("some-vm").Return(false, nil),
						mockFS.EXPECT().Exists(filepath.Join("some-vm-dir", "some-vm")).Return(true, nil),
					)

					invalidVM, err := builder.VM("some-vm")
					Expect(err).NotTo(HaveOccurred())

					switch u := invalidVM.(type) {
					case *vm.Invalid:
						Expect(u.UI).NotTo(BeNil())
					default:
						Fail("wrong type")
					}
				})
			})

			Context("when the disk exists", func() {
				It("should return an invalid vm", func() {
					gomock.InOrder(
						mockDriver.EXPECT().VMExists("some-vm").Return(false, nil),
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
					gomock.InOrder(
						mockDriver.EXPECT().VMExists("some-vm").Return(true, nil),
						mockDriver.EXPECT().GetVMIP("some-vm").Return("192.168.11.11", nil),
						mockDriver.EXPECT().GetMemory("some-vm").Return(uint64(3456), nil),
						mockDriver.EXPECT().GetHostForwardPort("some-vm", "ssh").Return("some-port", nil),
						mockDriver.EXPECT().VMState("some-vm").Return(vbox.StateStopped, nil),
					)

					stoppedVM, err := builder.VM("some-vm")
					Expect(err).NotTo(HaveOccurred())

					switch u := stoppedVM.(type) {
					case *vm.Stopped:
						Expect(u.VMConfig.Name).To(Equal("some-vm"))
						Expect(u.VMConfig.IP).To(Equal("192.168.11.11"))
						Expect(u.VMConfig.SSHPort).To(Equal("some-port"))
						Expect(u.VMConfig.Domain).To(Equal("local.pcfdev.io"))
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
					gomock.InOrder(
						mockDriver.EXPECT().VMExists("some-vm").Return(true, nil),
						mockDriver.EXPECT().GetVMIP("some-vm").Return("192.168.11.11", nil),
						mockDriver.EXPECT().GetMemory("some-vm").Return(uint64(3456), nil),
						mockDriver.EXPECT().GetHostForwardPort("some-vm", "ssh").Return("some-port", nil),
						mockDriver.EXPECT().VMState("some-vm").Return(vbox.StateAborted, nil),
					)

					abortedVM, err := builder.VM("some-vm")
					Expect(err).NotTo(HaveOccurred())

					switch u := abortedVM.(type) {
					case *vm.Stopped:
						Expect(u.VMConfig.Name).To(Equal("some-vm"))
						Expect(u.VMConfig.IP).To(Equal("192.168.11.11"))
						Expect(u.VMConfig.SSHPort).To(Equal("some-port"))
						Expect(u.VMConfig.Domain).To(Equal("local.pcfdev.io"))
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
					mockDriver.EXPECT().VMExists("some-vm").Return(false, errors.New("some-error"))
					_, err := builder.VM("some-vm")
					Expect(err).To(MatchError("some-error"))
				})
			})

			Context("when there is an error getting the vm IP", func() {
				It("should return an invalid vm", func() {
					gomock.InOrder(
						mockDriver.EXPECT().VMExists("some-vm").Return(true, nil),
						mockDriver.EXPECT().GetVMIP("some-vm").Return("", errors.New("some-error")),
					)

					invalidVM, err := builder.VM("some-vm")
					Expect(err).NotTo(HaveOccurred())

					switch u := invalidVM.(type) {
					case *vm.Invalid:
						Expect(u.UI).NotTo(BeNil())
					default:
						Fail("wrong type")
					}
				})
			})

			Context("when there is an error getting domain for vm ip", func() {
				It("should return an invalid vm", func() {
					gomock.InOrder(
						mockDriver.EXPECT().VMExists("some-vm").Return(true, nil),
						mockDriver.EXPECT().GetVMIP("some-vm").Return("some-ip", nil),
					)

					invalidVM, err := builder.VM("some-vm")
					Expect(err).NotTo(HaveOccurred())

					switch u := invalidVM.(type) {
					case *vm.Invalid:
						Expect(u.UI).NotTo(BeNil())
					default:
						Fail("wrong type")
					}
				})
			})

			Context("when there is an error getting vm host forward port", func() {
				It("should return an invalid vm", func() {
					gomock.InOrder(
						mockDriver.EXPECT().VMExists("some-vm").Return(true, nil),
						mockDriver.EXPECT().GetVMIP("some-vm").Return("192.168.11.11", nil),
						mockDriver.EXPECT().GetMemory("some-vm").Return(uint64(3456), nil),
						mockDriver.EXPECT().GetHostForwardPort("some-vm", "ssh").Return("", errors.New("some-error")),
					)

					invalidVM, err := builder.VM("some-vm")
					Expect(err).NotTo(HaveOccurred())

					switch u := invalidVM.(type) {
					case *vm.Invalid:
						Expect(u.UI).NotTo(BeNil())
					default:
						Fail("wrong type")
					}
				})
			})

			Context("when vm is running and healthcheck passes on regular ssh port", func() {
				It("should return a running vm", func() {
					healthCheckCommand := "sudo /var/pcfdev/health-check"

					gomock.InOrder(
						mockDriver.EXPECT().VMExists("some-vm").Return(true, nil),
						mockDriver.EXPECT().GetVMIP("some-vm").Return("192.168.11.11", nil),
						mockDriver.EXPECT().GetMemory("some-vm").Return(uint64(3456), nil),
						mockDriver.EXPECT().GetHostForwardPort("some-vm", "ssh").Return("some-port", nil),
						mockDriver.EXPECT().VMState("some-vm").Return(vbox.StateRunning, nil),
					)
					mockSSH.EXPECT().GetSSHOutput(healthCheckCommand, "192.168.11.11", "22", 20*time.Second).AnyTimes().Do(
						func(string, string, string, time.Duration) { time.Sleep(time.Minute) },
					)
					mockSSH.EXPECT().GetSSHOutput(healthCheckCommand, "127.0.0.1", "some-port", 20*time.Second).Return("ok\n", nil)

					runningVM, err := builder.VM("some-vm")
					Expect(err).NotTo(HaveOccurred())

					switch u := runningVM.(type) {
					case *vm.Running:
						Expect(u.VMConfig.Name).To(Equal("some-vm"))
						Expect(u.VMConfig.IP).To(Equal("192.168.11.11"))
						Expect(u.VMConfig.SSHPort).To(Equal("some-port"))
						Expect(u.VMConfig.Domain).To(Equal("local.pcfdev.io"))
						Expect(u.VBox).NotTo(BeNil())
						Expect(u.UI).NotTo(BeNil())
					default:
						Fail("wrong type")
					}
				})
			})

			Context("when vm is running and healthcheck passes on forwarded ssh port", func() {
				It("should return a running vm", func() {
					healthCheckCommand := "sudo /var/pcfdev/health-check"

					gomock.InOrder(
						mockDriver.EXPECT().VMExists("some-vm").Return(true, nil),
						mockDriver.EXPECT().GetVMIP("some-vm").Return("192.168.11.11", nil),
						mockDriver.EXPECT().GetMemory("some-vm").Return(uint64(3456), nil),
						mockDriver.EXPECT().GetHostForwardPort("some-vm", "ssh").Return("some-port", nil),
						mockDriver.EXPECT().VMState("some-vm").Return(vbox.StateRunning, nil),
					)
					mockSSH.EXPECT().GetSSHOutput(healthCheckCommand, "127.0.0.1", "some-port", 20*time.Second).AnyTimes().Do(
						func(string, string, string, time.Duration) { time.Sleep(time.Minute) },
					)
					mockSSH.EXPECT().GetSSHOutput(healthCheckCommand, "192.168.11.11", "22", 20*time.Second).Return("ok\n", nil)

					runningVM, err := builder.VM("some-vm")
					Expect(err).NotTo(HaveOccurred())

					switch u := runningVM.(type) {
					case *vm.Running:
						Expect(u.VMConfig.Name).To(Equal("some-vm"))
						Expect(u.VMConfig.IP).To(Equal("192.168.11.11"))
						Expect(u.VMConfig.SSHPort).To(Equal("some-port"))
						Expect(u.VMConfig.Domain).To(Equal("local.pcfdev.io"))
						Expect(u.VBox).NotTo(BeNil())
						Expect(u.UI).NotTo(BeNil())
					default:
						Fail("wrong type")
					}
				})
			})

			Context("when vm is running and healthcheck fails on regular ssh port", func() {
				It("should return a recoverable vm", func() {
					healthCheckCommand := "sudo /var/pcfdev/health-check"
					done := make(chan bool, 1)

					mockDriver.EXPECT().VMExists("some-vm").Return(true, nil)
					mockDriver.EXPECT().GetVMIP("some-vm").Return("192.168.11.11", nil)
					mockDriver.EXPECT().GetMemory("some-vm").Return(uint64(3456), nil)
					mockDriver.EXPECT().GetHostForwardPort("some-vm", "ssh").Return("some-port", nil)
					mockDriver.EXPECT().VMState("some-vm").Return(vbox.StateRunning, nil)
					mockSSH.EXPECT().GetSSHOutput(healthCheckCommand, "127.0.0.1", "some-port", 20*time.Second).AnyTimes().Do(
						func(string, string, string, time.Duration) { time.Sleep(time.Minute) },
					)
					mockSSH.EXPECT().GetSSHOutput(healthCheckCommand, "192.168.11.11", "22", 20*time.Second).Return("", nil)

					recoverableVM, err := builder.VM("some-vm")
					done <- true
					Expect(err).NotTo(HaveOccurred())

					switch u := recoverableVM.(type) {
					case *vm.Recoverable:
						Expect(u.UI).NotTo(BeNil())
						Expect(u.VBox).NotTo(BeNil())
						Expect(u.VMConfig.Name).To(Equal("some-vm"))
						Expect(u.VMConfig.IP).To(Equal("192.168.11.11"))
						Expect(u.VMConfig.SSHPort).To(Equal("some-port"))
						Expect(u.VMConfig.Domain).To(Equal("local.pcfdev.io"))
					default:
						Fail("wrong type")
					}
				})
			})

			Context("when vm is running and healthcheck fails on forwarded port", func() {
				It("should return a recoverable vm", func() {
					healthCheckCommand := "sudo /var/pcfdev/health-check"

					mockDriver.EXPECT().VMExists("some-vm").Return(true, nil)
					mockDriver.EXPECT().GetVMIP("some-vm").Return("192.168.11.11", nil)
					mockDriver.EXPECT().GetMemory("some-vm").Return(uint64(3456), nil)
					mockDriver.EXPECT().GetHostForwardPort("some-vm", "ssh").Return("some-port", nil)
					mockDriver.EXPECT().VMState("some-vm").Return(vbox.StateRunning, nil)
					mockSSH.EXPECT().GetSSHOutput(healthCheckCommand, "192.168.11.11", "22", 20*time.Second).AnyTimes().Do(
						func(string, string, string, time.Duration) { time.Sleep(time.Minute) },
					)
					mockSSH.EXPECT().GetSSHOutput(healthCheckCommand, "127.0.0.1", "some-port", 20*time.Second).Return("", nil)

					recoverableVM, err := builder.VM("some-vm")
					Expect(err).NotTo(HaveOccurred())

					switch u := recoverableVM.(type) {
					case *vm.Recoverable:
						Expect(u.UI).NotTo(BeNil())
						Expect(u.VBox).NotTo(BeNil())
						Expect(u.VMConfig.Name).To(Equal("some-vm"))
						Expect(u.VMConfig.IP).To(Equal("192.168.11.11"))
						Expect(u.VMConfig.SSHPort).To(Equal("some-port"))
						Expect(u.VMConfig.Domain).To(Equal("local.pcfdev.io"))
					default:
						Fail("wrong type")
					}
				})
			})

			Context("when vm is saved", func() {
				It("should return a suspended vm", func() {
					gomock.InOrder(
						mockDriver.EXPECT().VMExists("some-vm").Return(true, nil),
						mockDriver.EXPECT().GetVMIP("some-vm").Return("192.168.11.11", nil),
						mockDriver.EXPECT().GetMemory("some-vm").Return(uint64(3456), nil),
						mockDriver.EXPECT().GetHostForwardPort("some-vm", "ssh").Return("some-port", nil),
						mockDriver.EXPECT().VMState("some-vm").Return(vbox.StateSaved, nil),
					)

					suspendedVM, err := builder.VM("some-vm")
					Expect(err).NotTo(HaveOccurred())

					switch u := suspendedVM.(type) {
					case *vm.Suspended:
						Expect(u.VMConfig.Name).To(Equal("some-vm"))
						Expect(u.VMConfig.IP).To(Equal("192.168.11.11"))
						Expect(u.VMConfig.SSHPort).To(Equal("some-port"))
						Expect(u.VMConfig.Domain).To(Equal("local.pcfdev.io"))
						Expect(u.VBox).NotTo(BeNil())
						Expect(u.UI).NotTo(BeNil())
					default:
						Fail("wrong type")
					}
				})
			})

			Context("when vm is paused", func() {
				It("should return a suspended vm", func() {
					gomock.InOrder(
						mockDriver.EXPECT().VMExists("some-vm").Return(true, nil),
						mockDriver.EXPECT().GetVMIP("some-vm").Return("192.168.11.11", nil),
						mockDriver.EXPECT().GetMemory("some-vm").Return(uint64(3456), nil),
						mockDriver.EXPECT().GetHostForwardPort("some-vm", "ssh").Return("some-port", nil),
						mockDriver.EXPECT().VMState("some-vm").Return(vbox.StatePaused, nil),
					)

					suspendedVM, err := builder.VM("some-vm")
					Expect(err).NotTo(HaveOccurred())

					switch u := suspendedVM.(type) {
					case *vm.Suspended:
						Expect(u.VMConfig.Name).To(Equal("some-vm"))
						Expect(u.VMConfig.IP).To(Equal("192.168.11.11"))
						Expect(u.VMConfig.SSHPort).To(Equal("some-port"))
						Expect(u.VMConfig.Domain).To(Equal("local.pcfdev.io"))
						Expect(u.VBox).NotTo(BeNil())
						Expect(u.UI).NotTo(BeNil())
					default:
						Fail("wrong type")
					}
				})
			})

			Context("when vm state is something unexpected", func() {
				It("should return an error", func() {
					gomock.InOrder(
						mockDriver.EXPECT().VMExists("some-vm").Return(true, nil),
						mockDriver.EXPECT().GetVMIP("some-vm").Return("192.168.11.11", nil),
						mockDriver.EXPECT().GetMemory("some-vm").Return(uint64(3456), nil),
						mockDriver.EXPECT().GetHostForwardPort("some-vm", "ssh").Return("some-port", nil),
						mockDriver.EXPECT().VMState("some-vm").Return("some-unexpected-state", nil),
					)

					vm, err := builder.VM("some-vm")
					Expect(err).To(MatchError("failed to handle VM state 'some-unexpected-state'"))
					Expect(vm).To(BeNil())
				})
			})

			Context("when there is an error getting the vm memory", func() {
				It("should return an error", func() {
					gomock.InOrder(
						mockDriver.EXPECT().VMExists("some-vm").Return(true, nil),
						mockDriver.EXPECT().GetVMIP("some-vm").Return("192.168.11.11", nil),
						mockDriver.EXPECT().GetMemory("some-vm").Return(uint64(0), errors.New("some-error")),
					)

					vm, err := builder.VM("some-vm")
					Expect(err).To(MatchError("some-error"))
					Expect(vm).To(BeNil())
				})
			})
		})
	})
})

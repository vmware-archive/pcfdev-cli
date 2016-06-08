package vm_test

import (
	"errors"

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
			mockSystem *mocks.MockSystem
			builder    vm.Builder
		)

		BeforeEach(func() {
			mockCtrl = gomock.NewController(GinkgoT())
			mockDriver = mocks.NewMockDriver(mockCtrl)
			mockSystem = mocks.NewMockSystem(mockCtrl)

			builder = &vm.VBoxBuilder{
				Driver: mockDriver,
				Config: &config.Config{
					MinMemory: 100,
					MaxMemory: 200,
				},
				System: mockSystem,
			}
		})

		AfterEach(func() {
			mockCtrl.Finish()
		})

		Context("when vm is not created", func() {
			Context("when desired memory is provided and desired cores are provided", func() {
				It("should use the desired config", func() {
					mockDriver.EXPECT().VMExists("some-vm").Return(false, nil)
					notCreatedVM, err := builder.VM("some-vm", &config.VMConfig{
						Memory: uint64(3072),
					})
					Expect(err).NotTo(HaveOccurred())

					switch u := notCreatedVM.(type) {
					case *vm.NotCreated:
						Expect(u.Name).To(Equal("some-vm"))
						Expect(u.GetConfig().Memory).To(Equal(uint64(3072)))
					default:
						Fail("wrong type")
					}
				})
			})

			Context("when desired memory is not provided", func() {
				Context("when the system has total/2 memory more than the minimum amount but less than the maximum", func() {
					It("should give the VM half the total memory", func() {
						gomock.InOrder(
							mockDriver.EXPECT().VMExists("some-vm").Return(false, nil),
							mockSystem.EXPECT().TotalMemory().Return(uint64(300), nil),
						)

						notCreatedVM, err := builder.VM("some-vm", &config.VMConfig{})
						Expect(err).NotTo(HaveOccurred())

						switch u := notCreatedVM.(type) {
						case *vm.NotCreated:
							Expect(u.Name).To(Equal("some-vm"))
							Expect(u.GetConfig().Memory).To(Equal(uint64(150)))
						default:
							Fail("wrong type")
						}
					})

				})

				Context("when the system has total/2 memory less than the minimum amount", func() {
					It("should give the VM the minimum amount of memory", func() {
						gomock.InOrder(
							mockDriver.EXPECT().VMExists("some-vm").Return(false, nil),
							mockSystem.EXPECT().TotalMemory().Return(uint64(100), nil),
						)

						notCreatedVM, err := builder.VM("some-vm", &config.VMConfig{})
						Expect(err).NotTo(HaveOccurred())

						switch u := notCreatedVM.(type) {
						case *vm.NotCreated:
							Expect(u.Name).To(Equal("some-vm"))
							Expect(u.GetConfig().Memory).To(Equal(uint64(100)))
						default:
							Fail("wrong type")
						}
					})
				})

				Context("when the system has total/2 memory more than the maximum amount", func() {
					It("should give the VM the maximum amount of memory", func() {
						gomock.InOrder(
							mockDriver.EXPECT().VMExists("some-vm").Return(false, nil),
							mockSystem.EXPECT().TotalMemory().Return(uint64(500), nil),
						)

						notCreatedVM, err := builder.VM("some-vm", &config.VMConfig{})
						Expect(err).NotTo(HaveOccurred())

						switch u := notCreatedVM.(type) {
						case *vm.NotCreated:
							Expect(u.Name).To(Equal("some-vm"))
							Expect(u.GetConfig().Memory).To(Equal(uint64(200)))
						default:
							Fail("wrong type")
						}
					})
				})

				Context("when there is an error getting the amount of total memory", func() {
					It("should return the error", func() {
						gomock.InOrder(
							mockDriver.EXPECT().VMExists("some-vm").Return(false, nil),
							mockSystem.EXPECT().TotalMemory().Return(uint64(250), errors.New("some-error")),
						)

						notCreatedVM, err := builder.VM("some-vm", &config.VMConfig{})
						Expect(err).To(MatchError("some-error"))
						Expect(notCreatedVM).To(BeNil())
					})
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

					notCreatedVM, err := builder.VM("some-vm", &config.VMConfig{})
					Expect(err).NotTo(HaveOccurred())

					switch u := notCreatedVM.(type) {
					case *vm.Stopped:
						Expect(u.Name).To(Equal("some-vm"))
						Expect(u.IP).To(Equal("192.168.11.11"))
						Expect(u.GetConfig().Memory).To(Equal(uint64(3456)))
						Expect(u.SSHPort).To(Equal("some-port"))
						Expect(u.Domain).To(Equal("local.pcfdev.io"))
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
					_, err := builder.VM("some-vm", &config.VMConfig{})
					Expect(err).To(MatchError("some-error"))
				})
			})

			Context("when there is an error getting the vm IP", func() {
				It("should return an error", func() {
					gomock.InOrder(
						mockDriver.EXPECT().VMExists("some-vm").Return(true, nil),
						mockDriver.EXPECT().GetVMIP("some-vm").Return("", errors.New("some-error")),
					)

					_, err := builder.VM("some-vm", &config.VMConfig{})
					Expect(err).To(MatchError("some-error"))
				})
			})

			Context("when there is an error getting domain for vm ip", func() {
				It("should return an error", func() {
					gomock.InOrder(
						mockDriver.EXPECT().VMExists("some-vm").Return(true, nil),
						mockDriver.EXPECT().GetVMIP("some-vm").Return("some-ip", nil),
					)

					_, err := builder.VM("some-vm", &config.VMConfig{})
					Expect(err).To(MatchError("some-ip is not one of the allowed PCF Dev ips"))
				})
			})

			Context("when there is an error getting vm host forward port", func() {
				It("should return an error", func() {
					gomock.InOrder(
						mockDriver.EXPECT().VMExists("some-vm").Return(true, nil),
						mockDriver.EXPECT().GetVMIP("some-vm").Return("192.168.11.11", nil),
						mockDriver.EXPECT().GetMemory("some-vm").Return(uint64(3456), nil),
						mockDriver.EXPECT().GetHostForwardPort("some-vm", "ssh").Return("", errors.New("some-error")),
					)

					_, err := builder.VM("some-vm", &config.VMConfig{})
					Expect(err).To(MatchError("some-error"))
				})
			})

			Context("when vm is running", func() {
				It("should return a running vm", func() {
					gomock.InOrder(
						mockDriver.EXPECT().VMExists("some-vm").Return(true, nil),
						mockDriver.EXPECT().GetVMIP("some-vm").Return("192.168.11.11", nil),
						mockDriver.EXPECT().GetMemory("some-vm").Return(uint64(3456), nil),
						mockDriver.EXPECT().GetHostForwardPort("some-vm", "ssh").Return("some-port", nil),
						mockDriver.EXPECT().VMState("some-vm").Return(vbox.StateRunning, nil),
					)

					notCreatedVM, err := builder.VM("some-vm", &config.VMConfig{})
					Expect(err).NotTo(HaveOccurred())

					switch u := notCreatedVM.(type) {
					case *vm.Running:
						Expect(u.Name).To(Equal("some-vm"))
						Expect(u.IP).To(Equal("192.168.11.11"))
						Expect(u.GetConfig().Memory).To(Equal(uint64(3456)))
						Expect(u.SSHPort).To(Equal("some-port"))
						Expect(u.Domain).To(Equal("local.pcfdev.io"))
						Expect(u.VBox).NotTo(BeNil())
						Expect(u.UI).NotTo(BeNil())
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

					suspendedVM, err := builder.VM("some-vm", &config.VMConfig{})
					Expect(err).NotTo(HaveOccurred())

					switch u := suspendedVM.(type) {
					case *vm.Suspended:
						Expect(u.Name).To(Equal("some-vm"))
						Expect(u.IP).To(Equal("192.168.11.11"))
						Expect(u.GetConfig().Memory).To(Equal(uint64(3456)))
						Expect(u.SSHPort).To(Equal("some-port"))
						Expect(u.Domain).To(Equal("local.pcfdev.io"))
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

					vm, err := builder.VM("some-vm", &config.VMConfig{})
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

					vm, err := builder.VM("some-vm", &config.VMConfig{})
					Expect(err).To(MatchError("some-error"))
					Expect(vm).To(BeNil())
				})
			})
		})
	})
})

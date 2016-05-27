package vm_test

import (
	"errors"

	"github.com/golang/mock/gomock"
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
			builder    vm.Builder
		)

		BeforeEach(func() {
			mockCtrl = gomock.NewController(GinkgoT())
			mockDriver = mocks.NewMockDriver(mockCtrl)

			builder = &vm.VBoxBuilder{
				Driver: mockDriver,
			}
		})

		AfterEach(func() {
			mockCtrl.Finish()
		})

		Context("when vm is not created", func() {
			It("should return a not created vm", func() {
				mockDriver.EXPECT().VMExists("some-vm").Return(false, nil)

				notCreatedVM, err := builder.VM("some-vm")
				Expect(err).NotTo(HaveOccurred())

				switch u := notCreatedVM.(type) {
				case *vm.NotCreated:
					Expect(u.Name).To(Equal("some-vm"))
				default:
					Fail("wrong type")
				}
			})
		})

		Context("when vm is created", func() {
			Context("when vm is not running", func() {
				It("should return a stopped vm", func() {
					gomock.InOrder(
						mockDriver.EXPECT().VMExists("some-vm").Return(true, nil),
						mockDriver.EXPECT().GetVMIP("some-vm").Return("192.168.11.11", nil),
						mockDriver.EXPECT().GetHostForwardPort("some-vm", "ssh").Return("some-port", nil),
						mockDriver.EXPECT().IsVMRunning("some-vm").Return(false),
					)

					notCreatedVM, err := builder.VM("some-vm")
					Expect(err).NotTo(HaveOccurred())

					switch u := notCreatedVM.(type) {
					case *vm.Stopped:
						Expect(u.Name).To(Equal("some-vm"))
						Expect(u.IP).To(Equal("192.168.11.11"))
						Expect(u.SSHPort).To(Equal("some-port"))
						Expect(u.Domain).To(Equal("local.pcfdev.io"))
						Expect(u.SSH).NotTo(BeNil())
						Expect(u.VBox).NotTo(BeNil())
						Expect(u.UI).NotTo(BeNil())
						Expect(u.RequirementsChecker).NotTo(BeNil())
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
				It("should return an error", func() {
					gomock.InOrder(
						mockDriver.EXPECT().VMExists("some-vm").Return(true, nil),
						mockDriver.EXPECT().GetVMIP("some-vm").Return("", errors.New("some-error")),
					)

					_, err := builder.VM("some-vm")
					Expect(err).To(MatchError("some-error"))
				})
			})
			Context("when there is an error getting domain for vm ip", func() {
				It("should return an error", func() {
					gomock.InOrder(
						mockDriver.EXPECT().VMExists("some-vm").Return(true, nil),
						mockDriver.EXPECT().GetVMIP("some-vm").Return("some-ip", nil),
					)

					_, err := builder.VM("some-vm")
					Expect(err).To(MatchError("some-ip is not one of the allowed PCF Dev ips"))
				})
			})
			Context("when there is an error getting vm host forward port", func() {
				It("should return an error", func() {
					gomock.InOrder(
						mockDriver.EXPECT().VMExists("some-vm").Return(true, nil),
						mockDriver.EXPECT().GetVMIP("some-vm").Return("192.168.11.11", nil),
						mockDriver.EXPECT().GetHostForwardPort("some-vm", "ssh").Return("", errors.New("some-error")),
					)

					_, err := builder.VM("some-vm")
					Expect(err).To(MatchError("some-error"))
				})
			})
			Context("when vm is running", func() {
				It("should return a running vm", func() {
					gomock.InOrder(
						mockDriver.EXPECT().VMExists("some-vm").Return(true, nil),
						mockDriver.EXPECT().GetVMIP("some-vm").Return("192.168.11.11", nil),
						mockDriver.EXPECT().GetHostForwardPort("some-vm", "ssh").Return("some-port", nil),
						mockDriver.EXPECT().IsVMRunning("some-vm").Return(true),
					)

					notCreatedVM, err := builder.VM("some-vm")
					Expect(err).NotTo(HaveOccurred())

					switch u := notCreatedVM.(type) {
					case *vm.Running:
						Expect(u.Name).To(Equal("some-vm"))
						Expect(u.IP).To(Equal("192.168.11.11"))
						Expect(u.SSHPort).To(Equal("some-port"))
						Expect(u.Domain).To(Equal("local.pcfdev.io"))
						Expect(u.VBox).NotTo(BeNil())
						Expect(u.UI).NotTo(BeNil())
					default:
						Fail("wrong type")
					}
				})
			})
		})
	})
})

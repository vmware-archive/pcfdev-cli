package vbox_test

import (
	"errors"

	"github.com/pivotal-cf/pcfdev-cli/vbox"
	"github.com/pivotal-cf/pcfdev-cli/vbox/mocks"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("vbox", func() {
	var (
		mockCtrl   *gomock.Controller
		mockDriver *mocks.MockDriver
		mockSSH    *mocks.MockSSH
		vbx        *vbox.VBox
		path       string
		name       string
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockDriver = mocks.NewMockDriver(mockCtrl)
		mockSSH = mocks.NewMockSSH(mockCtrl)
		vbx = &vbox.VBox{
			Driver: mockDriver,
			SSH:    mockSSH,
		}
		path = "some-path"
		name = "some-vm"
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("ImportVM", func() {
		var path string
		var name string

		BeforeEach(func() {
			path = "some-path"
			name = "some-vm"
		})
		It("should import the VM", func() {
			gomock.InOrder(
				mockSSH.EXPECT().FreePort().Return("1234", nil),
				mockDriver.EXPECT().VBoxManage("import", path),
				mockDriver.EXPECT().CreateHostOnlyInterface("192.168.11.1").Return("vboxnet1", nil),
				mockDriver.EXPECT().AttachNetworkInterface("vboxnet1", "some-vm"),
				mockDriver.EXPECT().ForwardPort("some-vm", "ssh", "22", "1234"),
			)
			err := vbx.ImportVM(path, name)
			Expect(err).NotTo(HaveOccurred())
		})
		Context("fail to aquire random host port", func() {
			It("should return an error", func() {
				mockSSH.EXPECT().FreePort().Return("", errors.New("some-error"))

				err := vbx.ImportVM(path, name)
				Expect(err.Error()).To(Equal("failed to aquire random port: some-error"))
			})
		})
		Context("VM fails to import", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockSSH.EXPECT().FreePort().Return("1234", nil),
					mockDriver.EXPECT().VBoxManage("import", path).Return(nil, errors.New("some-error")),
				)
				err := vbx.ImportVM(path, name)
				Expect(err.Error()).To(Equal("failed to import ova: some-error"))
			})
		})
		Context("Creation of host only interface fails", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockSSH.EXPECT().FreePort().Return("1234", nil),
					mockDriver.EXPECT().VBoxManage("import", path),
					mockDriver.EXPECT().CreateHostOnlyInterface("192.168.11.1").Return("", errors.New("some-error")),
				)
				err := vbx.ImportVM(path, name)
				Expect(err.Error()).To(Equal("failed to create host only interface: some-error"))
			})
		})
		Context("fails to attache interface", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockSSH.EXPECT().FreePort().Return("1234", nil),
					mockDriver.EXPECT().VBoxManage("import", path),
					mockDriver.EXPECT().CreateHostOnlyInterface("192.168.11.1").Return("vboxnet1", nil),
					mockDriver.EXPECT().AttachNetworkInterface("vboxnet1", "some-vm").Return(errors.New("some-error")),
				)
				err := vbx.ImportVM(path, name)
				Expect(err.Error()).To(Equal("failed to attach interface: some-error"))
			})
		})
		Context("Port fowarding fails", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockSSH.EXPECT().FreePort().Return("1234", nil),
					mockDriver.EXPECT().VBoxManage("import", path),
					mockDriver.EXPECT().CreateHostOnlyInterface("192.168.11.1").Return("vboxnet1", nil),
					mockDriver.EXPECT().AttachNetworkInterface("vboxnet1", "some-vm"),
					mockDriver.EXPECT().ForwardPort("some-vm", "ssh", "22", "1234").Return(errors.New("some-error")),
				)
				err := vbx.ImportVM(path, name)
				Expect(err.Error()).To(Equal("failed to forward ssh port: some-error"))
			})
		})
	})
	Describe("StartVM", func() {
		Context("VM is already imported", func() {
			It("starts without reimporting", func() {
				gomock.InOrder(
					mockDriver.EXPECT().GetHostForwardPort("some-vm", "ssh").Return("5678", nil),
					mockDriver.EXPECT().StartVM("some-vm"),
					mockSSH.EXPECT().RunSSHCommand("echo -e \"auto eth1\niface eth1 inet static\naddress 192.168.11.11\nnetmask 255.255.255.0\" | sudo tee -a /etc/network/interfaces", "5678"),
					mockDriver.EXPECT().StopVM("some-vm"),
					mockDriver.EXPECT().StartVM("some-vm"),
				)
				vm, err := vbx.StartVM(name)
				Expect(err).NotTo(HaveOccurred())
				Expect(vm.Name).To(Equal("some-vm"))
				Expect(vm.SSHPort).To(Equal("5678"))
			})
			Context("fails so get forward port", func() {
				It("should return an error", func() {
					mockDriver.EXPECT().GetHostForwardPort("some-vm", "ssh").Return("", errors.New("some-error"))

					vm, err := vbx.StartVM(name)
					Expect(vm).To(BeNil())
					Expect(err.Error()).To(Equal("failed to get host port for ssh forwarding: some-error"))
				})
			})
			Context("VM fails to start", func() {
				It("should return an error", func() {
					gomock.InOrder(
						mockDriver.EXPECT().GetHostForwardPort("some-vm", "ssh").Return("5678", nil),
						mockDriver.EXPECT().StartVM("some-vm").Return(errors.New("some-error")),
					)

					_, err := vbx.StartVM(name)
					Expect(err.Error()).To(Equal("failed to start vm: some-error"))
				})
			})
			Context("SSH Command to set static ip fails", func() {
				It("should return an error", func() {
					gomock.InOrder(
						mockDriver.EXPECT().GetHostForwardPort("some-vm", "ssh").Return("5678", nil),
						mockDriver.EXPECT().StartVM("some-vm"),
						mockSSH.EXPECT().RunSSHCommand("echo -e \"auto eth1\niface eth1 inet static\naddress 192.168.11.11\nnetmask 255.255.255.0\" | sudo tee -a /etc/network/interfaces", "5678").Return(errors.New("some-error")),
					)
					_, err := vbx.StartVM(name)
					Expect(err.Error()).To(Equal("failed to set static ip: some-error"))
				})
			})
			Context("VM fails to stop", func() {
				It("should return an error", func() {
					gomock.InOrder(
						mockDriver.EXPECT().GetHostForwardPort("some-vm", "ssh").Return("5678", nil),
						mockDriver.EXPECT().StartVM("some-vm"),
						mockSSH.EXPECT().RunSSHCommand("echo -e \"auto eth1\niface eth1 inet static\naddress 192.168.11.11\nnetmask 255.255.255.0\" | sudo tee -a /etc/network/interfaces", "5678"),
						mockDriver.EXPECT().StopVM("some-vm").Return(errors.New("some-error")),
					)
					_, err := vbx.StartVM(name)
					Expect(err.Error()).To(Equal("failed to stop vm: some-error"))
				})
			})
		})
	})
	Describe("StopVM", func() {
		It("should stop the VM", func() {
			mockDriver.EXPECT().StopVM("some-vm")

			err := vbx.StopVM("some-vm")
			Expect(err).NotTo(HaveOccurred())
		})
		Context("Driver fails to stop VM", func() {
			It("should return the error", func() {
				ExpectedError := errors.New("some-error")

				mockDriver.EXPECT().StopVM("some-vm").Return(ExpectedError)
				err := vbx.StopVM("some-vm")
				Expect(err).To(Equal(ExpectedError))
			})
		})
	})
	Describe("DestroyVM", func() {
		It("should stop the VM", func() {
			mockDriver.EXPECT().DestroyVM("some-vm")

			err := vbx.DestroyVM("some-vm")
			Expect(err).NotTo(HaveOccurred())
		})
		Context("Driver fails to stop VM", func() {
			It("should return the error", func() {
				ExpectedError := errors.New("some-error")

				mockDriver.EXPECT().DestroyVM("some-vm").Return(ExpectedError)
				err := vbx.DestroyVM("some-vm")
				Expect(err).To(Equal(ExpectedError))
			})
		})
	})
	Describe("IsVMRunning", func() {
		Context("VM is running", func() {
			It("should return true", func() {
				mockDriver.EXPECT().IsVMRunning("some-vm").Return(true)

				running := vbx.IsVMRunning("some-vm")
				Expect(running).To(BeTrue())
			})
		})
		Context("VM is not running", func() {
			It("should return false", func() {
				mockDriver.EXPECT().IsVMRunning("some-vm").Return(false)

				running := vbx.IsVMRunning("some-vm")
				Expect(running).To(BeFalse())
			})
		})
	})
	Describe("IsVMImported", func() {
		Context("VM exists", func() {
			It("should return true", func() {
				mockDriver.EXPECT().VMExists("some-vm").Return(true, nil)

				imported, err := vbx.IsVMImported("some-vm")
				Expect(err).NotTo(HaveOccurred())
				Expect(imported).To(BeTrue())
			})
		})
		Context("VM does not exist", func() {
			It("should return false", func() {
				mockDriver.EXPECT().VMExists("some-vm").Return(false, nil)

				imported, err := vbx.IsVMImported("some-vm")
				Expect(err).NotTo(HaveOccurred())
				Expect(imported).To(BeFalse())
			})
		})
		Context("VM query fails", func() {
			It("should return error", func() {
				mockDriver.EXPECT().VMExists("some-vm").Return(false, errors.New("some-error"))

				_, err := vbx.IsVMImported("some-vm")
				Expect(err.Error()).To(Equal("failed to query for VM: some-error"))
			})
		})
	})
})

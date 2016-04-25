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
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockDriver = mocks.NewMockDriver(mockCtrl)
		mockSSH = mocks.NewMockSSH(mockCtrl)
		vbx = &vbox.VBox{
			Driver: mockDriver,
			SSH:    mockSSH,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("ImportVM", func() {
		It("should import the VM", func() {
			gomock.InOrder(
				mockSSH.EXPECT().GenerateAddress().Return("some-host", "1234", nil),
				mockDriver.EXPECT().VBoxManage("import", "some-path"),
				mockDriver.EXPECT().CreateHostOnlyInterface("192.168.11.1").Return("vboxnet1", nil),
				mockDriver.EXPECT().AttachNetworkInterface("vboxnet1", "some-vm"),
				mockDriver.EXPECT().ForwardPort("some-vm", "ssh", "22", "1234"),
			)
			err := vbx.ImportVM("some-path", "some-vm")
			Expect(err).NotTo(HaveOccurred())
		})

		Context("fail to acquire random host port", func() {
			It("should return an error", func() {
				mockSSH.EXPECT().GenerateAddress().Return("", "", errors.New("some-error"))

				err := vbx.ImportVM("some-path", "some-vm")
				Expect(err).To(MatchError("some-error"))
			})
		})

		Context("VM fails to import", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockSSH.EXPECT().GenerateAddress().Return("some-host", "1234", nil),
					mockDriver.EXPECT().VBoxManage("import", "some-path").Return(nil, errors.New("some-error")),
				)
				err := vbx.ImportVM("some-path", "some-vm")
				Expect(err).To(MatchError("some-error"))
			})
		})

		Context("Creation of host only interface fails", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockSSH.EXPECT().GenerateAddress().Return("some-host", "1234", nil),
					mockDriver.EXPECT().VBoxManage("import", "some-path"),
					mockDriver.EXPECT().CreateHostOnlyInterface("192.168.11.1").Return("", errors.New("some-error")),
				)
				err := vbx.ImportVM("some-path", "some-vm")
				Expect(err).To(MatchError("some-error"))
			})
		})

		Context("fails to attach interface", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockSSH.EXPECT().GenerateAddress().Return("some-host", "1234", nil),
					mockDriver.EXPECT().VBoxManage("import", "some-path"),
					mockDriver.EXPECT().CreateHostOnlyInterface("192.168.11.1").Return("vboxnet1", nil),
					mockDriver.EXPECT().AttachNetworkInterface("vboxnet1", "some-vm").Return(errors.New("some-error")),
				)
				err := vbx.ImportVM("some-path", "some-vm")
				Expect(err).To(MatchError("some-error"))
			})
		})

		Context("Port fowarding fails", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockSSH.EXPECT().GenerateAddress().Return("some-host", "1234", nil),
					mockDriver.EXPECT().VBoxManage("import", "some-path"),
					mockDriver.EXPECT().CreateHostOnlyInterface("192.168.11.1").Return("vboxnet1", nil),
					mockDriver.EXPECT().AttachNetworkInterface("vboxnet1", "some-vm"),
					mockDriver.EXPECT().ForwardPort("some-vm", "ssh", "22", "1234").Return(errors.New("some-error")),
				)
				err := vbx.ImportVM("some-path", "some-vm")
				Expect(err).To(MatchError("some-error"))
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
				vm, err := vbx.StartVM("some-vm")
				Expect(err).NotTo(HaveOccurred())
				Expect(vm.Name).To(Equal("some-vm"))
				Expect(vm.SSHPort).To(Equal("5678"))
			})

			Context("fails so get forward port", func() {
				It("should return an error", func() {
					mockDriver.EXPECT().GetHostForwardPort("some-vm", "ssh").Return("", errors.New("some-error"))

					vm, err := vbx.StartVM("some-vm")
					Expect(vm).To(BeNil())
					Expect(err).To(MatchError("some-error"))
				})
			})

			Context("VM fails to start", func() {
				It("should return an error", func() {
					gomock.InOrder(
						mockDriver.EXPECT().GetHostForwardPort("some-vm", "ssh").Return("5678", nil),
						mockDriver.EXPECT().StartVM("some-vm").Return(errors.New("some-error")),
					)

					_, err := vbx.StartVM("some-vm")
					Expect(err).To(MatchError("some-error"))
				})
			})

			Context("SSH Command to set static ip fails", func() {
				It("should return an error", func() {
					gomock.InOrder(
						mockDriver.EXPECT().GetHostForwardPort("some-vm", "ssh").Return("5678", nil),
						mockDriver.EXPECT().StartVM("some-vm"),
						mockSSH.EXPECT().RunSSHCommand("echo -e \"auto eth1\niface eth1 inet static\naddress 192.168.11.11\nnetmask 255.255.255.0\" | sudo tee -a /etc/network/interfaces", "5678").Return(errors.New("some-error")),
					)
					_, err := vbx.StartVM("some-vm")
					Expect(err).To(MatchError("some-error"))
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
					_, err := vbx.StartVM("some-vm")
					Expect(err).To(MatchError("some-error"))
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
				expectedError := errors.New("some-error")

				mockDriver.EXPECT().StopVM("some-vm").Return(expectedError)
				err := vbx.StopVM("some-vm")
				Expect(err).To(MatchError(expectedError))
			})
		})
	})

	Describe("DestroyVM", func() {
		Context("VM is stopped", func() {
			It("should stop the VM", func() {
				mockDriver.EXPECT().VMExists("some-vm").Return(true, nil)
				mockDriver.EXPECT().IsVMRunning("some-vm").Return(false)
				mockDriver.EXPECT().DestroyVM("some-vm")

				err := vbx.DestroyVM("some-vm")
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("VM is running", func() {
			It("should stop the VM", func() {
				mockDriver.EXPECT().VMExists("some-vm").Return(true, nil)
				mockDriver.EXPECT().IsVMRunning("some-vm").Return(true)
				mockDriver.EXPECT().StopVM("some-vm")
				mockDriver.EXPECT().DestroyVM("some-vm")

				err := vbx.DestroyVM("some-vm")
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("Driver fails to stop VM", func() {
			It("should return the error", func() {
				mockDriver.EXPECT().VMExists("some-vm").Return(true, nil)
				mockDriver.EXPECT().IsVMRunning("some-vm").Return(true)

				expectedError := errors.New("some-error")
				mockDriver.EXPECT().StopVM("some-vm").Return(expectedError)
				err := vbx.DestroyVM("some-vm")
				Expect(err).To(MatchError(expectedError))
			})
		})

		Context("Driver fails to destroy VM", func() {
			It("should return the error", func() {
				mockDriver.EXPECT().VMExists("some-vm").Return(true, nil)
				mockDriver.EXPECT().IsVMRunning("some-vm").Return(true)
				mockDriver.EXPECT().StopVM("some-vm")

				expectedError := errors.New("some-error")
				mockDriver.EXPECT().DestroyVM("some-vm").Return(expectedError)
				err := vbx.DestroyVM("some-vm")
				Expect(err).To(MatchError(expectedError))
			})
		})
	})

	Describe("#Status", func() {
		Context("VM is running", func() {
			It("should return running", func() {
				mockDriver.EXPECT().VMExists("some-vm").Return(true, nil)
				mockDriver.EXPECT().IsVMRunning("some-vm").Return(true)

				status, err := vbx.Status("some-vm")
				Expect(err).NotTo(HaveOccurred())
				Expect(status).To(Equal("Running"))
			})
		})

		Context("VM is not created", func() {
			It("should return not created", func() {
				mockDriver.EXPECT().VMExists("some-vm").Return(false, nil)

				status, err := vbx.Status("some-vm")
				Expect(err).NotTo(HaveOccurred())
				Expect(status).To(Equal("Not created"))
			})
		})

		Context("VM is stopped", func() {
			It("should return stopped", func() {
				mockDriver.EXPECT().VMExists("some-vm").Return(true, nil)
				mockDriver.EXPECT().IsVMRunning("some-vm").Return(false)

				status, err := vbx.Status("some-vm")
				Expect(err).NotTo(HaveOccurred())
				Expect(status).To(Equal("Stopped"))
			})
		})

		Context("An error checking the VM Status", func() {
			It("should return an error", func() {
				someError := errors.New("some-error")
				mockDriver.EXPECT().VMExists("some-vm").Return(false, someError)

				_, err := vbx.Status("some-vm")
				Expect(err).To(MatchError(someError))
			})
		})
	})
})

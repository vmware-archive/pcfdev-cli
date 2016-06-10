package vbox_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/network"
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
		mockPicker *mocks.MockNetworkPicker
		vbx        *vbox.VBox
		conf       *config.Config
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockDriver = mocks.NewMockDriver(mockCtrl)
		mockSSH = mocks.NewMockSSH(mockCtrl)
		mockPicker = mocks.NewMockNetworkPicker(mockCtrl)

		conf = &config.Config{
			PCFDevHome: "some-pcfdev-home",
			OVADir:     "some-ova-dir",
			HTTPProxy:  "some-http-proxy",
			HTTPSProxy: "some-https-proxy",
			NoProxy:    "some-no-proxy",

			MinMemory: uint64(1000),
			MaxMemory: uint64(2000),
		}

		vbx = &vbox.VBox{
			Driver: mockDriver,
			SSH:    mockSSH,
			Picker: mockPicker,
			Config: conf,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("#ImportVM", func() {
		Context("when it selects an existing interface", func() {
			It("should attach that interface", func() {
				iface := &network.Interface{
					Name: "some-interface",
				}
				vboxnets := []*network.Interface{iface}

				gomock.InOrder(
					mockSSH.EXPECT().GenerateAddress().Return("some-host", "some-port", nil),
					mockDriver.EXPECT().GetVirtualSystemNumbersOfHardDiskImages(filepath.Join("some-ova-dir", "some-vm.ova")).Return([]string{"1"}, nil),
					mockDriver.EXPECT().VBoxManage("import", filepath.Join("some-ova-dir", "some-vm.ova"), "--vsys", "0", "--cpus", "7", "--unit", "1", "--disk", filepath.Join("some-pcfdev-home", "some-vm-disk0.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return(vboxnets, nil),
					mockPicker.EXPECT().SelectAvailableNetworkInterface(vboxnets).Return(iface, true, nil),
					mockDriver.EXPECT().AttachNetworkInterface("some-interface", "some-vm"),
					mockDriver.EXPECT().ForwardPort("some-vm", "ssh", "some-port", "22"),
					mockDriver.EXPECT().SetMemory("some-vm", uint64(2000)),
				)
				err := vbx.ImportVM("some-vm", &config.VMConfig{
					Memory: uint64(2000),
					CPUs:   7,
				})
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when it selects an interface that doesnt exist yet", func() {
			It("should create and attach that interface", func() {
				ip := "192.168.11.11"
				iface := &network.Interface{
					IP: ip,
				}
				vboxnets := []*network.Interface{iface}

				gomock.InOrder(
					mockSSH.EXPECT().GenerateAddress().Return("some-host", "some-port", nil),
					mockDriver.EXPECT().GetVirtualSystemNumbersOfHardDiskImages(filepath.Join("some-ova-dir", "some-vm.ova")).Return([]string{"1"}, nil),
					mockDriver.EXPECT().VBoxManage("import", filepath.Join("some-ova-dir", "some-vm.ova"), "--vsys", "0", "--cpus", "7", "--unit", "1", "--disk", filepath.Join("some-pcfdev-home", "some-vm-disk0.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return(vboxnets, nil),
					mockPicker.EXPECT().SelectAvailableNetworkInterface(vboxnets).Return(iface, false, nil),
					mockDriver.EXPECT().CreateHostOnlyInterface(ip).Return("some-interface", nil),
					mockDriver.EXPECT().AttachNetworkInterface("some-interface", "some-vm"),
					mockDriver.EXPECT().ForwardPort("some-vm", "ssh", "some-port", "22"),
					mockDriver.EXPECT().SetMemory("some-vm", uint64(2000)),
				)
				err := vbx.ImportVM("some-vm", &config.VMConfig{
					Memory: uint64(2000),
					CPUs:   7,
				})
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when setting the memory returns an error", func() {
			It("should return an error", func() {
				iface := &network.Interface{
					Name: "some-interface",
				}
				vboxnets := []*network.Interface{iface}
				gomock.InOrder(
					mockSSH.EXPECT().GenerateAddress().Return("some-host", "some-port", nil),
					mockDriver.EXPECT().GetVirtualSystemNumbersOfHardDiskImages(filepath.Join("some-ova-dir", "some-vm.ova")).Return([]string{"1"}, nil),
					mockDriver.EXPECT().VBoxManage("import", filepath.Join("some-ova-dir", "some-vm.ova"), "--vsys", "0", "--cpus", "7", "--unit", "1", "--disk", filepath.Join("some-pcfdev-home", "some-vm-disk0.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return(vboxnets, nil),
					mockPicker.EXPECT().SelectAvailableNetworkInterface(vboxnets).Return(iface, true, nil),
					mockDriver.EXPECT().AttachNetworkInterface("some-interface", "some-vm"),
					mockDriver.EXPECT().ForwardPort("some-vm", "ssh", "some-port", "22"),
					mockDriver.EXPECT().SetMemory("some-vm", uint64(2000)).Return(errors.New("some-error")),
				)
				Expect(vbx.ImportVM("some-vm", &config.VMConfig{
					Memory: uint64(2000),
					CPUs:   7,
				})).To(MatchError("some-error"))
			})
		})

		Context("when it fails to get vbox hostonly interfaces", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockSSH.EXPECT().GenerateAddress().Return("some-host", "some-port", nil),
					mockDriver.EXPECT().GetVirtualSystemNumbersOfHardDiskImages(filepath.Join("some-ova-dir", "some-vm.ova")).Return([]string{"1"}, nil),
					mockDriver.EXPECT().VBoxManage("import", filepath.Join("some-ova-dir", "some-vm.ova"), "--vsys", "0", "--cpus", "7", "--unit", "1", "--disk", filepath.Join("some-pcfdev-home", "some-vm-disk0.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return([]*network.Interface{}, errors.New("some-error")),
				)
				Expect(vbx.ImportVM("some-vm", &config.VMConfig{
					Memory: uint64(2000),
					CPUs:   7,
				})).To(MatchError("some-error"))
			})
		})

		Context("when it fails to get select available interface", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockSSH.EXPECT().GenerateAddress().Return("some-host", "some-port", nil),
					mockDriver.EXPECT().GetVirtualSystemNumbersOfHardDiskImages(filepath.Join("some-ova-dir", "some-vm.ova")).Return([]string{"1"}, nil),
					mockDriver.EXPECT().VBoxManage("import", filepath.Join("some-ova-dir", "some-vm.ova"), "--vsys", "0", "--cpus", "7", "--unit", "1", "--disk", filepath.Join("some-pcfdev-home", "some-vm-disk0.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return([]*network.Interface{}, nil),
					mockPicker.EXPECT().SelectAvailableNetworkInterface([]*network.Interface{}).Return(nil, false, errors.New("some-error")),
				)
				Expect(vbx.ImportVM("some-vm", &config.VMConfig{
					Memory: uint64(2000),
					CPUs:   7,
				})).To(MatchError("some-error"))
			})
		})

		Context("fail to acquire random host port", func() {
			It("should return an error", func() {
				mockSSH.EXPECT().GenerateAddress().Return("", "", errors.New("some-error"))

				Expect(vbx.ImportVM("some-vm", &config.VMConfig{
					Memory: uint64(2000),
					CPUs:   7,
				})).To(MatchError("some-error"))
			})
		})

		Context("VM fails to import", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockSSH.EXPECT().GenerateAddress().Return("some-host", "some-port", nil),
					mockDriver.EXPECT().GetVirtualSystemNumbersOfHardDiskImages(filepath.Join("some-ova-dir", "some-vm.ova")).Return([]string{"1"}, nil),
					mockDriver.EXPECT().VBoxManage("import", filepath.Join("some-ova-dir", "some-vm.ova"), "--vsys", "0", "--cpus", "7", "--unit", "1", "--disk", filepath.Join("some-pcfdev-home", "some-vm-disk0.vmdk")).Return(nil, errors.New("some-error")),
				)
				Expect(vbx.ImportVM("some-vm", &config.VMConfig{
					Memory: uint64(2000),
					CPUs:   7,
				})).To(MatchError("some-error"))
			})
		})

		Context("Creation of host only interface fails", func() {
			It("should return an error", func() {
				ip := "192.168.11.11"
				iface := &network.Interface{
					IP: ip,
				}
				gomock.InOrder(
					mockSSH.EXPECT().GenerateAddress().Return("some-host", "some-port", nil),
					mockDriver.EXPECT().GetVirtualSystemNumbersOfHardDiskImages(filepath.Join("some-ova-dir", "some-vm.ova")).Return([]string{"1"}, nil),
					mockDriver.EXPECT().VBoxManage("import", filepath.Join("some-ova-dir", "some-vm.ova"), "--vsys", "0", "--cpus", "7", "--unit", "1", "--disk", filepath.Join("some-pcfdev-home", "some-vm-disk0.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return([]*network.Interface{}, nil),
					mockPicker.EXPECT().SelectAvailableNetworkInterface([]*network.Interface{}).Return(iface, false, nil),
					mockDriver.EXPECT().CreateHostOnlyInterface(ip).Return("", errors.New("some-error")),
				)
				Expect(vbx.ImportVM("some-vm", &config.VMConfig{
					Memory: uint64(2000),
					CPUs:   7,
				})).To(MatchError("some-error"))
			})
		})

		Context("when attaching an interface fails", func() {
			It("should return an error", func() {
				iface := &network.Interface{
					Name: "some-interface",
				}
				gomock.InOrder(
					mockSSH.EXPECT().GenerateAddress().Return("some-host", "some-port", nil),
					mockDriver.EXPECT().GetVirtualSystemNumbersOfHardDiskImages(filepath.Join("some-ova-dir", "some-vm.ova")).Return([]string{"1"}, nil),
					mockDriver.EXPECT().VBoxManage("import", filepath.Join("some-ova-dir", "some-vm.ova"), "--vsys", "0", "--cpus", "7", "--unit", "1", "--disk", filepath.Join("some-pcfdev-home", "some-vm-disk0.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return([]*network.Interface{}, nil),
					mockPicker.EXPECT().SelectAvailableNetworkInterface([]*network.Interface{}).Return(iface, true, nil),
					mockDriver.EXPECT().AttachNetworkInterface("some-interface", "some-vm").Return(errors.New("some-error")),
				)
				Expect(vbx.ImportVM("some-vm", &config.VMConfig{
					Memory: uint64(2000),
					CPUs:   7,
				})).To(MatchError("some-error"))
			})
		})

		Context("when retriving the virtual system numbers of hard disk images fails", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockSSH.EXPECT().GenerateAddress().Return("some-host", "some-port", nil),
					mockDriver.EXPECT().GetVirtualSystemNumbersOfHardDiskImages(filepath.Join("some-ova-dir", "some-vm.ova")).Return(nil, errors.New("some-error")),
				)
				Expect(vbx.ImportVM("some-vm", &config.VMConfig{
					Memory: uint64(2000),
					CPUs:   7,
				})).To(MatchError("some-error"))
			})
		})

		Context("when port fowarding fails", func() {
			iface := &network.Interface{
				Name: "some-interface",
			}
			It("should return an error", func() {
				gomock.InOrder(
					mockSSH.EXPECT().GenerateAddress().Return("some-host", "some-port", nil),
					mockDriver.EXPECT().GetVirtualSystemNumbersOfHardDiskImages(filepath.Join("some-ova-dir", "some-vm.ova")).Return([]string{"1"}, nil),
					mockDriver.EXPECT().VBoxManage("import", filepath.Join("some-ova-dir", "some-vm.ova"), "--vsys", "0", "--cpus", "7", "--unit", "1", "--disk", filepath.Join("some-pcfdev-home", "some-vm-disk0.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return([]*network.Interface{}, nil),
					mockPicker.EXPECT().SelectAvailableNetworkInterface([]*network.Interface{}).Return(iface, true, nil),
					mockDriver.EXPECT().AttachNetworkInterface("some-interface", "some-vm"),
					mockDriver.EXPECT().ForwardPort("some-vm", "ssh", "some-port", "22").Return(errors.New("some-error")),
				)
				Expect(vbx.ImportVM("some-vm", &config.VMConfig{
					Memory: uint64(2000),
					CPUs:   7,
				})).To(MatchError("some-error"))
			})
		})
	})

	Describe("#StartVM", func() {
		Context("when VM is already imported", func() {
			It("starts without reimporting", func() {
				gomock.InOrder(
					mockDriver.EXPECT().StartVM("some-vm"),
					mockSSH.EXPECT().RunSSHCommand("echo -e \"auto eth1\niface eth1 inet static\naddress 192.168.22.11\nnetmask 255.255.255.0\" | sudo tee -a /etc/network/interfaces", "some-port", 2*time.Minute, ioutil.Discard, ioutil.Discard),
					mockSSH.EXPECT().RunSSHCommand("echo -e \""+
						"HTTP_PROXY=some-http-proxy\n"+
						"HTTPS_PROXY=some-https-proxy\n"+
						"NO_PROXY=localhost,127.0.0.1,192.168.22.1,192.168.22.11,local2.pcfdev.io,some-no-proxy\n"+
						"http_proxy=some-http-proxy\n"+
						"https_proxy=some-https-proxy\n"+
						"no_proxy=localhost,127.0.0.1,192.168.22.1,192.168.22.11,local2.pcfdev.io,some-no-proxy\" "+
						"| sudo tee -a /etc/environment",
						"some-port",
						2*time.Minute,
						ioutil.Discard,
						ioutil.Discard),
					mockDriver.EXPECT().StopVM("some-vm"),
					mockDriver.EXPECT().StartVM("some-vm"),
				)

				Expect(vbx.StartVM("some-vm", "192.168.22.11", "some-port", "some-domain")).To(Succeed())
			})

			It("translates 127.0.0.1 to subnetIP in proxy settings", func() {
				conf.HTTPProxy = "127.0.0.1"
				conf.HTTPSProxy = "127.0.0.1:8080"

				gomock.InOrder(
					mockDriver.EXPECT().StartVM("some-vm"),
					mockSSH.EXPECT().RunSSHCommand("echo -e \"auto eth1\niface eth1 inet static\naddress 192.168.22.11\nnetmask 255.255.255.0\" | sudo tee -a /etc/network/interfaces", "some-port", 2*time.Minute, ioutil.Discard, ioutil.Discard),
					mockSSH.EXPECT().RunSSHCommand("echo -e \""+
						"HTTP_PROXY=192.168.22.1\n"+
						"HTTPS_PROXY=192.168.22.1:8080\n"+
						"NO_PROXY=localhost,127.0.0.1,192.168.22.1,192.168.22.11,local2.pcfdev.io,some-no-proxy\n"+
						"http_proxy=192.168.22.1\n"+
						"https_proxy=192.168.22.1:8080\n"+
						"no_proxy=localhost,127.0.0.1,192.168.22.1,192.168.22.11,local2.pcfdev.io,some-no-proxy\" "+
						"| sudo tee -a /etc/environment",
						"some-port",
						2*time.Minute,
						ioutil.Discard,
						ioutil.Discard),
					mockDriver.EXPECT().StopVM("some-vm"),
					mockDriver.EXPECT().StartVM("some-vm"),
				)

				Expect(vbx.StartVM("some-vm", "192.168.22.11", "some-port", "some-domain")).To(Succeed())
			})

			Context("when a bad ip is passed to StartVM command", func() {
				It("should return an error", func() {
					gomock.InOrder(
						mockDriver.EXPECT().StartVM("some-vm"),
						mockSSH.EXPECT().RunSSHCommand("echo -e \"auto eth1\niface eth1 inet static\naddress some-bad-ip\nnetmask 255.255.255.0\" | sudo tee -a /etc/network/interfaces", "some-port", 2*time.Minute, ioutil.Discard, ioutil.Discard),
					)

					Expect(vbx.StartVM("some-vm", "some-bad-ip", "some-port", "some-domain")).To(MatchError("some-bad-ip is not one of the allowed PCF Dev ips"))
				})
			})

			Context("when VM fails to start", func() {
				It("should return an error", func() {
					gomock.InOrder(
						mockDriver.EXPECT().StartVM("some-vm").Return(errors.New("some-error")),
					)

					Expect(vbx.StartVM("some-vm", "192.168.22.11", "some-port", "some-domain")).To(MatchError("some-error"))
				})
			})

			Context("when SSH Command to set static ip fails", func() {
				It("should return an error", func() {
					gomock.InOrder(
						mockDriver.EXPECT().StartVM("some-vm"),
						mockSSH.EXPECT().RunSSHCommand(fmt.Sprintf("echo -e \"auto eth1\niface eth1 inet static\naddress 192.168.11.11\nnetmask 255.255.255.0\" | sudo tee -a /etc/network/interfaces"), "some-port", 2*time.Minute, ioutil.Discard, ioutil.Discard).Return(errors.New("some-error")),
					)

					Expect(vbx.StartVM("some-vm", "192.168.11.11", "some-port", "some-domain")).To(MatchError("some-error"))
				})
			})

			Context("when VM fails to stop", func() {
				It("should return an error", func() {
					ip := "192.168.11.11"
					gomock.InOrder(
						mockDriver.EXPECT().StartVM("some-vm"),
						mockSSH.EXPECT().RunSSHCommand(fmt.Sprintf("echo -e \"auto eth1\niface eth1 inet static\naddress %s\nnetmask 255.255.255.0\" | sudo tee -a /etc/network/interfaces", ip), "some-port", 2*time.Minute, ioutil.Discard, ioutil.Discard),
						mockSSH.EXPECT().RunSSHCommand("echo -e \""+
							"HTTP_PROXY=some-http-proxy\n"+
							"HTTPS_PROXY=some-https-proxy\n"+
							"NO_PROXY=localhost,127.0.0.1,192.168.11.1,192.168.11.11,local.pcfdev.io,some-no-proxy\n"+
							"http_proxy=some-http-proxy\n"+
							"https_proxy=some-https-proxy\n"+
							"no_proxy=localhost,127.0.0.1,192.168.11.1,192.168.11.11,local.pcfdev.io,some-no-proxy\" "+
							"| sudo tee -a /etc/environment",
							"some-port",
							2*time.Minute,
							ioutil.Discard,
							ioutil.Discard),
						mockDriver.EXPECT().StopVM("some-vm").Return(errors.New("some-error")),
					)

					Expect(vbx.StartVM("some-vm", "192.168.11.11", "some-port", "some-domain")).To(MatchError("some-error"))
				})
			})
		})
	})

	Describe("#StopVM", func() {
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

	Describe("#SuspendVM", func() {
		It("should suspend the VM", func() {
			mockDriver.EXPECT().SuspendVM("some-vm")

			err := vbx.SuspendVM("some-vm")
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when the Driver fails to suspend the VM", func() {
			It("should return the error", func() {
				expectedError := errors.New("some-error")

				mockDriver.EXPECT().SuspendVM("some-vm").Return(expectedError)
				err := vbx.SuspendVM("some-vm")
				Expect(err).To(MatchError(expectedError))
			})
		})
	})

	Describe("#ResumeVM", func() {
		It("should resume the VM", func() {
			mockDriver.EXPECT().ResumeVM("some-vm")

			err := vbx.ResumeVM("some-vm")
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when the Driver fails to resume the VM", func() {
			It("should return the error", func() {
				expectedError := errors.New("some-error")

				mockDriver.EXPECT().ResumeVM("some-vm").Return(expectedError)
				err := vbx.ResumeVM("some-vm")
				Expect(err).To(MatchError(expectedError))
			})
		})
	})

	Describe("#ConflictingVMPresent", func() {
		Context("when there are no conflicting VMs with the prefix pcfdev-", func() {
			It("should return false", func() {
				mockDriver.EXPECT().RunningVMs().Return([]string{"some-other-vm", "pcfdev-our-vm"}, nil)
				Expect(vbx.ConflictingVMPresent("pcfdev-our-vm")).To(BeFalse())
			})
		})

		Context("when there are conflicting VMs with the prefix pcfdev- running", func() {
			It("should return true", func() {
				mockDriver.EXPECT().RunningVMs().Return([]string{"pcfdev-conflicting-vm", "pcfdev-our-vm"}, nil)
				Expect(vbx.ConflictingVMPresent("pcfdev-our-vm")).To(BeTrue())
			})
		})

		Context("when getting running vms returns an error", func() {
			It("should return an error", func() {
				mockDriver.EXPECT().RunningVMs().Return(nil, errors.New("some-error"))
				_, err := vbx.ConflictingVMPresent("pcfdev-our-vm")
				Expect(err).To(MatchError("some-error"))
			})
		})
	})

	Describe("#Destroy", func() {
		It("should destroy the VM", func() {
			mockDriver.EXPECT().DestroyVM("some-vm")

			Expect(vbx.DestroyVM("some-vm")).To(Succeed())
		})

		Context("when the driver fails to destroy VM", func() {
			It("should return the error", func() {
				mockDriver.EXPECT().DestroyVM("some-vm").Return(errors.New("some-error"))

				Expect(vbx.DestroyVM("some-vm")).To(MatchError("some-error"))
			})
		})
	})

	Describe("#PowerOffVM", func() {
		It("should power off the VM", func() {
			mockDriver.EXPECT().PowerOffVM("some-vm")

			Expect(vbx.PowerOffVM("some-vm")).To(Succeed())
		})

		Context("when the driver fails to power off the VM", func() {
			It("should return the error", func() {
				mockDriver.EXPECT().PowerOffVM("some-vm").Return(errors.New("some-error"))

				Expect(vbx.PowerOffVM("some-vm")).To(MatchError("some-error"))
			})
		})
	})

	Describe("#GetPCFDevVMs", func() {
		It("should return VM names that begin with pcfdev-", func() {
			mockDriver.EXPECT().VMs().Return([]string{"pcfdev-0.0.0", "pcfdev-0.0.1", "some-bad-vm-name"}, nil)

			Expect(vbx.GetPCFDevVMs()).To(Equal([]string{"pcfdev-0.0.0", "pcfdev-0.0.1"}))
		})

		Context("when getting VMs fails", func() {
			It("should return an error", func() {
				mockDriver.EXPECT().VMs().Return([]string{}, errors.New("some-error"))

				_, err := vbx.GetPCFDevVMs()
				Expect(err).To(MatchError("some-error"))
			})
		})
	})
})

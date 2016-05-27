package vbox_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"

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
		mockSystem *mocks.MockSystem
		mockConfig *mocks.MockConfig
		vbx        *vbox.VBox
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockDriver = mocks.NewMockDriver(mockCtrl)
		mockSSH = mocks.NewMockSSH(mockCtrl)
		mockPicker = mocks.NewMockNetworkPicker(mockCtrl)
		mockSystem = mocks.NewMockSystem(mockCtrl)
		mockConfig = mocks.NewMockConfig(mockCtrl)

		vbx = &vbox.VBox{
			Driver: mockDriver,
			SSH:    mockSSH,
			Picker: mockPicker,
			System: mockSystem,
			Config: mockConfig,
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
					mockConfig.EXPECT().GetOVAPath().Return("some-ova-path", nil),
					mockConfig.EXPECT().GetPCFDevDir().Return("some-pcfdev-dir", nil),
					mockDriver.EXPECT().GetVirtualSystemNumbersOfHardDiskImages("some-ova-path").Return([]string{"1"}, nil),
					mockDriver.EXPECT().VBoxManage("import", "some-ova-path", "--vsys", "0", "--unit", "1", "--disk", filepath.Join("some-pcfdev-dir", "some-vm-disk0.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return(vboxnets, nil),
					mockPicker.EXPECT().SelectAvailableNetworkInterface(vboxnets).Return(iface, true, nil),
					mockDriver.EXPECT().AttachNetworkInterface("some-interface", "some-vm"),
					mockDriver.EXPECT().ForwardPort("some-vm", "ssh", "some-port", "22"),
					mockConfig.EXPECT().GetDesiredMemory().Return(uint64(2000), nil),
					mockDriver.EXPECT().SetMemory("some-vm", uint64(2000)),
				)
				err := vbx.ImportVM("some-vm")
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
					mockConfig.EXPECT().GetOVAPath().Return("some-ova-path", nil),
					mockConfig.EXPECT().GetPCFDevDir().Return("some-pcfdev-dir", nil),
					mockDriver.EXPECT().GetVirtualSystemNumbersOfHardDiskImages("some-ova-path").Return([]string{"1"}, nil),
					mockDriver.EXPECT().VBoxManage("import", "some-ova-path", "--vsys", "0", "--unit", "1", "--disk", filepath.Join("some-pcfdev-dir", "some-vm-disk0.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return(vboxnets, nil),
					mockPicker.EXPECT().SelectAvailableNetworkInterface(vboxnets).Return(iface, false, nil),
					mockDriver.EXPECT().CreateHostOnlyInterface(ip).Return("some-interface", nil),
					mockDriver.EXPECT().AttachNetworkInterface("some-interface", "some-vm"),
					mockDriver.EXPECT().ForwardPort("some-vm", "ssh", "some-port", "22"),
					mockConfig.EXPECT().GetDesiredMemory().Return(uint64(2000), nil),
					mockDriver.EXPECT().SetMemory("some-vm", uint64(2000)),
				)
				err := vbx.ImportVM("some-vm")
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when the system has more than the maximum amount of free memory", func() {
			It("should give the VM the maximum amount of memory", func() {
				iface := &network.Interface{
					Name: "some-interface",
				}
				vboxnets := []*network.Interface{iface}
				gomock.InOrder(
					mockSSH.EXPECT().GenerateAddress().Return("some-host", "some-port", nil),
					mockConfig.EXPECT().GetOVAPath().Return("some-ova-path", nil),
					mockConfig.EXPECT().GetPCFDevDir().Return("some-pcfdev-dir", nil),
					mockDriver.EXPECT().GetVirtualSystemNumbersOfHardDiskImages("some-ova-path").Return([]string{"1"}, nil),
					mockDriver.EXPECT().VBoxManage("import", "some-ova-path", "--vsys", "0", "--unit", "1", "--disk", filepath.Join("some-pcfdev-dir", "some-vm-disk0.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return(vboxnets, nil),
					mockPicker.EXPECT().SelectAvailableNetworkInterface(vboxnets).Return(iface, true, nil),
					mockDriver.EXPECT().AttachNetworkInterface("some-interface", "some-vm"),
					mockDriver.EXPECT().ForwardPort("some-vm", "ssh", "some-port", "22"),
					mockConfig.EXPECT().GetDesiredMemory().Return(uint64(0), nil),
					mockConfig.EXPECT().GetMaxMemory().Return(uint64(1000)),
					mockConfig.EXPECT().GetMinMemory().Return(uint64(1000)),
					mockSystem.EXPECT().FreeMemory().Return(uint64(2000), nil),
					mockDriver.EXPECT().SetMemory("some-vm", uint64(1000)),
				)
				err := vbx.ImportVM("some-vm")
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when the system has more than the minimum amount of free memory but less than the maximum", func() {
			It("should give the VM all the free memory", func() {
				iface := &network.Interface{
					Name: "some-interface",
				}
				vboxnets := []*network.Interface{iface}
				gomock.InOrder(
					mockSSH.EXPECT().GenerateAddress().Return("some-host", "some-port", nil),
					mockConfig.EXPECT().GetOVAPath().Return("some-ova-path", nil),
					mockConfig.EXPECT().GetPCFDevDir().Return("some-pcfdev-dir", nil),
					mockDriver.EXPECT().GetVirtualSystemNumbersOfHardDiskImages("some-ova-path").Return([]string{"1"}, nil),
					mockDriver.EXPECT().VBoxManage("import", "some-ova-path", "--vsys", "0", "--unit", "1", "--disk", filepath.Join("some-pcfdev-dir", "some-vm-disk0.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return(vboxnets, nil),
					mockPicker.EXPECT().SelectAvailableNetworkInterface(vboxnets).Return(iface, true, nil),
					mockDriver.EXPECT().AttachNetworkInterface("some-interface", "some-vm"),
					mockDriver.EXPECT().ForwardPort("some-vm", "ssh", "some-port", "22"),
					mockConfig.EXPECT().GetDesiredMemory().Return(uint64(0), nil),
					mockConfig.EXPECT().GetMaxMemory().Return(uint64(2000)),
					mockConfig.EXPECT().GetMinMemory().Return(uint64(1000)),
					mockSystem.EXPECT().FreeMemory().Return(uint64(1300), nil),
					mockDriver.EXPECT().SetMemory("some-vm", uint64(1300)),
				)
				err := vbx.ImportVM("some-vm")
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when the system has less than the minimum amount of free memory", func() {
			It("should give the VM the minimum amount of memory", func() {
				iface := &network.Interface{
					Name: "some-interface",
				}
				vboxnets := []*network.Interface{iface}
				gomock.InOrder(
					mockSSH.EXPECT().GenerateAddress().Return("some-host", "some-port", nil),
					mockConfig.EXPECT().GetOVAPath().Return("some-ova-path", nil),
					mockConfig.EXPECT().GetPCFDevDir().Return("some-pcfdev-dir", nil),
					mockDriver.EXPECT().GetVirtualSystemNumbersOfHardDiskImages("some-ova-path").Return([]string{"1"}, nil),
					mockDriver.EXPECT().VBoxManage("import", "some-ova-path", "--vsys", "0", "--unit", "1", "--disk", filepath.Join("some-pcfdev-dir", "some-vm-disk0.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return(vboxnets, nil),
					mockPicker.EXPECT().SelectAvailableNetworkInterface(vboxnets).Return(iface, true, nil),
					mockDriver.EXPECT().AttachNetworkInterface("some-interface", "some-vm"),
					mockDriver.EXPECT().ForwardPort("some-vm", "ssh", "some-port", "22"),
					mockConfig.EXPECT().GetDesiredMemory().Return(uint64(0), nil),
					mockConfig.EXPECT().GetMaxMemory().Return(uint64(2000)),
					mockConfig.EXPECT().GetMinMemory().Return(uint64(1000)),
					mockSystem.EXPECT().FreeMemory().Return(uint64(500), nil),
					mockDriver.EXPECT().SetMemory("some-vm", uint64(1000)),
				)
				err := vbx.ImportVM("some-vm")
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when the user has set the memory explicitly", func() {
			It("should give the VM the desired amount of memory", func() {
				iface := &network.Interface{
					Name: "some-interface",
				}
				vboxnets := []*network.Interface{iface}
				gomock.InOrder(
					mockSSH.EXPECT().GenerateAddress().Return("some-host", "some-port", nil),
					mockConfig.EXPECT().GetOVAPath().Return("some-ova-path", nil),
					mockConfig.EXPECT().GetPCFDevDir().Return("some-pcfdev-dir", nil),
					mockDriver.EXPECT().GetVirtualSystemNumbersOfHardDiskImages("some-ova-path").Return([]string{"1"}, nil),
					mockDriver.EXPECT().VBoxManage("import", "some-ova-path", "--vsys", "0", "--unit", "1", "--disk", filepath.Join("some-pcfdev-dir", "some-vm-disk0.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return(vboxnets, nil),
					mockPicker.EXPECT().SelectAvailableNetworkInterface(vboxnets).Return(iface, true, nil),
					mockDriver.EXPECT().AttachNetworkInterface("some-interface", "some-vm"),
					mockDriver.EXPECT().ForwardPort("some-vm", "ssh", "some-port", "22"),
					mockConfig.EXPECT().GetDesiredMemory().Return(uint64(2000), nil),
					mockDriver.EXPECT().SetMemory("some-vm", uint64(2000)),
				)
				err := vbx.ImportVM("some-vm")
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when getting free memory returns an error", func() {
			It("should return an error", func() {
				iface := &network.Interface{
					Name: "some-interface",
				}
				vboxnets := []*network.Interface{iface}
				gomock.InOrder(
					mockSSH.EXPECT().GenerateAddress().Return("some-host", "some-port", nil),
					mockConfig.EXPECT().GetOVAPath().Return("some-ova-path", nil),
					mockConfig.EXPECT().GetPCFDevDir().Return("some-pcfdev-dir", nil),
					mockDriver.EXPECT().GetVirtualSystemNumbersOfHardDiskImages("some-ova-path").Return([]string{"1"}, nil),
					mockDriver.EXPECT().VBoxManage("import", "some-ova-path", "--vsys", "0", "--unit", "1", "--disk", filepath.Join("some-pcfdev-dir", "some-vm-disk0.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return(vboxnets, nil),
					mockPicker.EXPECT().SelectAvailableNetworkInterface(vboxnets).Return(iface, true, nil),
					mockDriver.EXPECT().AttachNetworkInterface("some-interface", "some-vm"),
					mockDriver.EXPECT().ForwardPort("some-vm", "ssh", "some-port", "22"),
					mockConfig.EXPECT().GetDesiredMemory().Return(uint64(0), nil),
					mockConfig.EXPECT().GetMaxMemory().Return(uint64(2000)),
					mockConfig.EXPECT().GetMinMemory().Return(uint64(1000)),
					mockSystem.EXPECT().FreeMemory().Return(uint64(0), errors.New("some-error")),
				)
				Expect(vbx.ImportVM("some-vm")).To(MatchError("some-error"))
			})
		})

		Context("when getting desired memory returns an error", func() {
			It("should return an error", func() {
				iface := &network.Interface{
					Name: "some-interface",
				}
				vboxnets := []*network.Interface{iface}
				gomock.InOrder(
					mockSSH.EXPECT().GenerateAddress().Return("some-host", "some-port", nil),
					mockConfig.EXPECT().GetOVAPath().Return("some-ova-path", nil),
					mockConfig.EXPECT().GetPCFDevDir().Return("some-pcfdev-dir", nil),
					mockDriver.EXPECT().GetVirtualSystemNumbersOfHardDiskImages("some-ova-path").Return([]string{"1"}, nil),
					mockDriver.EXPECT().VBoxManage("import", "some-ova-path", "--vsys", "0", "--unit", "1", "--disk", filepath.Join("some-pcfdev-dir", "some-vm-disk0.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return(vboxnets, nil),
					mockPicker.EXPECT().SelectAvailableNetworkInterface(vboxnets).Return(iface, true, nil),
					mockDriver.EXPECT().AttachNetworkInterface("some-interface", "some-vm"),
					mockDriver.EXPECT().ForwardPort("some-vm", "ssh", "some-port", "22"),
					mockConfig.EXPECT().GetDesiredMemory().Return(uint64(0), errors.New("some-error")),
				)
				Expect(vbx.ImportVM("some-vm")).To(MatchError("some-error"))
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
					mockConfig.EXPECT().GetOVAPath().Return("some-ova-path", nil),
					mockConfig.EXPECT().GetPCFDevDir().Return("some-pcfdev-dir", nil),
					mockDriver.EXPECT().GetVirtualSystemNumbersOfHardDiskImages("some-ova-path").Return([]string{"1"}, nil),
					mockDriver.EXPECT().VBoxManage("import", "some-ova-path", "--vsys", "0", "--unit", "1", "--disk", filepath.Join("some-pcfdev-dir", "some-vm-disk0.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return(vboxnets, nil),
					mockPicker.EXPECT().SelectAvailableNetworkInterface(vboxnets).Return(iface, true, nil),
					mockDriver.EXPECT().AttachNetworkInterface("some-interface", "some-vm"),
					mockDriver.EXPECT().ForwardPort("some-vm", "ssh", "some-port", "22"),
					mockConfig.EXPECT().GetDesiredMemory().Return(uint64(2000), nil),
					mockDriver.EXPECT().SetMemory("some-vm", uint64(2000)).Return(errors.New("some-error")),
				)
				Expect(vbx.ImportVM("some-vm")).To(MatchError("some-error"))
			})
		})

		Context("when it fails to get vbox hostonly interfaces", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockSSH.EXPECT().GenerateAddress().Return("some-host", "some-port", nil),
					mockConfig.EXPECT().GetOVAPath().Return("some-ova-path", nil),
					mockConfig.EXPECT().GetPCFDevDir().Return("some-pcfdev-dir", nil),
					mockDriver.EXPECT().GetVirtualSystemNumbersOfHardDiskImages("some-ova-path").Return([]string{"1"}, nil),
					mockDriver.EXPECT().VBoxManage("import", "some-ova-path", "--vsys", "0", "--unit", "1", "--disk", filepath.Join("some-pcfdev-dir", "some-vm-disk0.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return([]*network.Interface{}, errors.New("some-error")),
				)
				err := vbx.ImportVM("some-vm")
				Expect(err).To(MatchError("some-error"))
			})
		})

		Context("when it fails to get select available interface", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockSSH.EXPECT().GenerateAddress().Return("some-host", "some-port", nil),
					mockConfig.EXPECT().GetOVAPath().Return("some-ova-path", nil),
					mockConfig.EXPECT().GetPCFDevDir().Return("some-pcfdev-dir", nil),
					mockDriver.EXPECT().GetVirtualSystemNumbersOfHardDiskImages("some-ova-path").Return([]string{"1"}, nil),
					mockDriver.EXPECT().VBoxManage("import", "some-ova-path", "--vsys", "0", "--unit", "1", "--disk", filepath.Join("some-pcfdev-dir", "some-vm-disk0.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return([]*network.Interface{}, nil),
					mockPicker.EXPECT().SelectAvailableNetworkInterface([]*network.Interface{}).Return(nil, false, errors.New("some-error")),
				)
				err := vbx.ImportVM("some-vm")
				Expect(err).To(MatchError("some-error"))
			})
		})

		Context("fail to acquire random host port", func() {
			It("should return an error", func() {
				mockSSH.EXPECT().GenerateAddress().Return("", "", errors.New("some-error"))

				err := vbx.ImportVM("some-vm")
				Expect(err).To(MatchError("some-error"))
			})
		})

		Context("VM fails to import", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockSSH.EXPECT().GenerateAddress().Return("some-host", "some-port", nil),
					mockConfig.EXPECT().GetOVAPath().Return("some-ova-path", nil),
					mockConfig.EXPECT().GetPCFDevDir().Return("some-pcfdev-dir", nil),
					mockDriver.EXPECT().GetVirtualSystemNumbersOfHardDiskImages("some-ova-path").Return([]string{"1"}, nil),
					mockDriver.EXPECT().VBoxManage("import", "some-ova-path", "--vsys", "0", "--unit", "1", "--disk", filepath.Join("some-pcfdev-dir", "some-vm-disk0.vmdk")).Return(nil, errors.New("some-error")),
				)
				err := vbx.ImportVM("some-vm")
				Expect(err).To(MatchError("some-error"))
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
					mockConfig.EXPECT().GetOVAPath().Return("some-ova-path", nil),
					mockConfig.EXPECT().GetPCFDevDir().Return("some-pcfdev-dir", nil),
					mockDriver.EXPECT().GetVirtualSystemNumbersOfHardDiskImages("some-ova-path").Return([]string{"1"}, nil),
					mockDriver.EXPECT().VBoxManage("import", "some-ova-path", "--vsys", "0", "--unit", "1", "--disk", filepath.Join("some-pcfdev-dir", "some-vm-disk0.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return([]*network.Interface{}, nil),
					mockPicker.EXPECT().SelectAvailableNetworkInterface([]*network.Interface{}).Return(iface, false, nil),
					mockDriver.EXPECT().CreateHostOnlyInterface(ip).Return("", errors.New("some-error")),
				)
				err := vbx.ImportVM("some-vm")
				Expect(err).To(MatchError("some-error"))
			})
		})

		Context("when getting the ova path fails", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockSSH.EXPECT().GenerateAddress().Return("some-host", "some-port", nil),
					mockConfig.EXPECT().GetOVAPath().Return("", errors.New("some-error")),
				)

				err := vbx.ImportVM("some-vm")
				Expect(err).To(MatchError("some-error"))
			})
		})
		Context("when getting the pcfdev dir fails", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockSSH.EXPECT().GenerateAddress().Return("some-host", "some-port", nil),
					mockConfig.EXPECT().GetOVAPath().Return("some-ova-path", nil),
					mockConfig.EXPECT().GetPCFDevDir().Return("", errors.New("some-error")),
				)

				err := vbx.ImportVM("some-vm")
				Expect(err).To(MatchError("some-error"))
			})
		})

		Context("when attaching an interface fails", func() {
			It("should return an error", func() {
				iface := &network.Interface{
					Name: "some-interface",
				}
				gomock.InOrder(
					mockSSH.EXPECT().GenerateAddress().Return("some-host", "some-port", nil),
					mockConfig.EXPECT().GetOVAPath().Return("some-ova-path", nil),
					mockConfig.EXPECT().GetPCFDevDir().Return("some-pcfdev-dir", nil),
					mockDriver.EXPECT().GetVirtualSystemNumbersOfHardDiskImages("some-ova-path").Return([]string{"1"}, nil),
					mockDriver.EXPECT().VBoxManage("import", "some-ova-path", "--vsys", "0", "--unit", "1", "--disk", filepath.Join("some-pcfdev-dir", "some-vm-disk0.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return([]*network.Interface{}, nil),
					mockPicker.EXPECT().SelectAvailableNetworkInterface([]*network.Interface{}).Return(iface, true, nil),
					mockDriver.EXPECT().AttachNetworkInterface("some-interface", "some-vm").Return(errors.New("some-error")),
				)
				err := vbx.ImportVM("some-vm")
				Expect(err).To(MatchError("some-error"))
			})
		})

		Context("when retriving the virtual system numbers of hard disk images fails", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockSSH.EXPECT().GenerateAddress().Return("some-host", "some-port", nil),
					mockConfig.EXPECT().GetOVAPath().Return("some-ova-path", nil),
					mockConfig.EXPECT().GetPCFDevDir().Return("some-pcfdev-dir", nil),
					mockDriver.EXPECT().GetVirtualSystemNumbersOfHardDiskImages("some-ova-path").Return(nil, errors.New("some-error")),
				)
				err := vbx.ImportVM("some-vm")
				Expect(err).To(MatchError("some-error"))
			})
		})

		Context("when port fowarding fails", func() {
			iface := &network.Interface{
				Name: "some-interface",
			}
			It("should return an error", func() {
				gomock.InOrder(
					mockSSH.EXPECT().GenerateAddress().Return("some-host", "some-port", nil),
					mockConfig.EXPECT().GetOVAPath().Return("some-ova-path", nil),
					mockConfig.EXPECT().GetPCFDevDir().Return("some-pcfdev-dir", nil),
					mockDriver.EXPECT().GetVirtualSystemNumbersOfHardDiskImages("some-ova-path").Return([]string{"1"}, nil),
					mockDriver.EXPECT().VBoxManage("import", "some-ova-path", "--vsys", "0", "--unit", "1", "--disk", filepath.Join("some-pcfdev-dir", "some-vm-disk0.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return([]*network.Interface{}, nil),
					mockPicker.EXPECT().SelectAvailableNetworkInterface([]*network.Interface{}).Return(iface, true, nil),
					mockDriver.EXPECT().AttachNetworkInterface("some-interface", "some-vm"),
					mockDriver.EXPECT().ForwardPort("some-vm", "ssh", "some-port", "22").Return(errors.New("some-error")),
				)
				err := vbx.ImportVM("some-vm")
				Expect(err).To(MatchError("some-error"))
			})
		})
	})

	Describe("#StartVM", func() {
		Context("when VM is already imported", func() {
			It("starts without reimporting", func() {
				gomock.InOrder(
					mockDriver.EXPECT().StartVM("some-vm"),
					mockSSH.EXPECT().RunSSHCommand("echo -e \"auto eth1\niface eth1 inet static\naddress 192.168.22.11\nnetmask 255.255.255.0\" | sudo tee -a /etc/network/interfaces", "some-port", 2*time.Minute, ioutil.Discard, ioutil.Discard),
					mockConfig.EXPECT().GetHTTPProxy().Return("some-http-proxy"),
					mockConfig.EXPECT().GetHTTPSProxy().Return("some-https-proxy"),
					mockConfig.EXPECT().GetNoProxy().Return("some-no-proxy"),
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
				gomock.InOrder(
					mockDriver.EXPECT().StartVM("some-vm"),
					mockSSH.EXPECT().RunSSHCommand("echo -e \"auto eth1\niface eth1 inet static\naddress 192.168.22.11\nnetmask 255.255.255.0\" | sudo tee -a /etc/network/interfaces", "some-port", 2*time.Minute, ioutil.Discard, ioutil.Discard),
					mockConfig.EXPECT().GetHTTPProxy().Return("127.0.0.1"),
					mockConfig.EXPECT().GetHTTPSProxy().Return("127.0.0.1:8080"),
					mockConfig.EXPECT().GetNoProxy().Return("some-no-proxy"),
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
						mockConfig.EXPECT().GetHTTPProxy().Return("some-http-proxy"),
						mockConfig.EXPECT().GetHTTPSProxy().Return("some-https-proxy"),
						mockConfig.EXPECT().GetNoProxy().Return("some-no-proxy"),
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

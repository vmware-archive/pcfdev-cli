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
		mockFS     *mocks.MockFS
		vbx        *vbox.VBox
		conf       *config.Config
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockDriver = mocks.NewMockDriver(mockCtrl)
		mockSSH = mocks.NewMockSSH(mockCtrl)
		mockFS = mocks.NewMockFS(mockCtrl)
		mockPicker = mocks.NewMockNetworkPicker(mockCtrl)

		conf = &config.Config{
			PCFDevHome: "some-pcfdev-home",
			OVADir:     "some-ova-dir",
			VMDir:      "some-vm-dir",
			HTTPProxy:  "some-http-proxy",
			HTTPSProxy: "some-https-proxy",
			NoProxy:    "some-no-proxy",

			MinMemory: uint64(1000),
			MaxMemory: uint64(2000),
		}

		vbx = &vbox.VBox{
			Driver: mockDriver,
			SSH:    mockSSH,
			FS:     mockFS,
			Picker: mockPicker,
			Config: conf,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("#ImportVM", func() {
		Context("when there is no unused VBox interface", func() {
			It("should create and attach that interface", func() {
				vboxnets := []*network.Interface{
					&network.Interface{
						Name: "some-used-vbox-interface",
						IP:   "some-used-ip",
					},
				}
				gomock.InOrder(
					mockDriver.EXPECT().CreateVM("some-vm", "some-vm-dir"),
					mockFS.EXPECT().Extract(filepath.Join("some-ova-dir", "some-vm.ova"), "some-ova-dir", "some-vm-disk1.vmdk"),
					mockDriver.EXPECT().CloneDisk(filepath.Join("some-ova-dir", "some-vm-disk1.vmdk"), filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockFS.EXPECT().Remove(filepath.Join("some-ova-dir", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().AttachDisk("some-vm", filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return(vboxnets, nil),
					mockPicker.EXPECT().SelectAvailableIP(vboxnets).Return("some-unused-ip", nil),
					mockDriver.EXPECT().GetUnusedHostOnlyInterface().Return("", nil),
					mockDriver.EXPECT().CreateHostOnlyInterface("some-unused-ip").Return("some-interface", nil),
					mockDriver.EXPECT().AttachNetworkInterface("some-interface", "some-vm"),
					mockSSH.EXPECT().GenerateAddress().Return("some-host", "some-port", nil),
					mockDriver.EXPECT().ForwardPort("some-vm", "ssh", "some-port", "22"),
					mockDriver.EXPECT().SetCPUs("some-vm", 7),
					mockDriver.EXPECT().SetMemory("some-vm", uint64(2000)),
				)
				err := vbx.ImportVM(&config.VMConfig{
					Name:     "some-vm",
					DiskName: "some-vm-disk1.vmdk",
					Memory:   uint64(2000),
					CPUs:     7,
				})
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when there is an unused VBox interface", func() {
			It("should attach that interface", func() {
				vboxnets := []*network.Interface{
					&network.Interface{
						Name: "some-unused-vbox-interface",
						IP:   "some-unused-ip",
					},
				}
				gomock.InOrder(
					mockDriver.EXPECT().CreateVM("some-vm", "some-vm-dir"),
					mockFS.EXPECT().Extract(filepath.Join("some-ova-dir", "some-vm.ova"), "some-ova-dir", "some-vm-disk1.vmdk"),
					mockDriver.EXPECT().CloneDisk(filepath.Join("some-ova-dir", "some-vm-disk1.vmdk"), filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockFS.EXPECT().Remove(filepath.Join("some-ova-dir", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().AttachDisk("some-vm", filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return(vboxnets, nil),
					mockPicker.EXPECT().SelectAvailableIP(vboxnets).Return("some-unused-ip", nil),
					mockDriver.EXPECT().GetUnusedHostOnlyInterface().Return("some-unused-vbox-interface", nil),
					mockDriver.EXPECT().ConfigureHostOnlyInterface("some-unused-vbox-interface", "some-unused-ip"),
					mockDriver.EXPECT().AttachNetworkInterface("some-unused-vbox-interface", "some-vm"),
					mockSSH.EXPECT().GenerateAddress().Return("some-host", "some-port", nil),
					mockDriver.EXPECT().ForwardPort("some-vm", "ssh", "some-port", "22"),
					mockDriver.EXPECT().SetCPUs("some-vm", 7),
					mockDriver.EXPECT().SetMemory("some-vm", uint64(2000)),
				)
				err := vbx.ImportVM(&config.VMConfig{
					Name:     "some-vm",
					DiskName: "some-vm-disk1.vmdk",
					Memory:   uint64(2000),
					CPUs:     7,
				})
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when creating the VM returns an error", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockDriver.EXPECT().CreateVM("some-vm", "some-vm-dir").Return(errors.New("some-error")),
				)
				Expect(vbx.ImportVM(
					&config.VMConfig{
						Name:     "some-vm",
						DiskName: "some-vm-disk1.vmdk",
					})).To(MatchError("some-error"))
			})
		})

		Context("when extracting the file returns an error", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockDriver.EXPECT().CreateVM("some-vm", "some-vm-dir").Return(nil),
					mockFS.EXPECT().Extract(filepath.Join("some-ova-dir", "some-vm.ova"), "some-ova-dir", "some-vm-disk1.vmdk").Return(errors.New("some-error")),
				)
				Expect(vbx.ImportVM(
					&config.VMConfig{
						Name:     "some-vm",
						DiskName: "some-vm-disk1.vmdk",
					})).To(MatchError("some-error"))
			})
		})

		Context("when cloning the disk fails", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockDriver.EXPECT().CreateVM("some-vm", "some-vm-dir").Return(nil),
					mockFS.EXPECT().Extract(filepath.Join("some-ova-dir", "some-vm.ova"), "some-ova-dir", "some-vm-disk1.vmdk"),
					mockDriver.EXPECT().CloneDisk(filepath.Join("some-ova-dir", "some-vm-disk1.vmdk"), filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")).Return(errors.New("some-error")),
				)
				Expect(vbx.ImportVM(
					&config.VMConfig{
						Name:     "some-vm",
						DiskName: "some-vm-disk1.vmdk",
					})).To(MatchError("some-error"))
			})
		})

		Context("when removing the compressed disk fails", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockDriver.EXPECT().CreateVM("some-vm", "some-vm-dir").Return(nil),
					mockFS.EXPECT().Extract(filepath.Join("some-ova-dir", "some-vm.ova"), "some-ova-dir", "some-vm-disk1.vmdk"),
					mockDriver.EXPECT().CloneDisk(filepath.Join("some-ova-dir", "some-vm-disk1.vmdk"), filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")).Return(nil),
					mockFS.EXPECT().Remove(filepath.Join("some-ova-dir", "some-vm-disk1.vmdk")).Return(errors.New("some-error")),
				)
				Expect(vbx.ImportVM(
					&config.VMConfig{
						Name:     "some-vm",
						DiskName: "some-vm-disk1.vmdk",
					})).To(MatchError("some-error"))
			})
		})

		Context("when attaching the disk fails", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockDriver.EXPECT().CreateVM("some-vm", "some-vm-dir").Return(nil),
					mockFS.EXPECT().Extract(filepath.Join("some-ova-dir", "some-vm.ova"), "some-ova-dir", "some-vm-disk1.vmdk"),
					mockDriver.EXPECT().CloneDisk(filepath.Join("some-ova-dir", "some-vm-disk1.vmdk"), filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")).Return(nil),
					mockFS.EXPECT().Remove(filepath.Join("some-ova-dir", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().AttachDisk("some-vm", filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")).Return(errors.New("some-error")),
				)
				Expect(vbx.ImportVM(
					&config.VMConfig{
						Name:     "some-vm",
						DiskName: "some-vm-disk1.vmdk",
					})).To(MatchError("some-error"))
			})
		})

		Context("when geting vbox host-only interfaces fails", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockDriver.EXPECT().CreateVM("some-vm", "some-vm-dir").Return(nil),
					mockFS.EXPECT().Extract(filepath.Join("some-ova-dir", "some-vm.ova"), "some-ova-dir", "some-vm-disk1.vmdk"),
					mockDriver.EXPECT().CloneDisk(filepath.Join("some-ova-dir", "some-vm-disk1.vmdk"), filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")).Return(nil),
					mockFS.EXPECT().Remove(filepath.Join("some-ova-dir", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().AttachDisk("some-vm", filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return([]*network.Interface{}, errors.New("some-error")),
				)
				Expect(vbx.ImportVM(&config.VMConfig{
					Name:     "some-vm",
					DiskName: "some-vm-disk1.vmdk",
					Memory:   uint64(2000),
					CPUs:     7,
				})).To(MatchError("some-error"))
			})
		})

		Context("when selecting an available IP fails", func() {
			It("should return an error", func() {
				vboxnets := []*network.Interface{
					&network.Interface{
						Name: "some-used-vbox-interface",
						IP:   "some-used-ip",
					},
				}
				gomock.InOrder(
					mockDriver.EXPECT().CreateVM("some-vm", "some-vm-dir").Return(nil),
					mockFS.EXPECT().Extract(filepath.Join("some-ova-dir", "some-vm.ova"), "some-ova-dir", "some-vm-disk1.vmdk"),
					mockDriver.EXPECT().CloneDisk(filepath.Join("some-ova-dir", "some-vm-disk1.vmdk"), filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")).Return(nil),
					mockFS.EXPECT().Remove(filepath.Join("some-ova-dir", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().AttachDisk("some-vm", filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return(vboxnets, nil),
					mockPicker.EXPECT().SelectAvailableIP(vboxnets).Return("", errors.New("some-error")),
				)
				Expect(vbx.ImportVM(&config.VMConfig{
					Name:   "some-vm",
					Memory: uint64(2000),
					CPUs:   7,
				})).To(MatchError("some-error"))
			})
		})

		Context("when getting an unused host-only interface fails", func() {
			It("should return an error", func() {
				vboxnets := []*network.Interface{
					&network.Interface{
						Name: "some-used-vbox-interface",
						IP:   "some-used-ip",
					},
				}
				gomock.InOrder(
					mockDriver.EXPECT().CreateVM("some-vm", "some-vm-dir").Return(nil),
					mockFS.EXPECT().Extract(filepath.Join("some-ova-dir", "some-vm.ova"), "some-ova-dir", "some-vm-disk1.vmdk"),
					mockDriver.EXPECT().CloneDisk(filepath.Join("some-ova-dir", "some-vm-disk1.vmdk"), filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")).Return(nil),
					mockFS.EXPECT().Remove(filepath.Join("some-ova-dir", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().AttachDisk("some-vm", filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return(vboxnets, nil),
					mockPicker.EXPECT().SelectAvailableIP(vboxnets).Return("some-unused-ip", nil),
					mockDriver.EXPECT().GetUnusedHostOnlyInterface().Return("", errors.New("some-error")),
				)
				Expect(vbx.ImportVM(&config.VMConfig{
					Name:   "some-vm",
					Memory: uint64(2000),
					CPUs:   7,
				})).To(MatchError("some-error"))
			})
		})

		Context("when creating a host-only interface fails", func() {
			It("should return an error", func() {
				vboxnets := []*network.Interface{
					&network.Interface{
						Name: "some-used-vbox-interface",
						IP:   "some-used-ip",
					},
				}
				gomock.InOrder(
					mockDriver.EXPECT().CreateVM("some-vm", "some-vm-dir").Return(nil),
					mockFS.EXPECT().Extract(filepath.Join("some-ova-dir", "some-vm.ova"), "some-ova-dir", "some-vm-disk1.vmdk"),
					mockDriver.EXPECT().CloneDisk(filepath.Join("some-ova-dir", "some-vm-disk1.vmdk"), filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")).Return(nil),
					mockFS.EXPECT().Remove(filepath.Join("some-ova-dir", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().AttachDisk("some-vm", filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return(vboxnets, nil),
					mockPicker.EXPECT().SelectAvailableIP(vboxnets).Return("some-unused-ip", nil),
					mockDriver.EXPECT().GetUnusedHostOnlyInterface().Return("", nil),
					mockDriver.EXPECT().CreateHostOnlyInterface("some-unused-ip").Return("", errors.New("some-error")),
				)
				Expect(vbx.ImportVM(&config.VMConfig{
					Name:     "some-vm",
					DiskName: "some-vm-disk1.vmdk",
					Memory:   uint64(2000),
					CPUs:     7,
				})).To(MatchError("some-error"))
			})
		})

		Context("when configuring a host-only interface fails", func() {
			It("should return an error", func() {
				vboxnets := []*network.Interface{
					&network.Interface{
						Name: "some-unused-vbox-interface",
						IP:   "some-unused-ip",
					},
				}
				gomock.InOrder(
					mockDriver.EXPECT().CreateVM("some-vm", "some-vm-dir").Return(nil),
					mockFS.EXPECT().Extract(filepath.Join("some-ova-dir", "some-vm.ova"), "some-ova-dir", "some-vm-disk1.vmdk"),
					mockDriver.EXPECT().CloneDisk(filepath.Join("some-ova-dir", "some-vm-disk1.vmdk"), filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")).Return(nil),
					mockFS.EXPECT().Remove(filepath.Join("some-ova-dir", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().AttachDisk("some-vm", filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return(vboxnets, nil),
					mockPicker.EXPECT().SelectAvailableIP(vboxnets).Return("some-unused-ip", nil),
					mockDriver.EXPECT().GetUnusedHostOnlyInterface().Return("some-unused-vbox-interface", nil),
					mockDriver.EXPECT().ConfigureHostOnlyInterface("some-unused-vbox-interface", "some-unused-ip").Return(errors.New("some-error")),
				)
				Expect(vbx.ImportVM(&config.VMConfig{
					Name:     "some-vm",
					DiskName: "some-vm-disk1.vmdk",
					Memory:   uint64(2000),
					CPUs:     7,
				})).To(MatchError("some-error"))
			})
		})

		Context("when attaching an interface fails", func() {
			It("should return an error", func() {
				vboxnets := []*network.Interface{
					&network.Interface{
						Name: "some-used-vbox-interface",
						IP:   "some-used-ip",
					},
				}
				gomock.InOrder(
					mockDriver.EXPECT().CreateVM("some-vm", "some-vm-dir").Return(nil),
					mockFS.EXPECT().Extract(filepath.Join("some-ova-dir", "some-vm.ova"), "some-ova-dir", "some-vm-disk1.vmdk"),
					mockDriver.EXPECT().CloneDisk(filepath.Join("some-ova-dir", "some-vm-disk1.vmdk"), filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")).Return(nil),
					mockFS.EXPECT().Remove(filepath.Join("some-ova-dir", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().AttachDisk("some-vm", filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return(vboxnets, nil),
					mockPicker.EXPECT().SelectAvailableIP(vboxnets).Return("some-unused-ip", nil),
					mockDriver.EXPECT().GetUnusedHostOnlyInterface().Return("", nil),
					mockDriver.EXPECT().CreateHostOnlyInterface("some-unused-ip").Return("some-interface", nil),
					mockDriver.EXPECT().AttachNetworkInterface("some-interface", "some-vm").Return(errors.New("some-error")),
				)
				Expect(vbx.ImportVM(&config.VMConfig{
					Name:     "some-vm",
					DiskName: "some-vm-disk1.vmdk",
					Memory:   uint64(2000),
					CPUs:     7,
				})).To(MatchError("some-error"))
			})
		})

		Context("when generating an address fails", func() {
			It("should return an error", func() {
				vboxnets := []*network.Interface{
					&network.Interface{
						Name: "some-used-vbox-interface",
						IP:   "some-used-ip",
					},
				}
				gomock.InOrder(
					mockDriver.EXPECT().CreateVM("some-vm", "some-vm-dir").Return(nil),
					mockFS.EXPECT().Extract(filepath.Join("some-ova-dir", "some-vm.ova"), "some-ova-dir", "some-vm-disk1.vmdk"),
					mockDriver.EXPECT().CloneDisk(filepath.Join("some-ova-dir", "some-vm-disk1.vmdk"), filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")).Return(nil),
					mockFS.EXPECT().Remove(filepath.Join("some-ova-dir", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().AttachDisk("some-vm", filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return(vboxnets, nil),
					mockPicker.EXPECT().SelectAvailableIP(vboxnets).Return("some-unused-ip", nil),
					mockDriver.EXPECT().GetUnusedHostOnlyInterface().Return("", nil),
					mockDriver.EXPECT().CreateHostOnlyInterface("some-unused-ip").Return("some-interface", nil),
					mockDriver.EXPECT().AttachNetworkInterface("some-interface", "some-vm"),
					mockSSH.EXPECT().GenerateAddress().Return("", "", errors.New("some-error")),
				)

				Expect(vbx.ImportVM(&config.VMConfig{
					Name:     "some-vm",
					DiskName: "some-vm-disk1.vmdk",
					Memory:   uint64(2000),
					CPUs:     7,
				})).To(MatchError("some-error"))
			})
		})

		Context("when port fowarding fails", func() {
			It("should return an error", func() {
				vboxnets := []*network.Interface{
					&network.Interface{
						Name: "some-used-vbox-interface",
						IP:   "some-used-ip",
					},
				}
				gomock.InOrder(
					mockDriver.EXPECT().CreateVM("some-vm", "some-vm-dir").Return(nil),
					mockFS.EXPECT().Extract(filepath.Join("some-ova-dir", "some-vm.ova"), "some-ova-dir", "some-vm-disk1.vmdk"),
					mockDriver.EXPECT().CloneDisk(filepath.Join("some-ova-dir", "some-vm-disk1.vmdk"), filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")).Return(nil),
					mockFS.EXPECT().Remove(filepath.Join("some-ova-dir", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().AttachDisk("some-vm", filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return(vboxnets, nil),
					mockPicker.EXPECT().SelectAvailableIP(vboxnets).Return("some-unused-ip", nil),
					mockDriver.EXPECT().GetUnusedHostOnlyInterface().Return("", nil),
					mockDriver.EXPECT().CreateHostOnlyInterface("some-unused-ip").Return("some-interface", nil),
					mockDriver.EXPECT().AttachNetworkInterface("some-interface", "some-vm"),
					mockSSH.EXPECT().GenerateAddress().Return("some-host", "some-port", nil),
					mockDriver.EXPECT().ForwardPort("some-vm", "ssh", "some-port", "22").Return(errors.New("some-error")),
				)
				Expect(vbx.ImportVM(&config.VMConfig{
					Name:     "some-vm",
					DiskName: "some-vm-disk1.vmdk",
					Memory:   uint64(2000),
					CPUs:     7,
				})).To(MatchError("some-error"))
			})
		})

		Context("when setting the CPUs returns an error", func() {
			It("should return an error", func() {
				vboxnets := []*network.Interface{
					&network.Interface{
						Name: "some-used-vbox-interface",
						IP:   "some-used-ip",
					},
				}
				gomock.InOrder(
					mockDriver.EXPECT().CreateVM("some-vm", "some-vm-dir").Return(nil),
					mockFS.EXPECT().Extract(filepath.Join("some-ova-dir", "some-vm.ova"), "some-ova-dir", "some-vm-disk1.vmdk"),
					mockDriver.EXPECT().CloneDisk(filepath.Join("some-ova-dir", "some-vm-disk1.vmdk"), filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")).Return(nil),
					mockFS.EXPECT().Remove(filepath.Join("some-ova-dir", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().AttachDisk("some-vm", filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return(vboxnets, nil),
					mockPicker.EXPECT().SelectAvailableIP(vboxnets).Return("some-unused-ip", nil),
					mockDriver.EXPECT().GetUnusedHostOnlyInterface().Return("", nil),
					mockDriver.EXPECT().CreateHostOnlyInterface("some-unused-ip").Return("some-interface", nil),
					mockDriver.EXPECT().AttachNetworkInterface("some-interface", "some-vm"),
					mockSSH.EXPECT().GenerateAddress().Return("some-host", "some-port", nil),
					mockDriver.EXPECT().ForwardPort("some-vm", "ssh", "some-port", "22"),
					mockDriver.EXPECT().SetCPUs("some-vm", 7).Return(errors.New("some-error")),
				)
				Expect(vbx.ImportVM(&config.VMConfig{
					Name:     "some-vm",
					DiskName: "some-vm-disk1.vmdk",
					Memory:   uint64(2000),
					CPUs:     7,
				})).To(MatchError("some-error"))
			})
		})

		Context("when setting the memory returns an error", func() {
			It("should return an error", func() {
				vboxnets := []*network.Interface{
					&network.Interface{
						Name: "some-used-vbox-interface",
						IP:   "some-used-ip",
					},
				}
				gomock.InOrder(
					mockDriver.EXPECT().CreateVM("some-vm", "some-vm-dir").Return(nil),
					mockFS.EXPECT().Extract(filepath.Join("some-ova-dir", "some-vm.ova"), "some-ova-dir", "some-vm-disk1.vmdk"),
					mockDriver.EXPECT().CloneDisk(filepath.Join("some-ova-dir", "some-vm-disk1.vmdk"), filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")).Return(nil),
					mockFS.EXPECT().Remove(filepath.Join("some-ova-dir", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().AttachDisk("some-vm", filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return(vboxnets, nil),
					mockPicker.EXPECT().SelectAvailableIP(vboxnets).Return("some-unused-ip", nil),
					mockDriver.EXPECT().GetUnusedHostOnlyInterface().Return("", nil),
					mockDriver.EXPECT().CreateHostOnlyInterface("some-unused-ip").Return("some-interface", nil),
					mockDriver.EXPECT().AttachNetworkInterface("some-interface", "some-vm"),
					mockSSH.EXPECT().GenerateAddress().Return("some-host", "some-port", nil),
					mockDriver.EXPECT().ForwardPort("some-vm", "ssh", "some-port", "22"),
					mockDriver.EXPECT().SetCPUs("some-vm", 7),
					mockDriver.EXPECT().SetMemory("some-vm", uint64(2000)).Return(errors.New("some-error")),
				)
				Expect(vbx.ImportVM(&config.VMConfig{
					Name:     "some-vm",
					DiskName: "some-vm-disk1.vmdk",
					Memory:   uint64(2000),
					CPUs:     7,
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

				Expect(vbx.StartVM(&config.VMConfig{
					Name:    "some-vm",
					IP:      "192.168.22.11",
					SSHPort: "some-port",
					Domain:  "some-domain",
				})).To(Succeed())
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

				Expect(vbx.StartVM(&config.VMConfig{
					Name:    "some-vm",
					IP:      "192.168.22.11",
					SSHPort: "some-port",
					Domain:  "some-domain",
				})).To(Succeed())
			})

			Context("when a bad ip is passed to StartVM command", func() {
				It("should return an error", func() {
					gomock.InOrder(
						mockDriver.EXPECT().StartVM("some-vm"),
						mockSSH.EXPECT().RunSSHCommand("echo -e \"auto eth1\niface eth1 inet static\naddress some-bad-ip\nnetmask 255.255.255.0\" | sudo tee -a /etc/network/interfaces", "some-port", 2*time.Minute, ioutil.Discard, ioutil.Discard),
					)

					Expect(vbx.StartVM(&config.VMConfig{
						Name:    "some-vm",
						IP:      "some-bad-ip",
						SSHPort: "some-port",
						Domain:  "some-domain",
					})).To(MatchError("some-bad-ip is not one of the allowed PCF Dev ips"))
				})
			})

			Context("when VM fails to start", func() {
				It("should return an error", func() {
					gomock.InOrder(
						mockDriver.EXPECT().StartVM("some-vm").Return(errors.New("some-error")),
					)

					Expect(vbx.StartVM(&config.VMConfig{
						Name:    "some-vm",
						IP:      "192.168.22.11",
						SSHPort: "some-port",
						Domain:  "some-domain",
					})).To(MatchError("some-error"))
				})
			})

			Context("when SSH Command to set static ip fails", func() {
				It("should return an error", func() {
					gomock.InOrder(
						mockDriver.EXPECT().StartVM("some-vm"),
						mockSSH.EXPECT().RunSSHCommand(fmt.Sprintf("echo -e \"auto eth1\niface eth1 inet static\naddress some-ip\nnetmask 255.255.255.0\" | sudo tee -a /etc/network/interfaces"), "some-port", 2*time.Minute, ioutil.Discard, ioutil.Discard).Return(errors.New("some-error")),
					)

					Expect(vbx.StartVM(&config.VMConfig{
						Name:    "some-vm",
						IP:      "some-ip",
						SSHPort: "some-port",
						Domain:  "some-domain",
					})).To(MatchError("some-error"))
				})
			})

			Context("when VM fails to stop", func() {
				It("should return an error", func() {
					gomock.InOrder(
						mockDriver.EXPECT().StartVM("some-vm"),
						mockSSH.EXPECT().RunSSHCommand(fmt.Sprintf("echo -e \"auto eth1\niface eth1 inet static\naddress %s\nnetmask 255.255.255.0\" | sudo tee -a /etc/network/interfaces", "192.168.11.11"), "some-port", 2*time.Minute, ioutil.Discard, ioutil.Discard),
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

					Expect(vbx.StartVM(&config.VMConfig{
						Name:    "some-vm",
						IP:      "192.168.11.11",
						SSHPort: "some-port",
						Domain:  "some-domain",
					})).To(MatchError("some-error"))
				})
			})
		})
	})

	Describe("#StopVM", func() {
		It("should stop the VM", func() {
			mockDriver.EXPECT().StopVM("some-vm")

			err := vbx.StopVM(&config.VMConfig{Name: "some-vm"})
			Expect(err).NotTo(HaveOccurred())
		})

		Context("Driver fails to stop VM", func() {
			It("should return the error", func() {
				expectedError := errors.New("some-error")

				mockDriver.EXPECT().StopVM("some-vm").Return(expectedError)
				err := vbx.StopVM(&config.VMConfig{Name: "some-vm"})
				Expect(err).To(MatchError(expectedError))
			})
		})
	})

	Describe("#SuspendVM", func() {
		It("should suspend the VM", func() {
			mockDriver.EXPECT().SuspendVM("some-vm")

			err := vbx.SuspendVM(&config.VMConfig{Name: "some-vm"})
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when the Driver fails to suspend the VM", func() {
			It("should return the error", func() {
				expectedError := errors.New("some-error")

				mockDriver.EXPECT().SuspendVM("some-vm").Return(expectedError)
				err := vbx.SuspendVM(&config.VMConfig{Name: "some-vm"})
				Expect(err).To(MatchError(expectedError))
			})
		})
	})

	Describe("#ResumeVM", func() {
		It("should resume the VM", func() {
			mockDriver.EXPECT().ResumeVM("some-vm")

			err := vbx.ResumeVM(&config.VMConfig{Name: "some-vm"})
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when the Driver fails to resume the VM", func() {
			It("should return the error", func() {
				expectedError := errors.New("some-error")

				mockDriver.EXPECT().ResumeVM("some-vm").Return(expectedError)
				err := vbx.ResumeVM(&config.VMConfig{Name: "some-vm"})
				Expect(err).To(MatchError(expectedError))
			})
		})
	})

	Describe("#ConflictingVMPresent", func() {
		Context("when there are no conflicting VMs with the prefix pcfdev-", func() {
			It("should return false", func() {
				mockDriver.EXPECT().RunningVMs().Return([]string{"some-other-vm", "pcfdev-our-vm"}, nil)
				Expect(vbx.ConflictingVMPresent(&config.VMConfig{Name: "pcfdev-our-vm"})).To(BeFalse())
			})
		})

		Context("when there are conflicting VMs with the prefix pcfdev- running", func() {
			It("should return true", func() {
				mockDriver.EXPECT().RunningVMs().Return([]string{"pcfdev-conflicting-vm", "pcfdev-our-vm"}, nil)
				Expect(vbx.ConflictingVMPresent(&config.VMConfig{Name: "pcfdev-our-vm"})).To(BeTrue())
			})
		})

		Context("when getting running vms returns an error", func() {
			It("should return an error", func() {
				mockDriver.EXPECT().RunningVMs().Return(nil, errors.New("some-error"))
				_, err := vbx.ConflictingVMPresent(&config.VMConfig{Name: "pcfdev-our-vm"})
				Expect(err).To(MatchError("some-error"))
			})
		})
	})

	Describe("#Destroy", func() {
		It("should destroy the VM", func() {
			mockDriver.EXPECT().DestroyVM("some-vm")

			Expect(vbx.DestroyVM(&config.VMConfig{Name: "some-vm"})).To(Succeed())
		})

		Context("when the driver fails to destroy VM", func() {
			It("should return the error", func() {
				mockDriver.EXPECT().DestroyVM("some-vm").Return(errors.New("some-error"))

				Expect(vbx.DestroyVM(&config.VMConfig{Name: "some-vm"})).To(MatchError("some-error"))
			})
		})
	})

	Describe("#PowerOffVM", func() {
		It("should power off the VM", func() {
			mockDriver.EXPECT().PowerOffVM("some-vm")

			Expect(vbx.PowerOffVM(&config.VMConfig{Name: "some-vm"})).To(Succeed())
		})

		Context("when the driver fails to power off the VM", func() {
			It("should return the error", func() {
				mockDriver.EXPECT().PowerOffVM("some-vm").Return(errors.New("some-error"))

				Expect(vbx.PowerOffVM(&config.VMConfig{Name: "some-vm"})).To(MatchError("some-error"))
			})
		})
	})

	Describe("#DestroyPCFDevVMs", func() {
		It("should destroy VMs that begin with pcfdev-", func() {
			gomock.InOrder(
				mockDriver.EXPECT().VMs().Return([]string{"pcfdev-0.0.0", "pcfdev-0.0.1", "some-bad-vm-name"}, nil),
				mockDriver.EXPECT().PowerOffVM("pcfdev-0.0.0"),
				mockDriver.EXPECT().DestroyVM("pcfdev-0.0.0"),
				mockDriver.EXPECT().PowerOffVM("pcfdev-0.0.1"),
				mockDriver.EXPECT().DestroyVM("pcfdev-0.0.1"),
				mockDriver.EXPECT().VMs().Return([]string{}, nil),
			)

			Expect(vbx.DestroyPCFDevVMs()).To(Succeed())
		})

		Context("when getting VMs fails", func() {
			It("should return an error", func() {
				mockDriver.EXPECT().VMs().Return([]string{}, errors.New("some-error"))

				Expect(vbx.DestroyPCFDevVMs()).To(MatchError("some-error"))
			})
		})

		Context("when powering off a VM fails", func() {
			It("should destroy the VM and continue on to the next VM", func() {
				gomock.InOrder(
					mockDriver.EXPECT().VMs().Return([]string{"pcfdev-0.0.0", "pcfdev-0.0.1", "some-bad-vm-name"}, nil),
					mockDriver.EXPECT().PowerOffVM("pcfdev-0.0.0").Return(errors.New("some-error")),
					mockDriver.EXPECT().DestroyVM("pcfdev-0.0.0"),
					mockDriver.EXPECT().PowerOffVM("pcfdev-0.0.1"),
					mockDriver.EXPECT().DestroyVM("pcfdev-0.0.1"),
					mockDriver.EXPECT().VMs().Return([]string{"some-bad-vm-name"}, nil),
				)

				Expect(vbx.DestroyPCFDevVMs()).To(Succeed())
			})
		})

		Context("when destroying a VM fails", func() {
			It("should continue on to the next VM", func() {
				gomock.InOrder(
					mockDriver.EXPECT().VMs().Return([]string{"pcfdev-0.0.0", "pcfdev-0.0.1", "some-bad-vm-name"}, nil),
					mockDriver.EXPECT().PowerOffVM("pcfdev-0.0.0"),
					mockDriver.EXPECT().DestroyVM("pcfdev-0.0.0").Return(errors.New("some-error")),
					mockDriver.EXPECT().PowerOffVM("pcfdev-0.0.1"),
					mockDriver.EXPECT().DestroyVM("pcfdev-0.0.1"),
					mockDriver.EXPECT().VMs().Return([]string{"pcfdev-0.0.0"}, nil),
				)

				Expect(vbx.DestroyPCFDevVMs()).To(MatchError("failed to destroy all pcfdev vms"))
			})
		})

		Context("when re-getting vms fails", func() {
			It("shoudl return an error", func() {
				gomock.InOrder(
					mockDriver.EXPECT().VMs().Return([]string{"pcfdev-0.0.1"}, nil),
					mockDriver.EXPECT().PowerOffVM("pcfdev-0.0.1"),
					mockDriver.EXPECT().DestroyVM("pcfdev-0.0.1"),
					mockDriver.EXPECT().VMs().Return(nil, errors.New("some-error")),
				)

				Expect(vbx.DestroyPCFDevVMs()).To(MatchError("some-error"))
			})
		})
	})
})

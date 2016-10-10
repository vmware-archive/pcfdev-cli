package vbox_test

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
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
			PCFDevHome:         "some-pcfdev-home",
			OVADir:             "some-ova-dir",
			VMDir:              "some-vm-dir",
			HTTPProxy:          "some-http-proxy",
			HTTPSProxy:         "some-https-proxy",
			NoProxy:            "some-no-proxy",
			InsecurePrivateKey: []byte("some-insecure-private-key"),
			PrivateKeyPath:     "some-private-key-path",

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
						Name:   "some-used-vbox-interface",
						IP:     "some-used-ip",
						Exists: true,
					},
					&network.Interface{
						Name:   "some-other-used-vbox-interface",
						IP:     "some-other-used-ip",
						Exists: true,
					},
				}
				newInterface := &config.NetworkConfig{
					VMIP:     "some-vm-ip",
					VMDomain: "some-vm-domain",
					Interface: &network.Interface{
						IP:     "some-unused-ip",
						Exists: false,
					},
				}
				vmConfig := &config.VMConfig{
					Name:    "some-vm",
					Memory:  uint64(2000),
					CPUs:    7,
					OVAPath: "some-ova-path",
				}
				gomock.InOrder(
					mockDriver.EXPECT().CreateVM("some-vm", "some-vm-dir"),
					mockFS.EXPECT().Extract("some-ova-path", filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed"), `\w+\.vmdk`),
					mockDriver.EXPECT().CloneDisk(filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed"), filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().DeleteDisk(filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed")),
					mockDriver.EXPECT().AttachDisk("some-vm", filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return(vboxnets, nil),
					mockPicker.EXPECT().SelectAvailableInterface(vboxnets, vmConfig).Return(newInterface, nil),
					mockDriver.EXPECT().CreateHostOnlyInterface("some-unused-ip").Return("some-interface", nil),
					mockDriver.EXPECT().AttachNetworkInterface("some-interface", "some-vm"),
					mockFS.EXPECT().Write(filepath.Join("some-vm-dir", "vm_config"), strings.NewReader(`{"ip":"some-vm-ip","domain":"some-vm-domain"}`), false),
					mockDriver.EXPECT().UseDNSProxy("some-vm"),
					mockSSH.EXPECT().GenerateAddress().Return("some-host", "some-port", nil),
					mockDriver.EXPECT().ForwardPort("some-vm", "ssh", "some-port", "22"),
					mockDriver.EXPECT().SetCPUs("some-vm", 7),
					mockDriver.EXPECT().SetMemory("some-vm", uint64(2000)),
				)
				Expect(vbx.ImportVM(vmConfig)).To(Succeed())
			})
		})

		Context("when there are unused VBox interfaces", func() {
			It("should attach the first unused interface", func() {
				vboxnets := []*network.Interface{
					&network.Interface{
						Name:   "some-unused-vbox-interface",
						IP:     "some-unused-ip",
						Exists: true,
					},
					&network.Interface{
						Name:   "some-other-unused-vbox-interface",
						IP:     "some-other-unused-ip",
						Exists: true,
					},
				}
				unusedVBoxInterface := &config.NetworkConfig{
					VMIP:      "some-vm-ip",
					VMDomain:  "some-vm-domain",
					Interface: vboxnets[0],
				}
				vmConfig := &config.VMConfig{
					Name:    "some-vm",
					OVAPath: "some-ova-path",
					CPUs:    7,
					Memory:  uint64(2000),
				}
				gomock.InOrder(
					mockDriver.EXPECT().CreateVM("some-vm", "some-vm-dir"),
					mockFS.EXPECT().Extract("some-ova-path", filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed"), `\w+\.vmdk`),
					mockDriver.EXPECT().CloneDisk(filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed"), filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().DeleteDisk(filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed")),
					mockDriver.EXPECT().AttachDisk("some-vm", filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return(vboxnets, nil),
					mockPicker.EXPECT().SelectAvailableInterface(vboxnets, vmConfig).Return(unusedVBoxInterface, nil),
					mockDriver.EXPECT().ConfigureHostOnlyInterface("some-unused-vbox-interface", "some-unused-ip"),
					mockDriver.EXPECT().AttachNetworkInterface("some-unused-vbox-interface", "some-vm"),
					mockFS.EXPECT().Write(filepath.Join("some-vm-dir", "vm_config"), strings.NewReader(`{"ip":"some-vm-ip","domain":"some-vm-domain"}`), false),
					mockDriver.EXPECT().UseDNSProxy("some-vm"),
					mockSSH.EXPECT().GenerateAddress().Return("some-host", "some-port", nil),
					mockDriver.EXPECT().ForwardPort("some-vm", "ssh", "some-port", "22"),
					mockDriver.EXPECT().SetCPUs("some-vm", 7),
					mockDriver.EXPECT().SetMemory("some-vm", uint64(2000)),
				)
				Expect(vbx.ImportVM(vmConfig)).To(Succeed())
			})
		})

		Context("when the IP is specified", func() {
			It("should attach the first unused interface", func() {
				vboxnets := []*network.Interface{
					&network.Interface{
						Name:   "some-unused-vbox-interface",
						IP:     "some-unused-ip",
						Exists: true,
					},
					&network.Interface{
						Name:   "some-other-unused-vbox-interface",
						IP:     "some-other-unused-ip",
						Exists: true,
					},
				}
				unusedVBoxInterface := &config.NetworkConfig{
					VMIP:      "some-vm-ip",
					VMDomain:  "some-vm-domain",
					Interface: vboxnets[0],
				}
				vmConfig := &config.VMConfig{
					Name:    "some-vm",
					OVAPath: "some-ova-path",
					CPUs:    7,
					Memory:  uint64(2000),
				}
				gomock.InOrder(
					mockDriver.EXPECT().CreateVM("some-vm", "some-vm-dir"),
					mockFS.EXPECT().Extract("some-ova-path", filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed"), `\w+\.vmdk`),
					mockDriver.EXPECT().CloneDisk(filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed"), filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().DeleteDisk(filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed")),
					mockDriver.EXPECT().AttachDisk("some-vm", filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return(vboxnets, nil),
					mockPicker.EXPECT().SelectAvailableInterface(vboxnets, vmConfig).Return(unusedVBoxInterface, nil),
					mockDriver.EXPECT().ConfigureHostOnlyInterface("some-unused-vbox-interface", "some-unused-ip"),
					mockDriver.EXPECT().AttachNetworkInterface("some-unused-vbox-interface", "some-vm"),
					mockFS.EXPECT().Write(filepath.Join("some-vm-dir", "vm_config"), strings.NewReader(`{"ip":"some-vm-ip","domain":"some-vm-domain"}`), false),
					mockDriver.EXPECT().UseDNSProxy("some-vm"),
					mockSSH.EXPECT().GenerateAddress().Return("some-host", "some-port", nil),
					mockDriver.EXPECT().ForwardPort("some-vm", "ssh", "some-port", "22"),
					mockDriver.EXPECT().SetCPUs("some-vm", 7),
					mockDriver.EXPECT().SetMemory("some-vm", uint64(2000)),
				)
				Expect(vbx.ImportVM(vmConfig)).To(Succeed())
			})
		})

		Context("when extracting the file returns an error", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockDriver.EXPECT().CreateVM("some-vm", "some-vm-dir"),
					mockFS.EXPECT().Extract("some-ova-path", filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed"), `\w+\.vmdk`).Return(errors.New("some-error")),
				)
				Expect(vbx.ImportVM(
					&config.VMConfig{
						Name:    "some-vm",
						OVAPath: "some-ova-path",
					})).To(MatchError("some-error"))
			})
		})

		Context("when cloning the disk fails", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockDriver.EXPECT().CreateVM("some-vm", "some-vm-dir"),
					mockFS.EXPECT().Extract("some-ova-path", filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed"), `\w+\.vmdk`),
					mockDriver.EXPECT().CloneDisk(filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed"), filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")).Return(errors.New("some-error")),
				)
				Expect(vbx.ImportVM(
					&config.VMConfig{
						Name:    "some-vm",
						OVAPath: "some-ova-path",
					})).To(MatchError("some-error"))
			})
		})

		Context("when removing the compressed disk fails", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockDriver.EXPECT().CreateVM("some-vm", "some-vm-dir"),
					mockFS.EXPECT().Extract("some-ova-path", filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed"), `\w+\.vmdk`),
					mockDriver.EXPECT().CloneDisk(filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed"), filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().DeleteDisk(filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed")).Return(errors.New("some-error")),
				)
				Expect(vbx.ImportVM(
					&config.VMConfig{
						Name:    "some-vm",
						OVAPath: "some-ova-path",
					})).To(MatchError("some-error"))
			})
		})

		Context("when attaching the disk fails", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockDriver.EXPECT().CreateVM("some-vm", "some-vm-dir"),
					mockFS.EXPECT().Extract("some-ova-path", filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed"), `\w+\.vmdk`),
					mockDriver.EXPECT().CloneDisk(filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed"), filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().DeleteDisk(filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed")),
					mockDriver.EXPECT().AttachDisk("some-vm", filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")).Return(errors.New("some-error")),
				)
				Expect(vbx.ImportVM(
					&config.VMConfig{
						Name:    "some-vm",
						OVAPath: "some-ova-path",
					})).To(MatchError("some-error"))
			})
		})

		Context("when geting vbox host-only interfaces fails", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockDriver.EXPECT().CreateVM("some-vm", "some-vm-dir"),
					mockFS.EXPECT().Extract("some-ova-path", filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed"), `\w+\.vmdk`),
					mockDriver.EXPECT().CloneDisk(filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed"), filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().DeleteDisk(filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed")),
					mockDriver.EXPECT().AttachDisk("some-vm", filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return([]*network.Interface{}, errors.New("some-error")),
				)
				Expect(vbx.ImportVM(&config.VMConfig{
					Name:    "some-vm",
					OVAPath: "some-ova-path",
					Memory:  uint64(2000),
					CPUs:    7,
				})).To(MatchError("some-error"))
			})
		})

		Context("when selecting an available IP fails", func() {
			It("should return an error", func() {
				vboxnets := []*network.Interface{
					&network.Interface{
						Name:   "some-used-vbox-interface",
						IP:     "some-used-ip",
						Exists: true,
					},
				}
				vmConfig := &config.VMConfig{
					Name:    "some-vm",
					OVAPath: "some-ova-path",
					Memory:  uint64(2000),
					CPUs:    7,
				}
				gomock.InOrder(
					mockDriver.EXPECT().CreateVM("some-vm", "some-vm-dir"),
					mockFS.EXPECT().Extract("some-ova-path", filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed"), `\w+\.vmdk`),
					mockDriver.EXPECT().CloneDisk(filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed"), filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().DeleteDisk(filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed")),
					mockDriver.EXPECT().AttachDisk("some-vm", filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return(vboxnets, nil),
					mockPicker.EXPECT().SelectAvailableInterface(vboxnets, vmConfig).Return(nil, errors.New("some-error")),
				)
				Expect(vbx.ImportVM(vmConfig)).To(MatchError("some-error"))
			})
		})

		Context("when creating a host-only interface fails", func() {
			It("should return an error", func() {
				vboxnets := []*network.Interface{
					&network.Interface{
						Name:   "some-used-vbox-interface",
						IP:     "some-used-ip",
						Exists: true,
					},
				}
				vmConfig := &config.VMConfig{
					Name:    "some-vm",
					OVAPath: "some-ova-path",
					Memory:  uint64(2000),
					CPUs:    7,
				}
				gomock.InOrder(
					mockDriver.EXPECT().CreateVM("some-vm", "some-vm-dir"),
					mockFS.EXPECT().Extract("some-ova-path", filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed"), `\w+\.vmdk`),
					mockDriver.EXPECT().CloneDisk(filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed"), filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().DeleteDisk(filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed")),
					mockDriver.EXPECT().AttachDisk("some-vm", filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return(vboxnets, nil),
					mockPicker.EXPECT().SelectAvailableInterface(vboxnets, vmConfig).Return(&config.NetworkConfig{VMIP: "some-vm-ip", VMDomain: "some-vm-domain", Interface: &network.Interface{IP: "some-unused-ip", Exists: false}}, nil),
					mockDriver.EXPECT().CreateHostOnlyInterface("some-unused-ip").Return("", errors.New("some-error")),
				)
				Expect(vbx.ImportVM(vmConfig)).To(MatchError("some-error"))
			})
		})

		Context("when configuring a host-only interface fails", func() {
			It("should return an error", func() {
				vboxnets := []*network.Interface{
					&network.Interface{
						Name:   "some-unused-vbox-interface",
						IP:     "some-unused-ip",
						Exists: true,
					},
				}
				unusedVBoxInterface := &config.NetworkConfig{
					VMIP:      "some-vm-ip",
					VMDomain:  "some-vm-domain",
					Interface: vboxnets[0],
				}
				vmConfig := &config.VMConfig{
					Name:    "some-vm",
					OVAPath: "some-ova-path",
					Memory:  uint64(2000),
					CPUs:    7,
				}
				gomock.InOrder(
					mockDriver.EXPECT().CreateVM("some-vm", "some-vm-dir"),
					mockFS.EXPECT().Extract("some-ova-path", filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed"), `\w+\.vmdk`),
					mockDriver.EXPECT().CloneDisk(filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed"), filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().DeleteDisk(filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed")),
					mockDriver.EXPECT().AttachDisk("some-vm", filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return(vboxnets, nil),
					mockPicker.EXPECT().SelectAvailableInterface(vboxnets, vmConfig).Return(unusedVBoxInterface, nil),
					mockDriver.EXPECT().ConfigureHostOnlyInterface("some-unused-vbox-interface", "some-unused-ip").Return(errors.New("some-error")),
				)
				Expect(vbx.ImportVM(vmConfig)).To(MatchError("some-error"))
			})
		})

		Context("when attaching an interface fails", func() {
			It("should return an error", func() {
				vboxnets := []*network.Interface{
					&network.Interface{
						Name:   "some-used-vbox-interface",
						IP:     "some-used-ip",
						Exists: true,
					},
				}
				vmConfig := &config.VMConfig{
					Name:    "some-vm",
					OVAPath: "some-ova-path",
					Memory:  uint64(2000),
					CPUs:    7,
				}
				gomock.InOrder(
					mockDriver.EXPECT().CreateVM("some-vm", "some-vm-dir"),
					mockFS.EXPECT().Extract("some-ova-path", filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed"), `\w+\.vmdk`),
					mockDriver.EXPECT().CloneDisk(filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed"), filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().DeleteDisk(filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed")),
					mockDriver.EXPECT().AttachDisk("some-vm", filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return(vboxnets, nil),
					mockPicker.EXPECT().SelectAvailableInterface(vboxnets, vmConfig).Return(&config.NetworkConfig{VMIP: "some-vm-ip", VMDomain: "some-vm-domain", Interface: &network.Interface{IP: "some-unused-ip", Exists: false}}, nil),
					mockDriver.EXPECT().CreateHostOnlyInterface("some-unused-ip").Return("some-interface", nil),
					mockDriver.EXPECT().AttachNetworkInterface("some-interface", "some-vm").Return(errors.New("some-error")),
				)
				Expect(vbx.ImportVM(vmConfig)).To(MatchError("some-error"))
			})
		})

		Context("when writing the domain fails", func() {
			It("should return an error", func() {
				vboxnets := []*network.Interface{
					&network.Interface{
						Name:   "some-used-vbox-interface",
						IP:     "some-used-ip",
						Exists: true,
					},
				}
				vmConfig := &config.VMConfig{
					Name:    "some-vm",
					OVAPath: "some-ova-path",
					Memory:  uint64(2000),
					CPUs:    7,
				}
				gomock.InOrder(
					mockDriver.EXPECT().CreateVM("some-vm", "some-vm-dir"),
					mockFS.EXPECT().Extract("some-ova-path", filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed"), `\w+\.vmdk`),
					mockDriver.EXPECT().CloneDisk(filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed"), filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().DeleteDisk(filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed")),
					mockDriver.EXPECT().AttachDisk("some-vm", filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return(vboxnets, nil),
					mockPicker.EXPECT().SelectAvailableInterface(vboxnets, vmConfig).Return(&config.NetworkConfig{VMIP: "some-vm-ip", VMDomain: "some-vm-domain", Interface: &network.Interface{IP: "some-unused-ip", Exists: false}}, nil),
					mockDriver.EXPECT().CreateHostOnlyInterface("some-unused-ip").Return("some-interface", nil),
					mockDriver.EXPECT().AttachNetworkInterface("some-interface", "some-vm"),
					mockFS.EXPECT().Write(filepath.Join("some-vm-dir", "vm_config"), strings.NewReader(`{"ip":"some-vm-ip","domain":"some-vm-domain"}`), false).Return(errors.New("some-error")),
				)

				Expect(vbx.ImportVM(vmConfig)).To(MatchError("some-error"))
			})
		})

		Context("when using dns proxy fails", func() {
			It("should return an error", func() {
				vboxnets := []*network.Interface{
					&network.Interface{
						Name:   "some-used-vbox-interface",
						IP:     "some-used-ip",
						Exists: true,
					},
				}
				vmConfig := &config.VMConfig{
					Name:    "some-vm",
					OVAPath: "some-ova-path",
					Memory:  uint64(2000),
					CPUs:    7,
				}
				gomock.InOrder(
					mockDriver.EXPECT().CreateVM("some-vm", "some-vm-dir"),
					mockFS.EXPECT().Extract("some-ova-path", filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed"), `\w+\.vmdk`),
					mockDriver.EXPECT().CloneDisk(filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed"), filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().DeleteDisk(filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed")),
					mockDriver.EXPECT().AttachDisk("some-vm", filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return(vboxnets, nil),
					mockPicker.EXPECT().SelectAvailableInterface(vboxnets, vmConfig).Return(&config.NetworkConfig{VMIP: "some-vm-ip", VMDomain: "some-vm-domain", Interface: &network.Interface{IP: "some-unused-ip", Exists: false}}, nil),
					mockDriver.EXPECT().CreateHostOnlyInterface("some-unused-ip").Return("some-interface", nil),
					mockDriver.EXPECT().AttachNetworkInterface("some-interface", "some-vm"),
					mockFS.EXPECT().Write(filepath.Join("some-vm-dir", "vm_config"), strings.NewReader(`{"ip":"some-vm-ip","domain":"some-vm-domain"}`), false),
					mockDriver.EXPECT().UseDNSProxy("some-vm").Return(errors.New("some-error")),
				)

				Expect(vbx.ImportVM(vmConfig)).To(MatchError("some-error"))
			})
		})

		Context("when generating an address fails", func() {
			It("should return an error", func() {
				vboxnets := []*network.Interface{
					&network.Interface{
						Name:   "some-used-vbox-interface",
						IP:     "some-used-ip",
						Exists: true,
					},
				}
				vmConfig := &config.VMConfig{
					Name:    "some-vm",
					OVAPath: "some-ova-path",
					Memory:  uint64(2000),
					CPUs:    7,
				}
				gomock.InOrder(
					mockDriver.EXPECT().CreateVM("some-vm", "some-vm-dir"),
					mockFS.EXPECT().Extract("some-ova-path", filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed"), `\w+\.vmdk`),
					mockDriver.EXPECT().CloneDisk(filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed"), filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().DeleteDisk(filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed")),
					mockDriver.EXPECT().AttachDisk("some-vm", filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return(vboxnets, nil),
					mockPicker.EXPECT().SelectAvailableInterface(vboxnets, vmConfig).Return(&config.NetworkConfig{VMIP: "some-vm-ip", VMDomain: "some-vm-domain", Interface: &network.Interface{IP: "some-unused-ip", Exists: false}}, nil),
					mockDriver.EXPECT().CreateHostOnlyInterface("some-unused-ip").Return("some-interface", nil),
					mockDriver.EXPECT().AttachNetworkInterface("some-interface", "some-vm"),
					mockFS.EXPECT().Write(filepath.Join("some-vm-dir", "vm_config"), strings.NewReader(`{"ip":"some-vm-ip","domain":"some-vm-domain"}`), false),
					mockDriver.EXPECT().UseDNSProxy("some-vm"),
					mockSSH.EXPECT().GenerateAddress().Return("", "", errors.New("some-error")),
				)

				Expect(vbx.ImportVM(vmConfig)).To(MatchError("some-error"))
			})
		})

		Context("when port fowarding fails", func() {
			It("should return an error", func() {
				vboxnets := []*network.Interface{
					&network.Interface{
						Name:   "some-used-vbox-interface",
						IP:     "some-used-ip",
						Exists: true,
					},
				}
				vmConfig := &config.VMConfig{
					Name:    "some-vm",
					OVAPath: "some-ova-path",
					Memory:  uint64(2000),
					CPUs:    7,
				}
				gomock.InOrder(
					mockDriver.EXPECT().CreateVM("some-vm", "some-vm-dir"),
					mockFS.EXPECT().Extract("some-ova-path", filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed"), `\w+\.vmdk`),
					mockDriver.EXPECT().CloneDisk(filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed"), filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().DeleteDisk(filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed")),
					mockDriver.EXPECT().AttachDisk("some-vm", filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return(vboxnets, nil),
					mockPicker.EXPECT().SelectAvailableInterface(vboxnets, vmConfig).Return(&config.NetworkConfig{VMIP: "some-vm-ip", VMDomain: "some-vm-domain", Interface: &network.Interface{IP: "some-unused-ip", Exists: false}}, nil),
					mockDriver.EXPECT().CreateHostOnlyInterface("some-unused-ip").Return("some-interface", nil),
					mockDriver.EXPECT().AttachNetworkInterface("some-interface", "some-vm"),
					mockFS.EXPECT().Write(filepath.Join("some-vm-dir", "vm_config"), strings.NewReader(`{"ip":"some-vm-ip","domain":"some-vm-domain"}`), false),
					mockDriver.EXPECT().UseDNSProxy("some-vm"),
					mockSSH.EXPECT().GenerateAddress().Return("some-host", "some-port", nil),
					mockDriver.EXPECT().ForwardPort("some-vm", "ssh", "some-port", "22").Return(errors.New("some-error")),
				)
				Expect(vbx.ImportVM(vmConfig)).To(MatchError("some-error"))
			})
		})

		Context("when setting the CPUs returns an error", func() {
			It("should return an error", func() {
				vboxnets := []*network.Interface{
					&network.Interface{
						Name:   "some-used-vbox-interface",
						IP:     "some-used-ip",
						Exists: true,
					},
				}
				vmConfig := &config.VMConfig{
					Name:    "some-vm",
					OVAPath: "some-ova-path",
					Memory:  uint64(2000),
					CPUs:    7,
				}
				gomock.InOrder(
					mockDriver.EXPECT().CreateVM("some-vm", "some-vm-dir"),
					mockFS.EXPECT().Extract("some-ova-path", filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed"), `\w+\.vmdk`),
					mockDriver.EXPECT().CloneDisk(filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed"), filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().DeleteDisk(filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed")),
					mockDriver.EXPECT().AttachDisk("some-vm", filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return(vboxnets, nil),
					mockPicker.EXPECT().SelectAvailableInterface(vboxnets, vmConfig).Return(&config.NetworkConfig{VMIP: "some-vm-ip", VMDomain: "some-vm-domain", Interface: &network.Interface{IP: "some-unused-ip", Exists: false}}, nil),
					mockDriver.EXPECT().CreateHostOnlyInterface("some-unused-ip").Return("some-interface", nil),
					mockDriver.EXPECT().AttachNetworkInterface("some-interface", "some-vm"),
					mockFS.EXPECT().Write(filepath.Join("some-vm-dir", "vm_config"), strings.NewReader(`{"ip":"some-vm-ip","domain":"some-vm-domain"}`), false),
					mockDriver.EXPECT().UseDNSProxy("some-vm"),
					mockSSH.EXPECT().GenerateAddress().Return("some-host", "some-port", nil),
					mockDriver.EXPECT().ForwardPort("some-vm", "ssh", "some-port", "22"),
					mockDriver.EXPECT().SetCPUs("some-vm", 7).Return(errors.New("some-error")),
				)
				Expect(vbx.ImportVM(vmConfig)).To(MatchError("some-error"))
			})
		})

		Context("when setting the memory returns an error", func() {
			It("should return an error", func() {
				vboxnets := []*network.Interface{
					&network.Interface{
						Name:   "some-used-vbox-interface",
						IP:     "some-used-ip",
						Exists: true,
					},
				}
				vmConfig := &config.VMConfig{
					Name:    "some-vm",
					OVAPath: "some-ova-path",
					Memory:  uint64(2000),
					CPUs:    7,
				}
				gomock.InOrder(
					mockDriver.EXPECT().CreateVM("some-vm", "some-vm-dir"),
					mockFS.EXPECT().Extract("some-ova-path", filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed"), `\w+\.vmdk`),
					mockDriver.EXPECT().CloneDisk(filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed"), filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().DeleteDisk(filepath.Join("some-vm-dir", "some-vm-disk1.vmdk.compressed")),
					mockDriver.EXPECT().AttachDisk("some-vm", filepath.Join("some-vm-dir", "some-vm", "some-vm-disk1.vmdk")),
					mockDriver.EXPECT().GetHostOnlyInterfaces().Return(vboxnets, nil),
					mockPicker.EXPECT().SelectAvailableInterface(vboxnets, vmConfig).Return(&config.NetworkConfig{VMIP: "some-vm-ip", VMDomain: "some-vm-domain", Interface: &network.Interface{IP: "some-unused-ip", Exists: false}}, nil),
					mockDriver.EXPECT().CreateHostOnlyInterface("some-unused-ip").Return("some-interface", nil),
					mockDriver.EXPECT().AttachNetworkInterface("some-interface", "some-vm"),
					mockFS.EXPECT().Write(filepath.Join("some-vm-dir", "vm_config"), strings.NewReader(`{"ip":"some-vm-ip","domain":"some-vm-domain"}`), false),
					mockDriver.EXPECT().UseDNSProxy("some-vm"),
					mockSSH.EXPECT().GenerateAddress().Return("some-host", "some-port", nil),
					mockDriver.EXPECT().ForwardPort("some-vm", "ssh", "some-port", "22"),
					mockDriver.EXPECT().SetCPUs("some-vm", 7),
					mockDriver.EXPECT().SetMemory("some-vm", uint64(2000)).Return(errors.New("some-error")),
				)
				Expect(vbx.ImportVM(vmConfig)).To(MatchError("some-error"))
			})
		})
	})

	Describe("#StartVM", func() {
		Context("when VM is already imported", func() {
			It("starts without reimporting", func() {
				gomock.InOrder(
					mockDriver.EXPECT().StartVM("some-vm"),
					mockSSH.EXPECT().GenerateKeypair().Return([]byte("some-private-key"), []byte("some-public-key"), nil),
					mockSSH.EXPECT().RunSSHCommand(`echo -n "some-public-key" > /home/vcap/.ssh/authorized_keys`, "127.0.0.1", "some-port", []byte("some-insecure-private-key"), 5*time.Minute, ioutil.Discard, ioutil.Discard),
					mockFS.EXPECT().Write("some-private-key-path", bytes.NewReader([]byte("some-private-key")), false),
					mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
					mockSSH.EXPECT().RunSSHCommand(`echo -e '
auto lo
iface lo inet loopback

auto eth0
iface eth0 inet dhcp

auto eth1
iface eth1 inet static
address 192.168.22.11
netmask 255.255.255.0' | sudo tee /etc/network/interfaces`, "127.0.0.1", "some-port", []byte("some-private-key"), 5*time.Minute, ioutil.Discard, ioutil.Discard),
					mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
					mockSSH.EXPECT().RunSSHCommand(`echo -e '
PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/games:/usr/local/games
HTTP_PROXY=some-http-proxy
HTTPS_PROXY=some-https-proxy
NO_PROXY=localhost,127.0.0.1,192.168.22.1,192.168.22.11,local2.pcfdev.io,.local2.pcfdev.io,some-no-proxy
http_proxy=some-http-proxy
https_proxy=some-https-proxy
no_proxy=localhost,127.0.0.1,192.168.22.1,192.168.22.11,local2.pcfdev.io,.local2.pcfdev.io,some-no-proxy' | sudo tee /etc/environment`,
						"127.0.0.1",
						"some-port",
						[]byte("some-private-key"),
						5*time.Minute,
						ioutil.Discard,
						ioutil.Discard),
					mockDriver.EXPECT().StopVM("some-vm"),
					mockDriver.EXPECT().StartVM("some-vm"),
				)

				Expect(vbx.StartVM(&config.VMConfig{
					Name:    "some-vm",
					IP:      "192.168.22.11",
					SSHPort: "some-port",
					Domain:  "local2.pcfdev.io",
				})).To(Succeed())
			})

			It("translates 127.0.0.1 to subnetIP in proxy settings", func() {
				conf.HTTPProxy = "127.0.0.1"
				conf.HTTPSProxy = "127.0.0.1:8080"

				gomock.InOrder(
					mockDriver.EXPECT().StartVM("some-vm"),
					mockSSH.EXPECT().GenerateKeypair().Return([]byte("some-private-key"), []byte("some-public-key"), nil),
					mockSSH.EXPECT().RunSSHCommand(`echo -n "some-public-key" > /home/vcap/.ssh/authorized_keys`, "127.0.0.1", "some-port", []byte("some-insecure-private-key"), 5*time.Minute, ioutil.Discard, ioutil.Discard),
					mockFS.EXPECT().Write("some-private-key-path", bytes.NewReader([]byte("some-private-key")), false),
					mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
					mockSSH.EXPECT().RunSSHCommand(`echo -e '
auto lo
iface lo inet loopback

auto eth0
iface eth0 inet dhcp

auto eth1
iface eth1 inet static
address 192.168.22.11
netmask 255.255.255.0' | sudo tee /etc/network/interfaces`, "127.0.0.1", "some-port", []byte("some-private-key"), 5*time.Minute, ioutil.Discard, ioutil.Discard),
					mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
					mockSSH.EXPECT().RunSSHCommand(`echo -e '
PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/games:/usr/local/games
HTTP_PROXY=192.168.22.1
HTTPS_PROXY=192.168.22.1:8080
NO_PROXY=localhost,127.0.0.1,192.168.22.1,192.168.22.11,local2.pcfdev.io,.local2.pcfdev.io,some-no-proxy
http_proxy=192.168.22.1
https_proxy=192.168.22.1:8080
no_proxy=localhost,127.0.0.1,192.168.22.1,192.168.22.11,local2.pcfdev.io,.local2.pcfdev.io,some-no-proxy' | sudo tee /etc/environment`,
						"127.0.0.1",
						"some-port",
						[]byte("some-private-key"),
						5*time.Minute,
						ioutil.Discard,
						ioutil.Discard),
					mockDriver.EXPECT().StopVM("some-vm"),
					mockDriver.EXPECT().StartVM("some-vm"),
				)

				Expect(vbx.StartVM(&config.VMConfig{
					Name:    "some-vm",
					IP:      "192.168.22.11",
					SSHPort: "some-port",
					Domain:  "local2.pcfdev.io",
				})).To(Succeed())
			})

			Context("when a bad ip is passed to StartVM command", func() {
				It("should return an error", func() {
					gomock.InOrder(
						mockDriver.EXPECT().StartVM("some-vm"),
						mockSSH.EXPECT().GenerateKeypair().Return([]byte("some-private-key"), []byte("some-public-key"), nil),
						mockSSH.EXPECT().RunSSHCommand(`echo -n "some-public-key" > /home/vcap/.ssh/authorized_keys`, "127.0.0.1", "some-port", []byte("some-insecure-private-key"), 5*time.Minute, ioutil.Discard, ioutil.Discard),
						mockFS.EXPECT().Write("some-private-key-path", bytes.NewReader([]byte("some-private-key")), false),
						mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
						mockSSH.EXPECT().RunSSHCommand(`echo -e '
auto lo
iface lo inet loopback

auto eth0
iface eth0 inet dhcp

auto eth1
iface eth1 inet static
address some-bad-ip
netmask 255.255.255.0' | sudo tee /etc/network/interfaces`, "127.0.0.1", "some-port", []byte("some-private-key"), 5*time.Minute, ioutil.Discard, ioutil.Discard),
					)

					Expect(vbx.StartVM(&config.VMConfig{
						Name:    "some-vm",
						IP:      "some-bad-ip",
						SSHPort: "some-port",
						Domain:  "some-domain",
					})).To(MatchError("some-bad-ip is not a supported IP address"))
				})
			})

			Context("when the http proxy field is empty", func() {
				It("should not appear in the environment file", func() {
					conf.HTTPProxy = ""
					conf.HTTPSProxy = "127.0.0.1"
					gomock.InOrder(
						mockDriver.EXPECT().StartVM("some-vm"),
						mockSSH.EXPECT().GenerateKeypair().Return([]byte("some-private-key"), []byte("some-public-key"), nil),
						mockSSH.EXPECT().RunSSHCommand(`echo -n "some-public-key" > /home/vcap/.ssh/authorized_keys`, "127.0.0.1", "some-port", []byte("some-insecure-private-key"), 5*time.Minute, ioutil.Discard, ioutil.Discard),
						mockFS.EXPECT().Write("some-private-key-path", bytes.NewReader([]byte("some-private-key")), false),
						mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
						mockSSH.EXPECT().RunSSHCommand(`echo -e '
auto lo
iface lo inet loopback

auto eth0
iface eth0 inet dhcp

auto eth1
iface eth1 inet static
address 192.168.22.11
netmask 255.255.255.0' | sudo tee /etc/network/interfaces`, "127.0.0.1", "some-port", []byte("some-private-key"), 5*time.Minute, ioutil.Discard, ioutil.Discard),
						mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
						mockSSH.EXPECT().RunSSHCommand(`echo -e '
PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/games:/usr/local/games

HTTPS_PROXY=192.168.22.1
NO_PROXY=localhost,127.0.0.1,192.168.22.1,192.168.22.11,local2.pcfdev.io,.local2.pcfdev.io,some-no-proxy

https_proxy=192.168.22.1
no_proxy=localhost,127.0.0.1,192.168.22.1,192.168.22.11,local2.pcfdev.io,.local2.pcfdev.io,some-no-proxy' | sudo tee /etc/environment`,
							"127.0.0.1",
							"some-port",
							[]byte("some-private-key"),
							5*time.Minute,
							ioutil.Discard,
							ioutil.Discard),
						mockDriver.EXPECT().StopVM("some-vm"),
						mockDriver.EXPECT().StartVM("some-vm"),
					)

					Expect(vbx.StartVM(&config.VMConfig{
						Name:    "some-vm",
						IP:      "192.168.22.11",
						SSHPort: "some-port",
						Domain:  "local2.pcfdev.io",
					})).To(Succeed())
				})

			})

			Context("when the https proxy field is empty", func() {
				It("should not appear in the environment file", func() {
					conf.HTTPProxy = "127.0.0.1"
					conf.HTTPSProxy = ""
					gomock.InOrder(
						mockDriver.EXPECT().StartVM("some-vm"),
						mockSSH.EXPECT().GenerateKeypair().Return([]byte("some-private-key"), []byte("some-public-key"), nil),
						mockSSH.EXPECT().RunSSHCommand(`echo -n "some-public-key" > /home/vcap/.ssh/authorized_keys`, "127.0.0.1", "some-port", []byte("some-insecure-private-key"), 5*time.Minute, ioutil.Discard, ioutil.Discard),
						mockFS.EXPECT().Write("some-private-key-path", bytes.NewReader([]byte("some-private-key")), false),
						mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
						mockSSH.EXPECT().RunSSHCommand(`echo -e '
auto lo
iface lo inet loopback

auto eth0
iface eth0 inet dhcp

auto eth1
iface eth1 inet static
address 192.168.22.11
netmask 255.255.255.0' | sudo tee /etc/network/interfaces`, "127.0.0.1", "some-port", []byte("some-private-key"), 5*time.Minute, ioutil.Discard, ioutil.Discard),
						mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
						mockSSH.EXPECT().RunSSHCommand(`echo -e '
PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/games:/usr/local/games
HTTP_PROXY=192.168.22.1

NO_PROXY=localhost,127.0.0.1,192.168.22.1,192.168.22.11,local2.pcfdev.io,.local2.pcfdev.io,some-no-proxy
http_proxy=192.168.22.1

no_proxy=localhost,127.0.0.1,192.168.22.1,192.168.22.11,local2.pcfdev.io,.local2.pcfdev.io,some-no-proxy' | sudo tee /etc/environment`,
							"127.0.0.1",
							"some-port",
							[]byte("some-private-key"),
							5*time.Minute,
							ioutil.Discard,
							ioutil.Discard),
						mockDriver.EXPECT().StopVM("some-vm"),
						mockDriver.EXPECT().StartVM("some-vm"),
					)

					Expect(vbx.StartVM(&config.VMConfig{
						Name:    "some-vm",
						IP:      "192.168.22.11",
						SSHPort: "some-port",
						Domain:  "local2.pcfdev.io",
					})).To(Succeed())
				})

			})

			Context("when the no proxy field is empty", func() {
				It("should not have a trailing comma", func() {
					conf.HTTPProxy = "127.0.0.1"
					conf.HTTPSProxy = "127.0.0.1"
					conf.NoProxy = ""
					gomock.InOrder(
						mockDriver.EXPECT().StartVM("some-vm"),
						mockSSH.EXPECT().GenerateKeypair().Return([]byte("some-private-key"), []byte("some-public-key"), nil),
						mockSSH.EXPECT().RunSSHCommand(`echo -n "some-public-key" > /home/vcap/.ssh/authorized_keys`, "127.0.0.1", "some-port", []byte("some-insecure-private-key"), 5*time.Minute, ioutil.Discard, ioutil.Discard),
						mockFS.EXPECT().Write("some-private-key-path", bytes.NewReader([]byte("some-private-key")), false),
						mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
						mockSSH.EXPECT().RunSSHCommand(`echo -e '
auto lo
iface lo inet loopback

auto eth0
iface eth0 inet dhcp

auto eth1
iface eth1 inet static
address 192.168.22.11
netmask 255.255.255.0' | sudo tee /etc/network/interfaces`, "127.0.0.1", "some-port", []byte("some-private-key"), 5*time.Minute, ioutil.Discard, ioutil.Discard),
						mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
						mockSSH.EXPECT().RunSSHCommand(`echo -e '
PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/games:/usr/local/games
HTTP_PROXY=192.168.22.1
HTTPS_PROXY=192.168.22.1
NO_PROXY=localhost,127.0.0.1,192.168.22.1,192.168.22.11,local2.pcfdev.io,.local2.pcfdev.io
http_proxy=192.168.22.1
https_proxy=192.168.22.1
no_proxy=localhost,127.0.0.1,192.168.22.1,192.168.22.11,local2.pcfdev.io,.local2.pcfdev.io' | sudo tee /etc/environment`,
							"127.0.0.1",
							"some-port",
							[]byte("some-private-key"),
							5*time.Minute,
							ioutil.Discard,
							ioutil.Discard),
						mockDriver.EXPECT().StopVM("some-vm"),
						mockDriver.EXPECT().StartVM("some-vm"),
					)

					Expect(vbx.StartVM(&config.VMConfig{
						Name:    "some-vm",
						IP:      "192.168.22.11",
						SSHPort: "some-port",
						Domain:  "local2.pcfdev.io",
					})).To(Succeed())
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

			Context("when there is an error generating a private keypair", func() {
				It("should return the error", func() {
					gomock.InOrder(
						mockDriver.EXPECT().StartVM("some-vm"),
						mockSSH.EXPECT().GenerateKeypair().Return(nil, nil, errors.New("some-error")),
					)

					Expect(vbx.StartVM(&config.VMConfig{
						Name:    "some-vm",
						IP:      "192.168.22.11",
						SSHPort: "some-port",
						Domain:  "some-domain",
					})).To(MatchError("some-error"))
				})
			})

			Context("when there is an error replacing the authorized keys in the ova", func() {
				It("should return the error", func() {
					gomock.InOrder(
						mockDriver.EXPECT().StartVM("some-vm"),
						mockSSH.EXPECT().GenerateKeypair().Return([]byte("some-private-key"), []byte("some-public-key"), nil),
						mockSSH.EXPECT().RunSSHCommand(`echo -n "some-public-key" > /home/vcap/.ssh/authorized_keys`, "127.0.0.1", "some-port", []byte("some-insecure-private-key"), 5*time.Minute, ioutil.Discard, ioutil.Discard).Return(errors.New("some-error")),
					)

					Expect(vbx.StartVM(&config.VMConfig{
						Name:    "some-vm",
						IP:      "192.168.22.11",
						SSHPort: "some-port",
						Domain:  "some-domain",
					})).To(MatchError("some-error"))
				})
			})

			Context("when there is an error writing the secure private key to the .pcfdev dir", func() {
				It("should return the error", func() {
					gomock.InOrder(
						mockDriver.EXPECT().StartVM("some-vm"),
						mockSSH.EXPECT().GenerateKeypair().Return([]byte("some-private-key"), []byte("some-public-key"), nil),
						mockSSH.EXPECT().RunSSHCommand(`echo -n "some-public-key" > /home/vcap/.ssh/authorized_keys`, "127.0.0.1", "some-port", []byte("some-insecure-private-key"), 5*time.Minute, ioutil.Discard, ioutil.Discard),
						mockFS.EXPECT().Write("some-private-key-path", bytes.NewReader([]byte("some-private-key")), false).Return(errors.New("some-error")),
					)

					Expect(vbx.StartVM(&config.VMConfig{
						Name:    "some-vm",
						IP:      "192.168.22.11",
						SSHPort: "some-port",
						Domain:  "some-domain",
					})).To(MatchError("some-error"))
				})
			})

			Context("when there is an error reading the private key", func() {
				It("should return the error", func() {
					gomock.InOrder(
						mockDriver.EXPECT().StartVM("some-vm"),
						mockSSH.EXPECT().GenerateKeypair().Return([]byte("some-private-key"), []byte("some-public-key"), nil),
						mockSSH.EXPECT().RunSSHCommand(`echo -n "some-public-key" > /home/vcap/.ssh/authorized_keys`, "127.0.0.1", "some-port", []byte("some-insecure-private-key"), 5*time.Minute, ioutil.Discard, ioutil.Discard),
						mockFS.EXPECT().Write("some-private-key-path", bytes.NewReader([]byte("some-private-key")), false),
						mockFS.EXPECT().Read("some-private-key-path").Return(nil, errors.New("some-error")),
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
						mockSSH.EXPECT().GenerateKeypair().Return([]byte("some-private-key"), []byte("some-public-key"), nil),
						mockSSH.EXPECT().RunSSHCommand(`echo -n "some-public-key" > /home/vcap/.ssh/authorized_keys`, "127.0.0.1", "some-port", []byte("some-insecure-private-key"), 5*time.Minute, ioutil.Discard, ioutil.Discard),
						mockFS.EXPECT().Write("some-private-key-path", bytes.NewReader([]byte("some-private-key")), false),
						mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
						mockSSH.EXPECT().RunSSHCommand(fmt.Sprintf(`echo -e '
auto lo
iface lo inet loopback

auto eth0
iface eth0 inet dhcp

auto eth1
iface eth1 inet static
address some-ip
netmask 255.255.255.0' | sudo tee /etc/network/interfaces`), "127.0.0.1", "some-port", []byte("some-private-key"), 5*time.Minute, ioutil.Discard, ioutil.Discard).Return(errors.New("some-error")),
					)

					Expect(vbx.StartVM(&config.VMConfig{
						Name:    "some-vm",
						IP:      "some-ip",
						SSHPort: "some-port",
						Domain:  "some-domain",
					})).To(MatchError("some-error"))
				})
			})

			Context("when there is an error reading the private key", func() {
				It("should return an error", func() {
					gomock.InOrder(
						mockDriver.EXPECT().StartVM("some-vm"),
						mockSSH.EXPECT().GenerateKeypair().Return([]byte("some-private-key"), []byte("some-public-key"), nil),
						mockSSH.EXPECT().RunSSHCommand(`echo -n "some-public-key" > /home/vcap/.ssh/authorized_keys`, "127.0.0.1", "some-port", []byte("some-insecure-private-key"), 5*time.Minute, ioutil.Discard, ioutil.Discard),
						mockFS.EXPECT().Write("some-private-key-path", bytes.NewReader([]byte("some-private-key")), false),
						mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
						mockSSH.EXPECT().RunSSHCommand(fmt.Sprintf(`echo -e '
auto lo
iface lo inet loopback

auto eth0
iface eth0 inet dhcp

auto eth1
iface eth1 inet static
address 192.168.22.11
netmask 255.255.255.0' | sudo tee /etc/network/interfaces`), "127.0.0.1", "some-port", []byte("some-private-key"), 5*time.Minute, ioutil.Discard, ioutil.Discard),
						mockFS.EXPECT().Read("some-private-key-path").Return(nil, errors.New("some-error")),
					)

					Expect(vbx.StartVM(&config.VMConfig{
						Name:    "some-vm",
						IP:      "192.168.22.11",
						SSHPort: "some-port",
						Domain:  "some-domain",
					})).To(MatchError("some-error"))
				})
			})

			Context("when VM fails to stop", func() {
				It("should return an error", func() {
					gomock.InOrder(
						mockDriver.EXPECT().StartVM("some-vm"),
						mockSSH.EXPECT().GenerateKeypair().Return([]byte("some-private-key"), []byte("some-public-key"), nil),
						mockSSH.EXPECT().RunSSHCommand(`echo -n "some-public-key" > /home/vcap/.ssh/authorized_keys`, "127.0.0.1", "some-port", []byte("some-insecure-private-key"), 5*time.Minute, ioutil.Discard, ioutil.Discard),
						mockFS.EXPECT().Write("some-private-key-path", bytes.NewReader([]byte("some-private-key")), false),
						mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
						mockSSH.EXPECT().RunSSHCommand(`echo -e '
auto lo
iface lo inet loopback

auto eth0
iface eth0 inet dhcp

auto eth1
iface eth1 inet static
address 192.168.11.11
netmask 255.255.255.0' | sudo tee /etc/network/interfaces`, "127.0.0.1", "some-port", []byte("some-private-key"), 5*time.Minute, ioutil.Discard, ioutil.Discard),
						mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
						mockSSH.EXPECT().RunSSHCommand(`echo -e '
PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/games:/usr/local/games
HTTP_PROXY=some-http-proxy
HTTPS_PROXY=some-https-proxy
NO_PROXY=localhost,127.0.0.1,192.168.11.1,192.168.11.11,local.pcfdev.io,.local.pcfdev.io,some-no-proxy
http_proxy=some-http-proxy
https_proxy=some-https-proxy
no_proxy=localhost,127.0.0.1,192.168.11.1,192.168.11.11,local.pcfdev.io,.local.pcfdev.io,some-no-proxy' | sudo tee /etc/environment`,
							"127.0.0.1",
							"some-port",
							[]byte("some-private-key"),
							5*time.Minute,
							ioutil.Discard,
							ioutil.Discard),
						mockDriver.EXPECT().StopVM("some-vm").Return(errors.New("some-error")),
					)
					Expect(vbx.StartVM(&config.VMConfig{
						Name:    "some-vm",
						IP:      "192.168.11.11",
						SSHPort: "some-port",
						Domain:  "local.pcfdev.io",
					})).To(MatchError("some-error"))
				})
			})
		})
	})

	Describe("#StopVM", func() {
		It("should stop the VM", func() {
			mockDriver.EXPECT().StopVM("some-vm")

			Expect(vbx.StopVM(&config.VMConfig{Name: "some-vm"})).To(Succeed())
		})

		Context("Driver fails to stop VM", func() {
			It("should return the error", func() {
				expectedError := errors.New("some-error")

				mockDriver.EXPECT().StopVM("some-vm").Return(expectedError)
				Expect(vbx.StopVM(&config.VMConfig{Name: "some-vm"})).To(MatchError(expectedError))
			})
		})
	})

	Describe("#SuspendVM", func() {
		It("should suspend the VM", func() {
			mockDriver.EXPECT().SuspendVM("some-vm")

			Expect(vbx.SuspendVM(&config.VMConfig{Name: "some-vm"})).To(Succeed())
		})

		Context("when the Driver fails to suspend the VM", func() {
			It("should return the error", func() {
				expectedError := errors.New("some-error")

				mockDriver.EXPECT().SuspendVM("some-vm").Return(expectedError)
				Expect(vbx.SuspendVM(&config.VMConfig{Name: "some-vm"})).To(MatchError(expectedError))
			})
		})
	})

	Describe("#VMStatus", func() {
		Context("when vm is running", func() {
			It("should return a running status", func() {
				gomock.InOrder(
					mockDriver.EXPECT().VMExists("some-vm").Return(true, nil),
					mockDriver.EXPECT().VMState("some-vm").Return(vbox.StateRunning, nil),
				)
				Expect(vbx.VMStatus("some-vm")).To(Equal(vbox.StatusRunning))
			})

		})

		Context("when vm is stopped", func() {
			It("should return a stopped status", func() {
				gomock.InOrder(
					mockDriver.EXPECT().VMExists("some-vm").Return(true, nil),
					mockDriver.EXPECT().VMState("some-vm").Return(vbox.StateStopped, nil),
				)

				Expect(vbx.VMStatus("some-vm")).To(Equal(vbox.StatusStopped))
			})
		})

		Context("when vm is aborted", func() {
			It("should return a stopped status", func() {
				gomock.InOrder(
					mockDriver.EXPECT().VMExists("some-vm").Return(true, nil),
					mockDriver.EXPECT().VMState("some-vm").Return(vbox.StateAborted, nil),
				)

				Expect(vbx.VMStatus("some-vm")).To(Equal(vbox.StatusStopped))
			})
		})

		Context("when vm is saved", func() {
			It("should return a saved status", func() {
				gomock.InOrder(
					mockDriver.EXPECT().VMExists("some-vm").Return(true, nil),
					mockDriver.EXPECT().VMState("some-vm").Return(vbox.StateSaved, nil),
				)

				Expect(vbx.VMStatus("some-vm")).To(Equal(vbox.StatusSaved))
			})
		})

		Context("when vm is paused", func() {
			It("should return a paused status", func() {
				gomock.InOrder(
					mockDriver.EXPECT().VMExists("some-vm").Return(true, nil),
					mockDriver.EXPECT().VMState("some-vm").Return(vbox.StatePaused, nil),
				)

				Expect(vbx.VMStatus("some-vm")).To(Equal(vbox.StatusPaused))
			})
		})

		Context("when vm is in an unknown state", func() {
			It("should return a status unknown", func() {
				gomock.InOrder(
					mockDriver.EXPECT().VMExists("some-vm").Return(true, nil),
					mockDriver.EXPECT().VMState("some-vm").Return("some-unknown-status", nil),
				)

				Expect(vbx.VMStatus("some-vm")).To(Equal(vbox.StatusUnknown))
			})
		})

		Context("when vm does not exist", func() {
			It("should return a not created status", func() {
				mockDriver.EXPECT().VMExists("some-vm").Return(false, nil)

				Expect(vbx.VMStatus("some-vm")).To(Equal(vbox.StatusNotCreated))
			})
		})

		Context("when there is an error seeing if vm exists", func() {
			It("should return an error", func() {
				mockDriver.EXPECT().VMExists("some-vm").Return(false, errors.New("some-error"))

				status, err := vbx.VMStatus("some-vm")
				Expect(status).To(BeEmpty())
				Expect(err).To(MatchError("some-error"))
			})
		})

		Context("when there is an error retrieving vm state", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockDriver.EXPECT().VMExists("some-vm").Return(true, nil),
					mockDriver.EXPECT().VMState("some-vm").Return("", errors.New("some-error")),
				)

				status, err := vbx.VMStatus("some-vm")
				Expect(status).To(BeEmpty())
				Expect(err).To(MatchError("some-error"))
			})
		})
	})

	Describe("#VMConfig", func() {
		It("should get the vm config", func() {
			gomock.InOrder(
				mockDriver.EXPECT().GetMemory("some-vm").Return(uint64(4000), nil),
				mockDriver.EXPECT().GetHostForwardPort("some-vm", "ssh").Return("some-port", nil),
				mockFS.EXPECT().Read(filepath.Join("some-vm-dir", "vm_config")).Return([]byte(`{"ip":"192.168.22.11","domain":"local2.pcfdev.io"}`), nil),
			)

			Expect(vbx.VMConfig("some-vm")).To(Equal(&config.VMConfig{
				Domain:  "local2.pcfdev.io",
				IP:      "192.168.22.11",
				Memory:  uint64(4000),
				Name:    "some-vm",
				SSHPort: "some-port",
			}))
		})

		Context("when the driver fails to get the memory", func() {
			It("should return an error", func() {
				mockDriver.EXPECT().GetMemory("some-vm").Return(uint64(0), errors.New("some-error"))

				_, err := vbx.VMConfig("some-vm")
				Expect(err).To(MatchError("some-error"))
			})
		})

		Context("when the driver fails to get the SSHPort", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockDriver.EXPECT().GetMemory("some-vm").Return(uint64(4000), nil),
					mockDriver.EXPECT().GetHostForwardPort("some-vm", "ssh").Return("", errors.New("some-error")),
				)

				_, err := vbx.VMConfig("some-vm")
				Expect(err).To(MatchError("some-error"))
			})
		})

		Context("when retrieving the ip and domain fails", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockDriver.EXPECT().GetMemory("some-vm").Return(uint64(4000), nil),
					mockDriver.EXPECT().GetHostForwardPort("some-vm", "ssh").Return("some-port", nil),
					mockFS.EXPECT().Read(filepath.Join("some-vm-dir", "vm_config")).Return(nil, errors.New("some-error")),
				)

				_, err := vbx.VMConfig("some-vm")
				Expect(err).To(MatchError("some-error"))
			})
		})

		Context("when retrieving the vm_config file is not valid json", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockDriver.EXPECT().GetMemory("some-vm").Return(uint64(4000), nil),
					mockDriver.EXPECT().GetHostForwardPort("some-vm", "ssh").Return("some-port", nil),
					mockFS.EXPECT().Read(filepath.Join("some-vm-dir", "vm_config")).Return([]byte(`some-invalid-json`), nil),
				)

				_, err := vbx.VMConfig("some-vm")
				Expect(err).To(MatchError("invalid character 's' looking for beginning of value"))
			})
		})
	})

	Describe("#ResumeSavedVM", func() {
		It("should start the VM", func() {
			mockDriver.EXPECT().StartVM("some-vm")

			Expect(vbx.ResumeSavedVM(&config.VMConfig{Name: "some-vm"})).To(Succeed())
		})

		Context("when the Driver fails to start the VM", func() {
			It("should return the error", func() {
				expectedError := errors.New("some-error")

				mockDriver.EXPECT().StartVM("some-vm").Return(expectedError)
				Expect(vbx.ResumeSavedVM(&config.VMConfig{Name: "some-vm"})).To(MatchError(expectedError))
			})
		})
	})

	Describe("#ResumePausedVM", func() {
		It("should resume the VM", func() {
			mockDriver.EXPECT().ResumeVM("some-vm")

			Expect(vbx.ResumePausedVM(&config.VMConfig{Name: "some-vm"})).To(Succeed())
		})

		Context("when the Driver fails to resume the VM", func() {
			It("should return the error", func() {
				expectedError := errors.New("some-error")

				mockDriver.EXPECT().ResumeVM("some-vm").Return(expectedError)
				Expect(vbx.ResumePausedVM(&config.VMConfig{Name: "some-vm"})).To(MatchError(expectedError))
			})
		})
	})

	Describe("#GetVMName", func() {
		Context("if there is one PCF Dev VM present", func() {
			It("should return the name of that VM", func() {
				mockDriver.EXPECT().VMs().Return([]string{"some-vm-name", "pcfdev-our-vm"}, nil)
				Expect(vbx.GetVMName()).To(Equal("pcfdev-our-vm"))
			})
		})

		Context("if there is more than one PCF Dev VM present", func() {
			It("should return an error", func() {
				mockDriver.EXPECT().VMs().Return([]string{"some-vm-name", "pcfdev-our-vm", "pcfdev-other-vm"}, nil)
				_, err := vbx.GetVMName()
				Expect(err).To(MatchError("multiple PCF Dev VMs found"))
			})
		})
		Context("if Driver.VMs() returns an error", func() {
			It("should return an error", func() {
				mockDriver.EXPECT().VMs().Return(nil, errors.New("some-error"))
				_, err := vbx.GetVMName()
				Expect(err).To(MatchError("some-error"))
			})
		})

		Context("when there are no PCF Dev VMs present", func() {
			It("should return an empty string", func() {
				mockDriver.EXPECT().VMs().Return([]string{"some-vm-name"}, nil)
				Expect(vbx.GetVMName()).To(Equal(""))
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

	Describe("#Version", func() {
		It("return the VBoxDriver version", func() {
			driverVersion := &vbox.VBoxDriverVersion{Major: 1, Minor: 0, Build: 0}
			mockDriver.EXPECT().Version().Return(driverVersion, nil)

			Expect(vbx.Version()).To(Equal(driverVersion))
		})

		Context("when there is an error retrieving the version", func() {
			It("return the error", func() {
				mockDriver.EXPECT().Version().Return(nil, errors.New("some-error"))

				version, err := vbx.Version()
				Expect(version).To(BeNil())
				Expect(err).To(MatchError("some-error"))
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
		It("should destroy VMs and Disks that begin with pcfdev-", func() {
			gomock.InOrder(
				mockDriver.EXPECT().VMs().Return([]string{"pcfdev-0.0.0", "pcfdev-0.0.1", "some-bad-vm-name"}, nil),
				mockDriver.EXPECT().PowerOffVM("pcfdev-0.0.0"),
				mockDriver.EXPECT().DestroyVM("pcfdev-0.0.0"),
				mockDriver.EXPECT().PowerOffVM("pcfdev-0.0.1"),
				mockDriver.EXPECT().DestroyVM("pcfdev-0.0.1"),
				mockDriver.EXPECT().VMs().Return([]string{}, nil),
				mockDriver.EXPECT().Disks().Return([]string{
					filepath.Join("some-dir", "pcfdev-disk1.vmdk"),
					filepath.Join("some-other-dir", "pcfdev-disk1.vmdk"),
					filepath.Join("some-other-dir", "some-other-disk.vmdk"),
				}, nil),
				mockDriver.EXPECT().DeleteDisk(filepath.Join("some-dir", "pcfdev-disk1.vmdk")),
				mockDriver.EXPECT().DeleteDisk(filepath.Join("some-other-dir", "pcfdev-disk1.vmdk")),
				mockDriver.EXPECT().Disks().Return([]string{filepath.Join("some-other-dir", "some-other-disk.vmdk")}, nil),
			)

			Expect(vbx.DestroyPCFDevVMs()).To(Succeed())
		})

		Context("when it fails to retrieve disks", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockDriver.EXPECT().VMs().Return([]string{}, nil),
					mockDriver.EXPECT().VMs().Return([]string{}, nil),
					mockDriver.EXPECT().Disks().Return(nil, errors.New("some-error")),
				)

				Expect(vbx.DestroyPCFDevVMs()).To(MatchError("some-error"))
			})
		})

		Context("when it fails to retrieve disks a second time", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockDriver.EXPECT().VMs().Return([]string{}, nil),
					mockDriver.EXPECT().VMs().Return([]string{}, nil),
					mockDriver.EXPECT().Disks().Return([]string{filepath.Join("some-dir", "pcfdev-disk1.vmdk"), filepath.Join("some-other-dir", "pcfdev-disk1.vmdk")}, nil),
					mockDriver.EXPECT().DeleteDisk(filepath.Join("some-dir", "pcfdev-disk1.vmdk")),
					mockDriver.EXPECT().DeleteDisk(filepath.Join("some-other-dir", "pcfdev-disk1.vmdk")),
					mockDriver.EXPECT().Disks().Return(nil, errors.New("some-error")),
				)

				Expect(vbx.DestroyPCFDevVMs()).To(MatchError("some-error"))
			})
		})

		Context("when it fails to delete all the disks", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockDriver.EXPECT().VMs().Return([]string{}, nil),
					mockDriver.EXPECT().VMs().Return([]string{}, nil),
					mockDriver.EXPECT().Disks().Return([]string{filepath.Join("some-dir", "pcfdev-disk1.vmdk"), filepath.Join("some-other-dir", "pcfdev-disk1.vmdk")}, nil),
					mockDriver.EXPECT().DeleteDisk(filepath.Join("some-dir", "pcfdev-disk1.vmdk")),
					mockDriver.EXPECT().DeleteDisk(filepath.Join("some-other-dir", "pcfdev-disk1.vmdk")).Return(errors.New("some-error")),
					mockDriver.EXPECT().Disks().Return([]string{filepath.Join("some-other-dir", "pcfdev-disk1.vmdk")}, nil),
				)

				Expect(vbx.DestroyPCFDevVMs()).To(MatchError("failed to destroy all pcfdev disks"))
			})
		})

		Context("when getting VMs fails", func() {
			It("should return an error", func() {
				mockDriver.EXPECT().VMs().Return([]string{}, errors.New("some-error"))

				Expect(vbx.DestroyPCFDevVMs()).To(MatchError("some-error"))
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

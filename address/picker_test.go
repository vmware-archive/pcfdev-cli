package address_test

import (
	"errors"
	"fmt"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/pcfdev-cli/address"
	"github.com/pivotal-cf/pcfdev-cli/address/mocks"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/network"
)

var _ = Describe("Picker", func() {
	var (
		picker      *address.Picker
		mockCtrl    *gomock.Controller
		mockNetwork *mocks.MockNetwork
		mockDriver  *mocks.MockDriver
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockNetwork = mocks.NewMockNetwork(mockCtrl)
		mockDriver = mocks.NewMockDriver(mockCtrl)

		picker = &address.Picker{
			Network: mockNetwork,
			Driver:  mockDriver,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("#SelectAvailableInterface", func() {
		Context("when there is a desired ip passed in", func() {
			It("should return an interface with that IP", func() {
				vboxInterfaces := []*network.Interface{}
				expectedNetworkConfig := &config.NetworkConfig{
					VMIP:     "192.168.11.11",
					VMDomain: "local.pcfdev.io",
					Interface: &network.Interface{
						IP:     "192.168.11.1",
						Exists: false,
					},
				}

				Expect(picker.SelectAvailableInterface(vboxInterfaces, &config.VMConfig{
					IP: "192.168.11.11",
				})).To(Equal(expectedNetworkConfig))
			})
		})

		Context("when there is an error determining the subnet of a desired ip", func() {
			It("should return an error", func() {
				vboxInterfaces := []*network.Interface{}
				_, err := picker.SelectAvailableInterface(vboxInterfaces, &config.VMConfig{
					IP:     "some-bad-ip",
					Domain: "some-domain",
				})

				Expect(err).To(MatchError("some-bad-ip is not a supported IP address"))
			})
		})

		Context("when there is a desired invalid ip passed", func() {
			It("should return an error", func() {
				vboxInterfaces := []*network.Interface{}
				_, err := picker.SelectAvailableInterface(vboxInterfaces, &config.VMConfig{
					IP: "some-bad-ip",
				})

				Expect(err).To(MatchError("some-bad-ip is not a supported IP address"))
			})
		})

		Context("when there is a desired ip passed in and it is in use", func() {
			It("should return an interface with the corresponding IP that exists", func() {
				vboxInterfaces := []*network.Interface{
					&network.Interface{
						Name:   "some-net-iface",
						IP:     "192.168.11.1",
						Exists: true,
					},
				}
				expectedNetworkConfig := &config.NetworkConfig{
					VMIP:     "192.168.11.11",
					VMDomain: "local.pcfdev.io",
					Interface: &network.Interface{
						Name:   "some-net-iface",
						IP:     "192.168.11.1",
						Exists: true,
					},
				}

				Expect(picker.SelectAvailableInterface(vboxInterfaces, &config.VMConfig{
					IP: "192.168.11.11",
				})).To(Equal(expectedNetworkConfig))
			})
		})

		Context("when there is a desired PCFDev domain passed in", func() {
			It("should return an interface with the corresponding IP", func() {
				vboxInterfaces := []*network.Interface{}
				expectedNetworkConfig := &config.NetworkConfig{
					VMIP:     "192.168.22.11",
					VMDomain: "local2.pcfdev.io",
					Interface: &network.Interface{
						IP:     "192.168.22.1",
						Exists: false,
					},
				}

				Expect(picker.SelectAvailableInterface(vboxInterfaces, &config.VMConfig{
					Domain: "local2.pcfdev.io",
				})).To(Equal(expectedNetworkConfig))
			})
		})

		Context("when there is a desired PCFDev domain passed in and the corresponding ip already exists", func() {
			It("should return an interface with the corresponding IP that exists", func() {
				vboxInterfaces := []*network.Interface{
					&network.Interface{
						Name:   "some-net-iface",
						IP:     "192.168.22.1",
						Exists: true,
					},
				}
				expectedNetworkConfig := &config.NetworkConfig{
					VMIP:     "192.168.22.11",
					VMDomain: "local2.pcfdev.io",
					Interface: &network.Interface{
						Name:   "some-net-iface",
						IP:     "192.168.22.1",
						Exists: true,
					},
				}

				Expect(picker.SelectAvailableInterface(vboxInterfaces, &config.VMConfig{
					Domain: "local2.pcfdev.io",
				})).To(Equal(expectedNetworkConfig))
			})
		})

		Context("when there is a desired domain and IP passed in", func() {
			It("should return an interface with the corresponding IP that does not exist", func() {
				vboxInterfaces := []*network.Interface{}
				expectedNetworkConfig := &config.NetworkConfig{
					VMIP:     "192.168.99.99",
					VMDomain: "some-domain",
					Interface: &network.Interface{
						IP:     "192.168.99.1",
						Exists: false,
					},
				}

				Expect(picker.SelectAvailableInterface(vboxInterfaces, &config.VMConfig{
					Domain: "some-domain",
					IP:     "192.168.99.99",
				})).To(Equal(expectedNetworkConfig))
			})
		})

		Context("when there is a desired domain and IP passed in and the corresponding ip already exists", func() {
			It("should return an interface with the corresponding IP that exists", func() {
				vboxInterfaces := []*network.Interface{
					&network.Interface{
						Name:   "some-net-iface",
						IP:     "192.168.22.1",
						Exists: true,
					},
				}
				expectedNetworkConfig := &config.NetworkConfig{
					VMIP:     "192.168.22.11",
					VMDomain: "some-domain",
					Interface: &network.Interface{
						IP:     "192.168.22.1",
						Name:   "some-net-iface",
						Exists: true,
					},
				}

				Expect(picker.SelectAvailableInterface(vboxInterfaces, &config.VMConfig{
					Domain: "some-domain",
					IP:     "192.168.22.11",
				})).To(Equal(expectedNetworkConfig))
			})
		})

		Context("when there is a desired non-PCFDev domain passed in", func() {
			It("should return an error - even though this code path should never be reached", func() {
				vboxInterfaces := []*network.Interface{}

				_, err := picker.SelectAvailableInterface(vboxInterfaces, &config.VMConfig{
					Domain: "some-bad-domain",
				})
				Expect(err).To(MatchError("some-bad-domain is not one of the allowed PCF Dev domains"))
			})
		})

		Context("when there is no available network interface", func() {
			It("should return a new interface on 192.168.11.11", func() {
				vboxInterfaces := []*network.Interface{}
				allInterfaces := vboxInterfaces
				expectedNetworkConfig := &config.NetworkConfig{
					VMIP:     "192.168.11.11",
					VMDomain: "local.pcfdev.io",
					Interface: &network.Interface{
						IP:     "192.168.11.1",
						Exists: false,
					},
				}

				mockNetwork.EXPECT().Interfaces().Return(allInterfaces, nil)

				Expect(picker.SelectAvailableInterface(vboxInterfaces, &config.VMConfig{})).To(Equal(expectedNetworkConfig))
			})
		})

		Context("when there is a non-vbox interface on 192.168.11.1 in ifconfig", func() {
			It("should return the next interface", func() {
				vboxInterfaces := []*network.Interface{}
				allInterfaces := append(vboxInterfaces,
					&network.Interface{
						IP:              "192.168.11.1",
						Name:            "some-vmware-interface",
						HardwareAddress: "some-hardware-address",
						Exists:          true,
					},
				)
				expectedNetworkConfig := &config.NetworkConfig{
					VMIP:     "192.168.22.11",
					VMDomain: "local2.pcfdev.io",
					Interface: &network.Interface{
						IP:     "192.168.22.1",
						Exists: false,
					},
				}

				mockNetwork.EXPECT().Interfaces().Return(allInterfaces, nil)

				Expect(picker.SelectAvailableInterface(vboxInterfaces, &config.VMConfig{})).To(Equal(expectedNetworkConfig))
			})
		})

		Context("when there is a vbox and non-vbox interface on 192.168.11.1 in ifconfig", func() {
			It("should return the next interface", func() {
				vboxInterfaces := []*network.Interface{
					&network.Interface{
						IP:              "192.168.11.1",
						Name:            "some-vbox-interface",
						HardwareAddress: "some-vbox-hardware-address",
						Exists:          true,
					},
				}
				allInterfaces := append(vboxInterfaces,
					&network.Interface{
						IP:              "192.168.11.1",
						Name:            "some-vmware-interface",
						HardwareAddress: "some-vmware-hardware-address",
						Exists:          true,
					},
				)
				expectedNetworkConfig := &config.NetworkConfig{
					VMIP:     "192.168.22.11",
					VMDomain: "local2.pcfdev.io",
					Interface: &network.Interface{
						IP:     "192.168.22.1",
						Exists: false,
					},
				}

				mockNetwork.EXPECT().Interfaces().Return(allInterfaces, nil)

				Expect(picker.SelectAvailableInterface(vboxInterfaces, &config.VMConfig{})).To(Equal(expectedNetworkConfig))
			})
		})

		Context("when there are two vbox interfaces on 192.168.11.1", func() {
			It("should return the next interface", func() {
				vboxInterfaces := []*network.Interface{
					&network.Interface{
						Name:            "some-vbox-interface",
						IP:              "192.168.11.1",
						HardwareAddress: "some-vbox-hardware-address",
						Exists:          true,
					},
					&network.Interface{
						Name:            "some-other-vbox-interface",
						IP:              "192.168.11.1",
						HardwareAddress: "some-other-vbox-hardware-address",
						Exists:          true,
					},
				}
				allInterfaces := vboxInterfaces
				expectedNetworkConfig := &config.NetworkConfig{
					VMIP:     "192.168.22.11",
					VMDomain: "local2.pcfdev.io",
					Interface: &network.Interface{
						IP:     "192.168.22.1",
						Exists: false,
					},
				}

				mockNetwork.EXPECT().Interfaces().Return(allInterfaces, nil)

				Expect(picker.SelectAvailableInterface(vboxInterfaces, &config.VMConfig{})).To(Equal(expectedNetworkConfig))
			})
		})

		Context("when there are multiple vbox interfaces and some are not in use", func() {
			It("should reuse the first interface that is not in use", func() {
				vboxInterfaces := []*network.Interface{
					&network.Interface{
						Name:            "some-vbox-interface",
						IP:              "192.168.11.1",
						HardwareAddress: "some-vbox-hardware-address",
						Exists:          true,
					},
					&network.Interface{
						Name:            "some-other-vbox-interface",
						IP:              "192.168.22.1",
						HardwareAddress: "some-other-vbox-hardware-address",
						Exists:          true,
					},
				}
				allInterfaces := vboxInterfaces
				expectedNetworkConfig := &config.NetworkConfig{
					VMIP:     "192.168.22.11",
					VMDomain: "local2.pcfdev.io",
					Interface: &network.Interface{
						Name:            "some-other-vbox-interface",
						IP:              "192.168.22.1",
						HardwareAddress: "some-other-vbox-hardware-address",
						Exists:          true,
					},
				}

				gomock.InOrder(
					mockNetwork.EXPECT().Interfaces().Return(allInterfaces, nil),
					mockDriver.EXPECT().IsInterfaceInUse("some-vbox-interface").Return(true, nil),
					mockDriver.EXPECT().IsInterfaceInUse("some-other-vbox-interface").Return(false, nil),
				)

				Expect(picker.SelectAvailableInterface(vboxInterfaces, &config.VMConfig{})).To(Equal(expectedNetworkConfig))
			})
		})

		Context("when there are multiple vbox interfaces and they are all in use", func() {
			It("should return the next interface", func() {
				vboxInterfaces := []*network.Interface{
					&network.Interface{
						Name:            "some-vbox-interface",
						IP:              "192.168.11.1",
						HardwareAddress: "some-vbox-hardware-address",
						Exists:          true,
					},
					&network.Interface{
						Name:            "some-other-vbox-interface",
						IP:              "192.168.22.1",
						HardwareAddress: "some-other-vbox-hardware-address",
						Exists:          true,
					},
				}
				allInterfaces := vboxInterfaces
				expectedNetworkConfig := &config.NetworkConfig{
					VMIP:     "192.168.33.11",
					VMDomain: "local3.pcfdev.io",
					Interface: &network.Interface{
						IP:     "192.168.33.1",
						Exists: false,
					},
				}

				gomock.InOrder(
					mockNetwork.EXPECT().Interfaces().Return(allInterfaces, nil),
					mockDriver.EXPECT().IsInterfaceInUse("some-vbox-interface").Return(true, nil),
					mockDriver.EXPECT().IsInterfaceInUse("some-other-vbox-interface").Return(true, nil),
				)

				Expect(picker.SelectAvailableInterface(vboxInterfaces, &config.VMConfig{})).To(Equal(expectedNetworkConfig))
			})
		})

		Context("when all allowed interfaces are taken", func() {
			It("should return an error", func() {
				allInterfaces := []*network.Interface{}
				for i := 1; i < 10; i++ {
					allInterfaces = append(allInterfaces,
						&network.Interface{
							Name:            fmt.Sprintf("some-vbox-interface-%d", i),
							IP:              fmt.Sprintf("192.168.%d%d.1", i, i),
							HardwareAddress: fmt.Sprintf("some-vbox-hardware-address.%d", i),
							Exists:          true,
						},
					)
				}

				mockNetwork.EXPECT().Interfaces().Return(allInterfaces, nil)

				_, err := picker.SelectAvailableInterface([]*network.Interface{}, &config.VMConfig{})
				Expect(err).To(MatchError("all allowed network interfaces are currently taken"))
			})
		})

		Context("when there is an error getting all interfaces", func() {
			It("should return the error", func() {
				mockNetwork.EXPECT().Interfaces().Return(nil, errors.New("some-error"))

				_, err := picker.SelectAvailableInterface([]*network.Interface{}, &config.VMConfig{})
				Expect(err).To(MatchError("some-error"))
			})
		})

		Context("when there is an error checking if an interface is in use", func() {
			It("should return the error", func() {
				vboxInterfaces := []*network.Interface{
					&network.Interface{
						Name:            "some-vbox-interface",
						IP:              "192.168.11.1",
						HardwareAddress: "some-vbox-hardware-address",
						Exists:          true,
					},
				}
				allInterfaces := vboxInterfaces

				gomock.InOrder(
					mockNetwork.EXPECT().Interfaces().Return(allInterfaces, nil),
					mockDriver.EXPECT().IsInterfaceInUse("some-vbox-interface").Return(false, errors.New("some-error")),
				)

				_, err := picker.SelectAvailableInterface(vboxInterfaces, &config.VMConfig{})
				Expect(err).To(MatchError("some-error"))
			})
		})
	})
})

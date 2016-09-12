package address_test

import (
	"errors"
	"fmt"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/pcfdev-cli/address"
	"github.com/pivotal-cf/pcfdev-cli/address/mocks"
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
		Context("when there is no available network interface", func() {
			It("should return return a new interface on 192.168.11.11", func() {
				vboxInterfaces := []*network.Interface{}
				allInterfaces := vboxInterfaces
				expectedInterface := &network.Interface{
					IP:     "192.168.11.1",
					Exists: false,
				}

				mockNetwork.EXPECT().Interfaces().Return(allInterfaces, nil)

				Expect(picker.SelectAvailableInterface(vboxInterfaces)).To(Equal(expectedInterface))
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
				expectedInterface := &network.Interface{
					IP:     "192.168.22.1",
					Exists: false,
				}

				mockNetwork.EXPECT().Interfaces().Return(allInterfaces, nil)

				Expect(picker.SelectAvailableInterface(vboxInterfaces)).To(Equal(expectedInterface))
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
				expectedInterface := &network.Interface{
					IP:     "192.168.22.1",
					Exists: false,
				}

				mockNetwork.EXPECT().Interfaces().Return(allInterfaces, nil)

				Expect(picker.SelectAvailableInterface(vboxInterfaces)).To(Equal(expectedInterface))
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
				expectedInterface := &network.Interface{
					IP:     "192.168.22.1",
					Exists: false,
				}

				mockNetwork.EXPECT().Interfaces().Return(allInterfaces, nil)

				Expect(picker.SelectAvailableInterface(vboxInterfaces)).To(Equal(expectedInterface))
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
				expectedInterface := vboxInterfaces[1]

				gomock.InOrder(
					mockNetwork.EXPECT().Interfaces().Return(allInterfaces, nil),
					mockDriver.EXPECT().IsInterfaceInUse("some-vbox-interface").Return(true, nil),
					mockDriver.EXPECT().IsInterfaceInUse("some-other-vbox-interface").Return(false, nil),
				)

				Expect(picker.SelectAvailableInterface(vboxInterfaces)).To(Equal(expectedInterface))
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
				expectedInterface := &network.Interface{
					IP:     "192.168.33.1",
					Exists: false,
				}

				gomock.InOrder(
					mockNetwork.EXPECT().Interfaces().Return(allInterfaces, nil),
					mockDriver.EXPECT().IsInterfaceInUse("some-vbox-interface").Return(true, nil),
					mockDriver.EXPECT().IsInterfaceInUse("some-other-vbox-interface").Return(true, nil),
				)

				Expect(picker.SelectAvailableInterface(vboxInterfaces)).To(Equal(expectedInterface))
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

				_, err := picker.SelectAvailableInterface([]*network.Interface{})
				Expect(err).To(MatchError("all allowed network interfaces are currently taken"))
			})
		})

		Context("when there is an error getting all interfaces", func() {
			It("should return the error", func() {
				mockNetwork.EXPECT().Interfaces().Return(nil, errors.New("some-error"))

				_, err := picker.SelectAvailableInterface([]*network.Interface{})
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

				_, err := picker.SelectAvailableInterface(vboxInterfaces)
				Expect(err).To(MatchError("some-error"))
			})
		})
	})
})

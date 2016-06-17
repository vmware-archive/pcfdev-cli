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

	Describe("#SelectAvailableIP", func() {
		Context("when there is no available network interface", func() {
			It("should return return 192.168.11.11", func() {
				mockNetwork.EXPECT().Interfaces().Return([]*network.Interface{}, nil)

				Expect(picker.SelectAvailableIP([]*network.Interface{})).To(Equal("192.168.11.1"))
			})
		})

		Context("when there is not a vbox interface on 192.168.11.1 but there is an interface on 192.168.11.1 in ifconfig", func() {
			It("should return the next interface", func() {
				netInterfaces := []*network.Interface{
					&network.Interface{
						IP: "192.168.11.1",
					},
				}

				gomock.InOrder(
					mockNetwork.EXPECT().Interfaces().Return(netInterfaces, nil),
				)

				Expect(picker.SelectAvailableIP([]*network.Interface{})).To(Equal("192.168.22.1"))
			})
		})

		Context("when there is a vbox and non-vbox interface on 192.168.11.1 in ifconfig", func() {
			It("should return the next interface", func() {
				vboxInterfaces := []*network.Interface{
					&network.Interface{
						Name:            "some-interface",
						IP:              "192.168.11.1",
						HardwareAddress: "some-hardware-address",
					},
				}
				netInterfaces := []*network.Interface{
					&network.Interface{
						IP:              "192.168.11.1",
						HardwareAddress: "some-vmware-hardware-address",
					},
				}

				gomock.InOrder(
					mockNetwork.EXPECT().Interfaces().Return(netInterfaces, nil),
				)

				Expect(picker.SelectAvailableIP(vboxInterfaces)).To(Equal("192.168.22.1"))
			})
		})

		Context("when there is a vbox interface on 192.168.11.1 and the interface is not in use", func() {
			It("should reuse the existing interface", func() {
				vboxInterfaces := []*network.Interface{
					&network.Interface{
						Name:            "some-interface",
						IP:              "192.168.11.1",
						HardwareAddress: "some-hardware-address",
					},
				}
				netInterfaces := []*network.Interface{
					&network.Interface{
						IP:              "192.168.11.1",
						HardwareAddress: "some-hardware-address",
					},
				}

				gomock.InOrder(
					mockNetwork.EXPECT().Interfaces().Return(netInterfaces, nil),
					mockDriver.EXPECT().IsInterfaceInUse("some-interface").Return(false, nil),
				)

				Expect(picker.SelectAvailableIP(vboxInterfaces)).To(Equal("192.168.11.1"))
			})
		})

		Context("when there is vbox interface on 192.168.11.1 and it is in use", func() {
			It("should return the next interface", func() {
				vboxInterfaces := []*network.Interface{
					&network.Interface{
						Name:            "some-interface",
						IP:              "192.168.11.1",
						HardwareAddress: "some-hardware-address",
					},
				}
				netInterfaces := []*network.Interface{
					&network.Interface{
						IP:              "192.168.11.1",
						HardwareAddress: "some-hardware-address",
					},
				}

				gomock.InOrder(
					mockNetwork.EXPECT().Interfaces().Return(netInterfaces, nil),
					mockDriver.EXPECT().IsInterfaceInUse("some-interface").Return(true, nil),
				)

				Expect(picker.SelectAvailableIP(vboxInterfaces)).To(Equal("192.168.22.1"))
			})
		})

		Context("when all allowed interfaces are taken", func() {
			It("should return an error", func() {
				interfaces := make([]*network.Interface, 9)
				for i := 1; i < 10; i++ {
					interfaces[i-1] = &network.Interface{
						IP:              fmt.Sprintf("192.168.%d%d.1", i, i),
						HardwareAddress: fmt.Sprintf("some-hardware-address.%d", i),
					}
				}

				gomock.InOrder(
					mockNetwork.EXPECT().Interfaces().Return(interfaces, nil),
				)

				_, err := picker.SelectAvailableIP([]*network.Interface{})
				Expect(err).To(MatchError("all allowed network interfaces are currently taken"))
			})
		})

		Context("when it fails to find out if the interface is in use", func() {
			It("should return the error", func() {
				vboxInterfaces := []*network.Interface{
					&network.Interface{
						Name:            "some-interface",
						IP:              "192.168.11.1",
						HardwareAddress: "some-hardware-address",
					},
				}
				netInterfaces := []*network.Interface{
					&network.Interface{
						IP:              "192.168.11.1",
						HardwareAddress: "some-hardware-address",
					},
				}

				gomock.InOrder(
					mockNetwork.EXPECT().Interfaces().Return(netInterfaces, nil),
					mockDriver.EXPECT().IsInterfaceInUse("some-interface").Return(false, errors.New("some-error")),
				)

				_, err := picker.SelectAvailableIP(vboxInterfaces)
				Expect(err).To(MatchError("some-error"))
			})
		})
	})
})

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
		mockPinger  *mocks.MockPinger
		mockNetwork *mocks.MockNetwork
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockNetwork = mocks.NewMockNetwork(mockCtrl)
		mockPinger = mocks.NewMockPinger(mockCtrl)

		picker = &address.Picker{
			Pinger:  mockPinger,
			Network: mockNetwork,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("#SelectAvailableNetworkInterface", func() {
		Context("when there is no available network interface", func() {
			It("should return return 192.168.11.11 and false", func() {
				mockNetwork.EXPECT().Interfaces().Return([]*network.Interface{}, nil)

				iface, exists, err := picker.SelectAvailableNetworkInterface([]*network.Interface{})
				Expect(err).NotTo(HaveOccurred())
				Expect(exists).To(BeFalse())
				Expect(iface.IP).To(Equal("192.168.11.1"))
			})
		})

		Context("when there is a vbox interface on 192.168.11.1 and nothing responds to ping on 192.168.11.11", func() {
			It("should reuse the existing interface", func() {
				vboxInterface := &network.Interface{
					Name: "some-interface",
					IP:   "192.168.11.1",
				}
				netInterface := &network.Interface{
					IP: "192.168.11.1",
				}

				gomock.InOrder(
					mockNetwork.EXPECT().Interfaces().Return([]*network.Interface{netInterface}, nil),
					mockPinger.EXPECT().TryIP("192.168.11.11").Return(false, nil),
				)

				selected, exists, err := picker.SelectAvailableNetworkInterface([]*network.Interface{vboxInterface})
				Expect(err).NotTo(HaveOccurred())
				Expect(exists).To(BeTrue())
				Expect(selected.Name).To(Equal("some-interface"))
				Expect(selected.IP).To(Equal("192.168.11.1"))
			})
		})

		Context("when there is a vbox interface on 192.168.11.1 and something responds to ping on 192.168.11.11", func() {
			It("should return the next interface and false", func() {
				vboxInterface := &network.Interface{
					Name: "some-interface",
					IP:   "192.168.11.1",
				}
				netInterface := &network.Interface{
					IP: "192.168.11.1",
				}

				gomock.InOrder(
					mockNetwork.EXPECT().Interfaces().Return([]*network.Interface{netInterface}, nil),
					mockPinger.EXPECT().TryIP("192.168.11.11").Return(true, nil),
				)

				selected, exists, err := picker.SelectAvailableNetworkInterface([]*network.Interface{vboxInterface})
				Expect(err).NotTo(HaveOccurred())
				Expect(exists).To(BeFalse())
				Expect(selected.IP).To(Equal("192.168.22.1"))
			})
		})

		Context("when there is not a vbox interface on 192.168.11.1 but there an interface on 192.168.11.1 in ifconfig", func() {
			It("should return the next interface and false", func() {
				netInterface := &network.Interface{
					IP: "192.168.11.1",
				}

				gomock.InOrder(
					mockNetwork.EXPECT().Interfaces().Return([]*network.Interface{netInterface}, nil),
				)

				selected, exists, err := picker.SelectAvailableNetworkInterface([]*network.Interface{})
				Expect(err).NotTo(HaveOccurred())
				Expect(exists).To(BeFalse())
				Expect(selected.IP).To(Equal("192.168.22.1"))
			})
		})

		Context("all allowed interfaces are taken", func() {
			It("returns and error", func() {
				interfaces := make([]*network.Interface, 9)
				for i := 1; i < 10; i++ {
					interfaces[i-1] = &network.Interface{
						Name: "some-interface",
						IP:   fmt.Sprintf("192.168.%d%d.1", i, i),
					}
				}

				gomock.InOrder(
					mockNetwork.EXPECT().Interfaces().Return(interfaces, nil),
				)

				_, _, err := picker.SelectAvailableNetworkInterface([]*network.Interface{})
				Expect(err).To(MatchError("all allowed network interfaces are currently taken"))
			})
		})

		Context("when it fails to get all network interfaces", func() {
			It("should return an error", func() {
				mockNetwork.EXPECT().Interfaces().Return([]*network.Interface{}, errors.New("some-error"))

				_, _, err := picker.SelectAvailableNetworkInterface([]*network.Interface{})
				Expect(err).To(MatchError("some-error"))
			})
		})

		Context("when it fails attempt to ping ip", func() {
			It("should return an error", func() {
				netInterface := &network.Interface{
					IP: "192.168.11.1",
				}

				gomock.InOrder(
					mockNetwork.EXPECT().Interfaces().Return([]*network.Interface{netInterface}, nil),
					mockPinger.EXPECT().TryIP("192.168.11.11").Return(false, errors.New("some-error")),
				)

				_, _, err := picker.SelectAvailableNetworkInterface([]*network.Interface{netInterface})
				Expect(err).To(MatchError("some-error"))
			})
		})

	})
})

package ping_test

import (
	"errors"
	"net"

	"github.com/golang/mock/gomock"
	"github.com/pivotal-cf/pcfdev-cli/ping"
	"github.com/pivotal-cf/pcfdev-cli/ping/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ping", func() {

	var (
		mockCtrl *gomock.Controller
		mockUser *mocks.MockUser
		pinger   *ping.Pinger
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockUser = mocks.NewMockUser(mockCtrl)
		pinger = &ping.Pinger{
			User: mockUser,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Context("#ICMPProtocol", func() {
		Context("when it fails to determine user privilege", func() {
			It("should return an error", func() {
				mockUser.EXPECT().IsPrivileged().Return(false, errors.New("some-error"))

				_, err := pinger.ICMPProtocol()
				Expect(err).To(MatchError("failed to determine user privileges: some-error"))
			})
		})

		Context("when user is not privileged", func() {
			It("should return 'udp4'", func() {
				mockUser.EXPECT().IsPrivileged().Return(false, nil)

				Expect(pinger.ICMPProtocol()).To(Equal("udp4"))
			})
		})

		Context("when user is privileged", func() {
			It("should return 'ip4:1'", func() {
				mockUser.EXPECT().IsPrivileged().Return(true, nil)

				Expect(pinger.ICMPProtocol()).To(Equal("ip4:1"))
			})
		})
	})

	Context("#ICMPAddr", func() {
		Context("when it errors trying to determine user privilege", func() {
			It("should return an error", func() {
				mockUser.EXPECT().IsPrivileged().Return(false, errors.New("some-error"))

				_, err := pinger.ICMPAddr("some-ip")
				Expect(err).To(MatchError("failed to determine user privileges: some-error"))
			})
		})

		Context("when user is not privileged", func() {
			It("should return a UDPAddr object", func() {
				mockUser.EXPECT().IsPrivileged().Return(false, nil)

				expectedIMCPAddr := &net.UDPAddr{
					IP: net.ParseIP("some-ip"),
				}

				Expect(pinger.ICMPAddr("some-ip")).To(Equal(expectedIMCPAddr))
			})
		})

		Context("when user is privileged", func() {
			It("should return a UDPAddr object", func() {
				mockUser.EXPECT().IsPrivileged().Return(true, nil)

				expectedIMCPAddr := &net.IPAddr{
					IP: net.ParseIP("some-ip"),
				}

				Expect(pinger.ICMPAddr("some-ip")).To(Equal(expectedIMCPAddr))
			})
		})
	})

	Context("#TryIP", func() {
		Context("when a machine with the given ip responds", func() {
			It("should return true", func() {
				mockUser.EXPECT().IsPrivileged().Times(2)

				responds, err := pinger.TryIP("216.58.217.78")
				Expect(err).NotTo(HaveOccurred())
				Expect(responds).To(BeTrue())
			})
		})

		Context("when a machine with the given ip does not respond", func() {
			It("should return false", func() {
				mockUser.EXPECT().IsPrivileged().Times(2)

				responds, err := pinger.TryIP("192.168.23.23")
				Expect(err).NotTo(HaveOccurred())
				Expect(responds).To(BeFalse())
			})
		})
	})
})

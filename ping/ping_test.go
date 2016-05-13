package ping_test

import (
	"github.com/golang/mock/gomock"
	"github.com/pivotal-cf/pcfdev-cli/ping"
	currentUser "github.com/pivotal-cf/pcfdev-cli/user"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ping", func() {

	var (
		mockCtrl *gomock.Controller
		user     *currentUser.User
		pinger   *ping.Pinger
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		user = &currentUser.User{}
		pinger = &ping.Pinger{
			User: user,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Context("#TryIP", func() {
		Context("when a machine with the given ip responds", func() {
			It("should return true", func() {
				responds, err := pinger.TryIP("216.58.217.78")
				Expect(err).NotTo(HaveOccurred())
				Expect(responds).To(BeTrue())
			})
		})

		Context("when a machine with the given ip does not respond", func() {
			It("should return false", func() {
				responds, err := pinger.TryIP("192.168.23.23")
				Expect(err).NotTo(HaveOccurred())
				Expect(responds).To(BeFalse())
			})
		})
	})
})

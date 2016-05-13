package ping_test

import (
	"github.com/pivotal-cf/pcfdev-cli/ping"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ping", func() {
	Context("when a machine with the given ip responds", func() {
		It("should return true", func() {
			pinger := ping.Pinger{}
			responds, err := pinger.TryIP("216.58.217.78")
			Expect(err).NotTo(HaveOccurred())
			Expect(responds).To(BeTrue())
		})
	})

	Context("when a machine with the given ip does not respond", func() {
		It("should return false", func() {
			pinger := ping.Pinger{}
			responds, err := pinger.TryIP("192.168.23.23")
			Expect(err).NotTo(HaveOccurred())
			Expect(responds).To(BeFalse())
		})
	})
})

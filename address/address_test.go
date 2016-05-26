package address_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/pcfdev-cli/address"
)

var _ = Describe("Address", func() {

	Describe("#DomainForIP", func() {
		It("should convert a passed in ip to the correct domain", func() {
			Expect(address.DomainForIP("192.168.11.11")).To(Equal("local.pcfdev.io"))
			Expect(address.DomainForIP("192.168.22.11")).To(Equal("local2.pcfdev.io"))
			Expect(address.DomainForIP("192.168.33.11")).To(Equal("local3.pcfdev.io"))
		})

		Context("when the ip does not match any of the route53 records", func() {
			It("should return an error", func() {
				_, err := address.DomainForIP("192.168.41.81")
				Expect(err).To(MatchError("192.168.41.81 is not one of the allowed PCF Dev ips"))
			})
		})
	})

	Describe("#SubnetForIP", func() {
		It("should convert a passed in ip to the correct domain", func() {
			Expect(address.SubnetForIP("192.168.11.11")).To(Equal("192.168.11.1"))
			Expect(address.SubnetForIP("192.168.22.11")).To(Equal("192.168.22.1"))
			Expect(address.SubnetForIP("192.168.33.11")).To(Equal("192.168.33.1"))
		})

		Context("when the ip does not match any of the route53 records", func() {
			It("should return an error", func() {
				_, err := address.SubnetForIP("192.168.41.81")
				Expect(err).To(MatchError("192.168.41.81 is not one of the allowed PCF Dev ips"))
			})
		})
	})

	Describe("#IPForSubnet", func() {
		It("should convert a passed in ip to the correct domain", func() {
			Expect(address.IPForSubnet("192.168.11.1")).To(Equal("192.168.11.11"))
			Expect(address.IPForSubnet("192.168.22.1")).To(Equal("192.168.22.11"))
			Expect(address.IPForSubnet("192.168.33.1")).To(Equal("192.168.33.11"))
		})

		Context("when the ip does not match any of the route53 records", func() {
			It("should return an error", func() {
				_, err := address.IPForSubnet("192.168.41.1")
				Expect(err).To(MatchError("192.168.41.1 is not one of the allowed PCF Dev subnets"))
			})
		})
	})
})

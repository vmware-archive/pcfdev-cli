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
			Expect(address.DomainForIP("192.168.89.11")).To(Equal("192.168.89.11.xip.io"))
		})
	})

	Describe("#SubnetForIP", func() {
		It("should convert a passed in ip to the correct domain", func() {
			Expect(address.SubnetForIP("192.168.11.11")).To(Equal("192.168.11.1"))
			Expect(address.SubnetForIP("192.168.22.11")).To(Equal("192.168.22.1"))
			Expect(address.SubnetForIP("192.168.53.53")).To(Equal("192.168.53.1"))
		})

		Context("when the ip is not a valid IPv4 address", func() {
			It("should return an error", func() {
				_, err := address.SubnetForIP("some-bad-ip")
				Expect(err).To(MatchError("some-bad-ip is not a supported IP address"))
			})
		})
	})

	Describe("#SubnetForDomain", func() {
		It("should convert a passed in domain to the correct ip", func() {
			Expect(address.SubnetForDomain("local.pcfdev.io")).To(Equal("192.168.11.1"))
			Expect(address.SubnetForDomain("local2.pcfdev.io")).To(Equal("192.168.22.1"))
			Expect(address.SubnetForDomain("local3.pcfdev.io")).To(Equal("192.168.33.1"))
		})

		Context("when the domain does not match any of the route53 records", func() {
			It("should return an error", func() {
				_, err := address.SubnetForDomain("some-bad-domain")
				Expect(err).To(MatchError("some-bad-domain is not one of the allowed PCF Dev domains"))
			})
		})
	})

	Describe("#IPForSubnet", func() {
		It("returns the subnet + 1", func() {
			Expect(address.IPForSubnet("192.168.11.1")).To(Equal("192.168.11.11"))
		})
	})

	Describe("#IsDomainAllowed", func() {
		Context("when the domain is a standard pcfdev domain", func() {
			It("return true", func() {
				Expect(address.IsDomainAllowed("local.pcfdev.io")).To(BeTrue())
				Expect(address.IsDomainAllowed("local2.pcfdev.io")).To(BeTrue())
				Expect(address.IsDomainAllowed("local3.pcfdev.io")).To(BeTrue())
				Expect(address.IsDomainAllowed("local4.pcfdev.io")).To(BeTrue())
			})
		})

		Context("when the domain is not a standard pcfdev domain", func() {
			It("return false", func() {
				Expect(address.IsDomainAllowed("some-bad-domain")).To(BeFalse())
			})
		})
	})
})

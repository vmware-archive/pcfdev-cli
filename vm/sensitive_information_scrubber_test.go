package vm_test

import (
	"github.com/pivotal-cf/pcfdev-cli/vm"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SensitiveInformationScrubber", func() {
	var (
		scrubber *vm.SensitiveInformationScrubber
	)

	BeforeEach(func() {
		scrubber = &vm.SensitiveInformationScrubber{}
	})

	Describe("#Scrub", func() {
		It("should not scrub non-sensitive information away", func() {
			Expect(scrubber.Scrub("some-non-sensitive-information")).To(Equal("some-non-sensitive-information"))
		})

		It("should scrub certificates away", func() {
			Expect(scrubber.Scrub("some-info-----BEGIN CERTIFICATE-----some-certificate-----END CERTIFICATE-----some-other-info")).To(Equal("some-info<redacted certificate>some-other-info"))
		})

		It("should scrub private keys away", func() {
			Expect(scrubber.Scrub("some-info-----BEGIN RSA PRIVATE KEY-----some-private-key-----END RSA PRIVATE KEY-----some-other-info")).To(Equal("some-info<redacted private-key>some-other-info"))
		})

		It("should scrub emails away", func() {
			Expect(scrubber.Scrub("some-info someone@somedomain.com some-other-info")).To(Equal("some-info <redacted email> some-other-info"))
		})

		It("should scrub ip addresses away", func() {
			Expect(scrubber.Scrub("some-info88.88.88.88some-other-info")).To(Equal("some-info<redacted ip-address>some-other-info"))
		})

		It("should scrub ip addresses with commas away", func() {
			Expect(scrubber.Scrub("some-info88,88,88,88some-other-info")).To(Equal("some-info<redacted ip-address>some-other-info"))
		})

		It("should scrub urls away", func() {
			Expect(scrubber.Scrub("some-info http://some-domain.com some-other-info")).To(Equal("some-info <redacted uri> some-other-info"))
		})

		It("should scrub database uris away", func() {
			Expect(scrubber.Scrub("some-info jdbc:postgresql://some-domain/some-db?user=some-user&password=some-password&ssl=true some-other-info")).To(Equal("some-info <redacted uri> some-other-info"))
		})

		It("should scrub secrets and passwords away", func() {
			Expect(scrubber.Scrub(`some-info"password"=>"some-passworD"some-other-info`)).To(Equal("some-info<redacted secret>some-other-info"))
		})

		It("should scrub secure environment variables away", func() {
			Expect(scrubber.Scrub(`some-info password=some-passworD some-other-info`)).To(Equal("some-info <redacted secret>some-other-info"))
		})
	})
})

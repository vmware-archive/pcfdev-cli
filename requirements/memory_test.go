package requirements_test

import (
	"github.com/pivotal-cf/pcfdev-cli/requirements"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Memory", func() {
	Describe("Check", func() {
		Context("when the system has enough free memory", func() {
			It("should not return an error", func() {
				By("assuming test is running on a system with at least 128MB of free memory")
				memory := &requirements.Memory{MinimumFreeMemory: 128}
				Expect(memory.Check()).To(Succeed())
			})
		})

		Context("when the system does not have enough free memory", func() {
			It("should return an error", func() {
				By("assuming test is running on a system with less than 128GB of free memory")
				memory := &requirements.Memory{MinimumFreeMemory: 131072}
				Expect(memory.Check()).To(MatchError(MatchRegexp(`PCF Dev requires 131,072MB of free memory. This host machine has [\d,]+MB free.`)))
			})
		})
	})
})

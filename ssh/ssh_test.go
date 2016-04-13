package ssh_test

import (
	. "github.com/pivotal-cf/pcfdev-cli/ssh"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ssh", func() {
	It("Should return a random port", func() {
		ssh := &SSH{}
		port, err := ssh.RandomPort()
		Expect(err).NotTo(HaveOccurred())
		Expect(port).To(MatchRegexp("^[\\d]+$"))
	})
})

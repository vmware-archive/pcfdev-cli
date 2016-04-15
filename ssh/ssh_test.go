package ssh_test

import (
	. "github.com/pivotal-cf/pcfdev-cli/ssh"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ssh", func() {
	It("Should return a free port", func() {
		ssh := &SSH{}
		port, err := ssh.FreePort()
		Expect(err).NotTo(HaveOccurred())
		Expect(port).To(MatchRegexp("^[\\d]+$"))
	})
})

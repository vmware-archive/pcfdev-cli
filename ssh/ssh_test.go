package ssh_test

import (
	gossh "golang.org/x/crypto/ssh"

	. "github.com/pivotal-cf/pcfdev-cli/ssh"
	"github.com/pivotal-cf/pcfdev-cli/ssh/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ssh", func() {
	Describe("GenerateAddress", func() {
		It("Should return a host and free port", func() {
			ssh := &SSH{}
			host, port, err := ssh.GenerateAddress()
			Expect(err).NotTo(HaveOccurred())
			Expect(host).To(Equal("127.0.0.1"))
			Expect(port).To(MatchRegexp("^[\\d]+$"))
		})
	})

	Describe("WaitForSSH", func() {
		var (
			ssh           *SSH
			mockSSHServer *mocks.SSHServer
			port          string
			config        *gossh.ClientConfig
		)

		BeforeEach(func() {
			ssh = &SSH{}
			var address string
			var err error
			address, port, err = ssh.GenerateAddress()
			Expect(err).NotTo(HaveOccurred())
			mockSSHServer = &mocks.SSHServer{
				User:     "some-valid-user",
				Password: "some-valid-password",
				Host:     address,
				Port:     port,
			}
			config = &gossh.ClientConfig{
				User: "some-valid-user",
				Auth: []gossh.AuthMethod{
					gossh.Password("some-valid-password"),
				},
			}
			mockSSHServer.Start()
		})

		AfterEach(func() {
			mockSSHServer.Stop()
		})

		Context("when the server is available", func() {
			It("should return a pointer to an ssh client", func() {
				client, err := ssh.WaitForSSH(config, port)
				Expect(err).NotTo(HaveOccurred())
				session, err := client.NewSession()
				defer session.Close()
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when the server is unavailable", func() {
			It("should timeout and return an error", func() {
				_, err := ssh.WaitForSSH(config, "some-bad-port")
				Expect(err).To(HaveOccurred())
			})
		})
	})
})

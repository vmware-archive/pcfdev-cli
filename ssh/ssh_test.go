package ssh_test

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/pivotal-cf/pcfdev-cli/helpers"
	. "github.com/pivotal-cf/pcfdev-cli/ssh"

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

	Describe("RunSSHCommand", func() {
		var ssh *SSH

		Context("when SSH is available", func() {
			var (
				vmName string
				port   string
			)

			BeforeEach(func() {
				ssh = &SSH{}

				var err error
				vmName, err = helpers.ImportSnappy()
				Expect(err).NotTo(HaveOccurred())

				_, port, err = ssh.GenerateAddress()
				Expect(err).NotTo(HaveOccurred())

				Expect(exec.Command("VBoxManage", "modifyvm", vmName, "--natpf1", fmt.Sprintf("ssh,tcp,127.0.0.1,%s,,22", port)).Run()).To(Succeed())
				Expect(exec.Command("VBoxManage", "startvm", vmName, "--type", "headless").Run()).To(Succeed())
			})

			AfterEach(func() {
				Expect(exec.Command("VBoxManage", "controlvm", vmName, "poweroff").Run()).To(Succeed())
				Expect(exec.Command("VBoxManage", "unregistervm", vmName, "--delete").Run()).To(Succeed())
			})

			Context("when the command succeeds", func() {
				It("should return the output", func() {
					output, err := ssh.RunSSHCommand("echo -n some-output", port, 5*time.Minute)
					Expect(err).NotTo(HaveOccurred())
					Expect(string(output)).To(Equal("some-output"))
				})
			})

			Context("when the command fails", func() {
				It("should return an error", func() {
					_, err := ssh.RunSSHCommand("false", port, 5*time.Minute)
					Expect(err).To(MatchError(ContainSubstring("Process exited with: 1")))
				})
			})
		})

		Context("when SSH connection times out", func() {
			It("should return an error", func() {
				_, err := ssh.RunSSHCommand("echo -n some-output", "some-bad-port", time.Second)
				Expect(err).To(MatchError(ContainSubstring("ssh connection timed out:")))
			})
		})
	})
})

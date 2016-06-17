package ssh_test

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"time"

	"github.com/pivotal-cf/pcfdev-cli/helpers"
	. "github.com/pivotal-cf/pcfdev-cli/ssh"
	"github.com/pivotal-cf/pcfdev-cli/test_helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var vBoxManagePath string

var _ = BeforeSuite(func() {
	var err error
	vBoxManagePath, err = helpers.VBoxManagePath()
	Expect(err).NotTo(HaveOccurred())
})

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
				stdout *gbytes.Buffer
				stderr *gbytes.Buffer
			)

			BeforeEach(func() {
				ssh = &SSH{}

				var err error
				stdout = gbytes.NewBuffer()
				stderr = gbytes.NewBuffer()
				vmName, err = test_helpers.ImportSnappy()
				Expect(err).NotTo(HaveOccurred())

				_, port, err = ssh.GenerateAddress()
				Expect(err).NotTo(HaveOccurred())

				Expect(exec.Command(vBoxManagePath, "modifyvm", vmName, "--natpf1", fmt.Sprintf("ssh,tcp,127.0.0.1,%s,,22", port)).Run()).To(Succeed())
				Expect(exec.Command(vBoxManagePath, "startvm", vmName, "--type", "headless").Run()).To(Succeed())
			})

			AfterEach(func() {
				Expect(exec.Command(vBoxManagePath, "controlvm", vmName, "poweroff").Run()).To(Succeed())
				Expect(exec.Command(vBoxManagePath, "unregistervm", vmName, "--delete").Run()).To(Succeed())
			})

			Context("when the command succeeds", func() {
				It("should stream stdout to the terminal", func() {
					err := ssh.RunSSHCommand("echo -n some-output", port, 5*time.Minute, stdout, stderr)
					Expect(err).NotTo(HaveOccurred())
					Eventually(string(stdout.Contents()), 10*time.Second).Should(Equal("some-output"))
				})

				It("should stream stderr to the terminal", func() {
					err := ssh.RunSSHCommand(">&2 echo -n some-output", port, 5*time.Minute, stdout, stderr)
					Expect(err).NotTo(HaveOccurred())
					Eventually(string(stderr.Contents()), 10*time.Second).Should(Equal("some-output"))
				})
			})

			Context("when the command fails", func() {
				It("should return an error", func() {
					err := ssh.RunSSHCommand("false", port, 5*time.Minute, stdout, stderr)
					Expect(err).To(MatchError(ContainSubstring("Process exited with: 1")))
				})
			})
		})

		Context("when SSH connection times out", func() {
			It("should return an error", func() {
				err := ssh.RunSSHCommand("echo -n some-output", "some-bad-port", time.Second, ioutil.Discard, ioutil.Discard)
				Expect(err).To(MatchError(ContainSubstring("ssh connection timed out:")))
			})
		})
	})
})

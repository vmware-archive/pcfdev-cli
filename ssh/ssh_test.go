package ssh_test

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"time"

	gossh "golang.org/x/crypto/ssh"

	"github.com/pivotal-cf/pcfdev-cli/helpers"
	. "github.com/pivotal-cf/pcfdev-cli/ssh"
	"github.com/pivotal-cf/pcfdev-cli/test_helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var (
	vBoxManagePath  string
	vmName          string
	port            string
	privateKeyBytes []byte

	ssh *SSH
)

var _ = BeforeSuite(func() {
	var err error
	vBoxManagePath, err = helpers.VBoxManagePath()
	Expect(err).NotTo(HaveOccurred())

	privateKeyBytes, err = ioutil.ReadFile(filepath.Join("..", "assets", "insecure.key"))
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("ssh", func() {
	Describe("GenerateAddress", func() {
		It("Should return a host and free port", func() {
			ssh = &SSH{}
			host, port, err := ssh.GenerateAddress()
			Expect(err).NotTo(HaveOccurred())
			Expect(host).To(Equal("127.0.0.1"))
			Expect(port).To(MatchRegexp("^[\\d]+$"))
		})
	})

	Describe("#RunSSHCommand", func() {
		Context("when SSH is available", func() {
			var (
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
				Eventually(func() error {
					return exec.Command(vBoxManagePath, "unregistervm", vmName, "--delete").Run()
				}, "10s").Should(Succeed())
			})

			Context("when the command succeeds", func() {
				It("should stream stdout to the terminal", func() {
					Expect(ssh.RunSSHCommand("echo -n some-output", []SSHAddress{{IP: "127.0.0.1", Port: port}}, privateKeyBytes, 5*time.Minute, stdout, stderr)).To(Succeed())
					Eventually(string(stdout.Contents()), 20*time.Second).Should(Equal("some-output"))
				})

				It("should stream stderr to the terminal", func() {
					Expect(ssh.RunSSHCommand(">&2 echo -n some-output", []SSHAddress{{IP: "127.0.0.1", Port: port}}, privateKeyBytes, 5*time.Minute, stdout, stderr)).To(Succeed())
					Eventually(string(stderr.Contents()), 20*time.Second).Should(Equal("some-output"))
				})
			})

			Context("when the command fails", func() {
				It("should return an error", func() {
					Expect(ssh.RunSSHCommand("false", []SSHAddress{{IP: "127.0.0.1", Port: port}}, privateKeyBytes, 5*time.Minute, stdout, stderr)).To(MatchError(ContainSubstring("Process exited with: 1")))
				})
			})

			Context("when private key is bad", func() {
				It("should return an error", func() {
					Expect(ssh.RunSSHCommand("false", []SSHAddress{{IP: "127.0.0.1", Port: port}}, []byte("some-bad-private-key"), 5*time.Minute, stdout, stderr)).To(MatchError(ContainSubstring("could not parse private key:")))
				})
			})
		})

		Context("when SSH connection times out", func() {
			It("should return an error", func() {
				Expect(ssh.RunSSHCommand("echo -n some-output", []SSHAddress{{IP: "127.0.0.1", Port: "some-bad-port"}}, privateKeyBytes, time.Second, ioutil.Discard, ioutil.Discard)).To(MatchError(ContainSubstring("ssh connection timed out:")))
			})
		})
	})

	Describe("#WaitForSSH", func() {
		var ip string
		Context("when SSH is available", func() {
			BeforeEach(func() {
				ssh = &SSH{}

				var err error
				vmName, err = test_helpers.ImportSnappy()
				Expect(err).NotTo(HaveOccurred())

				ip, port, err = ssh.GenerateAddress()
				Expect(err).NotTo(HaveOccurred())

				Expect(exec.Command(vBoxManagePath, "modifyvm", vmName, "--natpf1", fmt.Sprintf("ssh,tcp,127.0.0.1,%s,,22", port)).Run()).To(Succeed())
				Expect(exec.Command(vBoxManagePath, "startvm", vmName, "--type", "headless").Run()).To(Succeed())
			})

			AfterEach(func() {
				Expect(exec.Command(vBoxManagePath, "controlvm", vmName, "poweroff").Run()).To(Succeed())
				Eventually(func() error {
					return exec.Command(vBoxManagePath, "unregistervm", vmName, "--delete").Run()
				}, "10s").Should(Succeed())
			})

			It("should succeed", func() {
				Expect(ssh.WaitForSSH([]SSHAddress{{IP: ip, Port: port}}, privateKeyBytes, 5*time.Minute)).To(Succeed())
			})

			Context("when a bad ssh address is passed in along with a good one", func() {
				It("should succeed", func() {
					Expect(ssh.WaitForSSH([]SSHAddress{{IP: ip, Port: port}, {IP: "some-bad-ip", Port: "some-port"}}, privateKeyBytes, 5*time.Minute)).To(Succeed())
				})
			})
		})

		Context("when there is more than one ssh port to the VM", func() {
			BeforeEach(func() {
				ssh = &SSH{}

				var err error
				vmName, err = test_helpers.ImportSnappy()
				Expect(err).NotTo(HaveOccurred())

				ip, port, err = ssh.GenerateAddress()
				Expect(err).NotTo(HaveOccurred())

				Expect(exec.Command(vBoxManagePath, "modifyvm", vmName, "--natpf1", fmt.Sprintf("ssh,tcp,127.0.0.1,%s,,22", port)).Run()).To(Succeed())

				ip, port, err = ssh.GenerateAddress()
				Expect(err).NotTo(HaveOccurred())

				Expect(exec.Command(vBoxManagePath, "modifyvm", vmName, "--natpf1", fmt.Sprintf("ssh2,tcp,127.0.0.1,%s,,22", port)).Run()).To(Succeed())
				Expect(exec.Command(vBoxManagePath, "startvm", vmName, "--type", "headless").Run()).To(Succeed())
			})

			AfterEach(func() {
				Expect(exec.Command(vBoxManagePath, "controlvm", vmName, "poweroff").Run()).To(Succeed())
				Eventually(func() error {
					return exec.Command(vBoxManagePath, "unregistervm", vmName, "--delete").Run()
				}, "10s").Should(Succeed())
			})
			It("should succeed", func() {
				Expect(ssh.WaitForSSH([]SSHAddress{{IP: ip, Port: port}}, privateKeyBytes, 5*time.Minute)).To(Succeed())
			})
		})

		Context("when SSH connection times out", func() {
			It("should return an error", func() {
				Expect(ssh.WaitForSSH([]SSHAddress{{IP: ip, Port: port}}, privateKeyBytes, 5*time.Second)).To(MatchError(ContainSubstring("ssh connection timed out:")))
			})
		})

		Context("when private key is bad", func() {
			It("should return an error", func() {
				Expect(ssh.WaitForSSH([]SSHAddress{{IP: ip, Port: port}}, []byte("some-bad-private-key"), 5*time.Second)).To(MatchError(ContainSubstring("could not parse private key:")))
			})
		})

	})

	Describe("#GetSSHOutput", func() {
		var ip string

		Context("when SSH is available", func() {
			BeforeEach(func() {
				ssh = &SSH{}

				var err error
				vmName, err = test_helpers.ImportSnappy()
				Expect(err).NotTo(HaveOccurred())

				ip, port, err = ssh.GenerateAddress()
				Expect(err).NotTo(HaveOccurred())

				Expect(exec.Command(vBoxManagePath, "modifyvm", vmName, "--natpf1", fmt.Sprintf("ssh,tcp,127.0.0.1,%s,,22", port)).Run()).To(Succeed())
				Expect(exec.Command(vBoxManagePath, "startvm", vmName, "--type", "headless").Run()).To(Succeed())
			})

			AfterEach(func() {
				Expect(exec.Command(vBoxManagePath, "controlvm", vmName, "poweroff").Run()).To(Succeed())
				Eventually(func() error {
					return exec.Command(vBoxManagePath, "unregistervm", vmName, "--delete").Run()
				}, "10s").Should(Succeed())
			})

			It("should return the output of the ssh command", func() {
				Expect(ssh.GetSSHOutput("echo -n some-output", []SSHAddress{{IP: ip, Port: port}}, privateKeyBytes, 5*time.Minute)).To(Equal("some-output"))
			})

			It("should return the stderr of the ssh command", func() {
				Expect(ssh.GetSSHOutput(">&2 echo -n some-output", []SSHAddress{{IP: ip, Port: port}}, privateKeyBytes, 5*time.Minute)).To(Equal("some-output"))
			})

			Context("when the command fails", func() {
				It("should return an error", func() {
					output, err := ssh.GetSSHOutput("echo -n some-output; false", []SSHAddress{{IP: ip, Port: port}}, privateKeyBytes, 5*time.Minute)
					Expect(output).To(Equal("some-output"))
					Expect(err).To(MatchError(ContainSubstring("Process exited with: 1")))
				})
			})
		})

		Context("when SSH connection times out", func() {
			It("should return an error", func() {
				_, err := ssh.GetSSHOutput("echo -n some-output", []SSHAddress{{IP: ip, Port: "some-bad-port"}}, privateKeyBytes, time.Second)
				Expect(err).To(MatchError(ContainSubstring("ssh connection timed out:")))
			})
		})

		Context("when private key is bad", func() {
			It("should return an error", func() {
				_, err := ssh.GetSSHOutput("echo -n some-output", []SSHAddress{{IP: ip, Port: port}}, []byte("some-bad-private-key"), time.Second)
				Expect(err).To(MatchError(ContainSubstring("could not parse private key:")))
			})
		})
	})

	Describe("#StartSSHSession", func() {
		var (
			stdin  *gbytes.Buffer
			stdout *gbytes.Buffer
			stderr *gbytes.Buffer
		)

		BeforeEach(func() {
			ssh = &SSH{}

			var err error
			stdin = gbytes.NewBuffer()
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
			Eventually(func() error {
				return exec.Command(vBoxManagePath, "unregistervm", vmName, "--delete").Run()
			}, "10s").Should(Succeed())
		})

		It("should start an ssh session into the VM", func() {
			go func() {
				time.Sleep(5 * time.Second)
				fmt.Fprintln(stdin, "exit")
			}()

			err := ssh.StartSSHSession([]SSHAddress{{IP: "127.0.0.1", Port: port}}, privateKeyBytes, 5*time.Minute, stdin, stdout, stderr)
			Expect(err).NotTo(HaveOccurred())

			Eventually(stdout).Should(gbytes.Say("Welcome to Ubuntu"))
		})

		Context("when there is an error creating the ssh session", func() {
			It("should return the error", func() {
				err := ssh.StartSSHSession([]SSHAddress{{IP: "127.0.0.1", Port: "some-bad-port"}}, privateKeyBytes, time.Second, stdin, stdout, stderr)
				Expect(err).To(MatchError(ContainSubstring("ssh connection timed out:")))
			})
		})

		Context("when the private key is bad", func() {
			It("should return the error", func() {
				err := ssh.StartSSHSession([]SSHAddress{{IP: "127.0.0.1", Port: port}}, []byte("some-bad-private-key"), time.Second, stdin, stdout, stderr)
				Expect(err).To(MatchError(ContainSubstring("could not parse private key:")))
			})
		})
	})

	Describe("#GenerateKeypair", func() {
		It("should generate an rsa keypair", func() {
			privateKey, publicKey, err := ssh.GenerateKeypair()
			Expect(err).NotTo(HaveOccurred())

			signer, err := gossh.ParsePrivateKey(privateKey)
			Expect(err).NotTo(HaveOccurred())

			Expect(gossh.MarshalAuthorizedKey(signer.PublicKey())).To(Equal(publicKey))
		})
	})
})

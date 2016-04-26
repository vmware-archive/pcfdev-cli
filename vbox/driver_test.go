package vbox_test

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/pivotal-cf/pcfdev-cli/ssh"
	"github.com/pivotal-cf/pcfdev-cli/vbox"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"

	cssh "golang.org/x/crypto/ssh"
)

var _ = Describe("driver", func() {
	var driver *vbox.VBoxDriver

	BeforeEach(func() {
		driver = &vbox.VBoxDriver{}
		_, err := os.Stat("../assets/snappy.ova")
		if os.IsNotExist(err) {
			By("Downloading ova...")
			resp, err := http.Get("https://s3.amazonaws.com/pcfdev/ovas/snappy.ova")
			Expect(err).NotTo(HaveOccurred())
			ovaFile, err := os.Create("../assets/snappy.ova")
			Expect(err).NotTo(HaveOccurred())
			defer ovaFile.Close()
			_, err = io.Copy(ovaFile, resp.Body)
		}

		_, err = driver.VBoxManage("import", "../assets/snappy.ova")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		driver.VBoxManage("controlvm", "Snappy", "poweroff")
		driver.VBoxManage("unregistervm", "Snappy", "--delete")
		driver.VBoxManage("hostonlyif", "remove", "vboxnet0")
	})

	Describe("#VBoxManage", func() {
		It("should execute VBoxManage with given args", func() {
			stdout, err := driver.VBoxManage("help")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(stdout)).To(ContainSubstring("Oracle VM VirtualBox Command Line Management Interface"))
		})

		It("should return any errors", func() {
			stdout, err := driver.VBoxManage("some-bad-command")
			Expect(err).To(HaveOccurred())
			Expect(string(stdout)).To(BeEmpty())
		})
	})

	Describe("StartVM and StopVM and DestroyVM", func() {
		It("Should start, stop, and then destroy a VBox VM", func() {
			sshClient := &ssh.SSH{}
			err := driver.StartVM("Snappy")
			Expect(err).NotTo(HaveOccurred())
			Expect(driver.IsVMRunning("Snappy")).To(BeTrue())
			client, err := sshClient.WaitForSSH(&cssh.ClientConfig{
				User: "ubuntu",
				Auth: []cssh.AuthMethod{
					cssh.Password("vagrant"),
				},
			}, "2222")
			Expect(err).NotTo(HaveOccurred())
			defer client.Close()

			err = driver.StopVM("Snappy")
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() bool { return driver.IsVMRunning("Snappy") }, 120*time.Second).Should(BeFalse())

			err = driver.DestroyVM("Snappy")
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() bool {
				exists, err := driver.VMExists("Snappy")
				Expect(err).NotTo(HaveOccurred())
				return exists
			}, 120*time.Second).Should(BeFalse())
		})

		It("Should destroy the VBox VM network interface", func() {
			vboxnet, err := driver.CreateHostOnlyInterface("192.168.88.1")
			Expect(err).NotTo(HaveOccurred())

			err = driver.AttachNetworkInterface(vboxnet, "Snappy")
			Expect(err).NotTo(HaveOccurred())

			err = driver.StartVM("Snappy")
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() string {
				exists, err := driver.GetVBoxNetName("Snappy")
				Expect(err).NotTo(HaveOccurred())
				return string(exists)
			}).Should(ContainSubstring(vboxnet))
			Expect(driver.VBoxManage("list", "hostonlyifs")).To(ContainSubstring(vboxnet))

			_, err = driver.VBoxManage("controlvm", "Snappy", "poweroff")
			Expect(err).NotTo(HaveOccurred())

			err = driver.DestroyVM("Snappy")
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() string {
				exists, err := driver.VBoxManage("list", "hostonlyifs")
				Expect(err).NotTo(HaveOccurred())
				return string(exists)
			}, 120*time.Second).ShouldNot(ContainSubstring(vboxnet))
		})
	})

	Describe("#VMExists", func() {
		Context("VM Exists", func() {
			It("returns true", func() {
				exists, err := driver.VMExists("Snappy")
				Expect(err).NotTo(HaveOccurred())
				Expect(exists).To(BeTrue())
			})
		})

		Context("VM does Not exist", func() {
			It("returns false", func() {
				exists, err := driver.VMExists("does-not-exist")
				Expect(err).NotTo(HaveOccurred())
				Expect(exists).To(BeFalse())
			})
		})
	})

	Describe("#StartVM", func() {
		Context("VM with given name does not exist", func() {
			It("should return an error", func() {
				err := driver.StartVM("some-bad-vm-name")
				Expect(err).To(MatchError("failed to execute 'VBoxManage startvm some-bad-vm-name --type headless': exit status 1"))
			})
		})
	})

	Describe("#StopVM", func() {
		Context("VM with given name does not exist", func() {
			It("should return an error", func() {
				err := driver.StopVM("some-bad-vm-name")
				Expect(err).To(MatchError("failed to execute 'VBoxManage controlvm some-bad-vm-name acpipowerbutton': exit status 1"))
			})
		})
	})

	Describe("#DestroyVM", func() {
		Context("VM with given name does not exist", func() {
			It("should return an error", func() {
				err := driver.DestroyVM("some-bad-vm-name")
				Expect(err).To(MatchError("failed to execute 'VBoxManage showvminfo some-bad-vm-name --machinereadable': exit status 1"))
			})
		})
	})

	Describe("#CreateHostOnlyInterface", func() {
		AfterEach(func() {
			exec.Command("VBoxManage", "hostonlyif", "remove", "vboxnet1").Run()
		})

		It("Should create a hostonlyif", func() {
			name, err := driver.CreateHostOnlyInterface("192.168.77.1")
			Expect(err).NotTo(HaveOccurred())
			listCommand := exec.Command("VBoxManage", "list", "hostonlyifs")
			grepCommand := exec.Command("grep", name, "-A10")
			var output bytes.Buffer
			grepCommand.Stdin, err = listCommand.StdoutPipe()
			Expect(err).NotTo(HaveOccurred())
			grepCommand.Stdout = &output
			grepCommand.Start()
			err = listCommand.Run()
			Expect(err).NotTo(HaveOccurred())
			err = grepCommand.Wait()
			Expect(err).NotTo(HaveOccurred())

			Expect(name).To(ContainSubstring("vboxnet"))
			Expect(output.String()).To(MatchRegexp(`Name:\s+` + name))
			Expect(output.String()).To(MatchRegexp(`IPAddress:\s+192.168.77.1`))
			Expect(output.String()).To(MatchRegexp(`NetworkMask:\s+255.255.255.0`))
		})
	})

	Describe("#DestroyHostOnlyInterface", func() {
		AfterEach(func() {
			exec.Command("VBoxManage", "hostonlyif", "remove", "vboxnet1").Run()
		})

		It("Should destroy a hostonlyif", func() {
			name, err := driver.CreateHostOnlyInterface("192.168.77.1")
			Expect(err).NotTo(HaveOccurred())

			err = driver.DestroyHostOnlyInterface(name)
			Expect(err).NotTo(HaveOccurred())
			listCommand := exec.Command("VBoxManage", "list", "hostonlyifs")
			session, err := gexec.Start(listCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
			Expect(session).NotTo(gbytes.Say(name))
		})
	})

	Describe("#AttachInterface", func() {
		BeforeEach(func() {
			_, err := driver.CreateHostOnlyInterface("192.168.77.1")
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			exec.Command("VBoxManage", "hostonlyif", "remove", "vboxnet1").Run()
		})

		It("Should attach a network interface to the vm", func() {
			err := driver.AttachNetworkInterface("vboxnet1", "Snappy")
			Expect(err).NotTo(HaveOccurred())

			showvmInfoCommand := exec.Command("VBoxManage", "showvminfo", "Snappy", "--machinereadable")
			session, err := gexec.Start(showvmInfoCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
			Expect(session).To(gbytes.Say(`hostonlyadapter2="vboxnet1"`))
			Expect(session).To(gbytes.Say(`nic2="hostonly"`))
		})

		Context("fails to attach interface", func() {
			It("returns an error", func() {
				name := "some-bad-vm"
				err := driver.AttachNetworkInterface("vboxnet1", name)
				Expect(err).To(MatchError("failed to execute 'VBoxManage modifyvm some-bad-vm --nic2 hostonly --hostonlyadapter2 vboxnet1': exit status 1"))
			})
		})
	})

	Describe("#GetHostForwardPort", func() {
		It("Returns the port of the forwarded port on the host", func() {
			err := driver.ForwardPort("Snappy", "some-rule-name", "22", "2738")
			Expect(err).NotTo(HaveOccurred())

			port, err := driver.GetHostForwardPort("Snappy", "some-rule-name")
			Expect(err).NotTo(HaveOccurred())

			Expect(port).To(Equal("2738"))
		})
	})

	Describe("#ForwardPort", func() {
		It("Should forward guest port to the given host port", func() {
			err := driver.ForwardPort("Snappy", "some-rule-name", "22", "2739")
			Expect(err).NotTo(HaveOccurred())
			err = driver.StartVM("Snappy")
			Expect(err).NotTo(HaveOccurred())
			sshClient := &ssh.SSH{}
			client, err := sshClient.WaitForSSH(&cssh.ClientConfig{
				User: "ubuntu",
				Auth: []cssh.AuthMethod{
					cssh.Password("vagrant"),
				},
			}, "2739")
			Expect(err).NotTo(HaveOccurred())
			client.Close()
		})
	})

	Describe("#IsVMRunning", func() {
		Context("VM does not exist", func() {
			It("Should return false", func() {
				Expect(driver.IsVMRunning("some-bad-vm")).To(BeFalse())
			})
		})

		Context("VM is not running", func() {
			It("Should return false", func() {
				Expect(driver.IsVMRunning("Snappy")).To(BeFalse())
			})
		})

		Context("VM is running", func() {
			It("Should return true", func() {
				sshClient := &ssh.SSH{}
				err := driver.StartVM("Snappy")
				Expect(err).NotTo(HaveOccurred())
				Expect(driver.IsVMRunning("Snappy")).To(BeTrue())

				client, err := sshClient.WaitForSSH(&cssh.ClientConfig{
					User: "ubuntu",
					Auth: []cssh.AuthMethod{
						cssh.Password("vagrant"),
					},
				}, "2222")
				Expect(err).NotTo(HaveOccurred())
				client.Close()
			})
		})
	})
})

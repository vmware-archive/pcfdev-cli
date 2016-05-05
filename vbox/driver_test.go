package vbox_test

import (
	"bytes"
	"io/ioutil"
	"os/exec"
	"time"

	"github.com/pivotal-cf/pcfdev-cli/helpers"
	"github.com/pivotal-cf/pcfdev-cli/ssh"
	"github.com/pivotal-cf/pcfdev-cli/vbox"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("driver", func() {
	var driver *vbox.VBoxDriver
	var vmName string

	BeforeEach(func() {
		driver = &vbox.VBoxDriver{}

		var err error
		vmName, err = helpers.ImportSnappy()
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		driver.VBoxManage("controlvm", vmName, "poweroff")
		driver.VBoxManage("unregistervm", vmName, "--delete")
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

	Describe("when starting and stopping and destroying the VM", func() {
		It("should start, stop, and then destroy a VBox VM", func() {
			sshClient := &ssh.SSH{}
			_, port, err := sshClient.GenerateAddress()
			Expect(err).NotTo(HaveOccurred())

			err = driver.ForwardPort(vmName, "some-rule-name", port, "22")
			Expect(err).NotTo(HaveOccurred())

			err = driver.StartVM(vmName)
			Expect(err).NotTo(HaveOccurred())
			Expect(driver.IsVMRunning(vmName)).To(BeTrue())

			stdout := gbytes.NewBuffer()
			err = sshClient.RunSSHCommand("hostname", port, 5*time.Minute, stdout, ioutil.Discard)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(stdout.Contents())).To(ContainSubstring("ubuntu-core-stable-15"))

			err = driver.StopVM(vmName)
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() bool { return driver.IsVMRunning(vmName) }, 120*time.Second).Should(BeFalse())

			Expect(driver.StartVM(vmName)).To(Succeed())
			Expect(driver.IsVMRunning(vmName)).To(BeTrue())

			Expect(driver.PowerOffVM(vmName)).To(Succeed())
			Expect(driver.IsVMRunning(vmName)).To(BeFalse())

			err = driver.DestroyVM(vmName)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() bool {
				exists, err := driver.VMExists(vmName)
				Expect(err).NotTo(HaveOccurred())
				return exists
			}, 120*time.Second).Should(BeFalse())
		})

		It("should destroy the VBox VM network interface", func() {
			interfaceName, err := driver.CreateHostOnlyInterface("192.168.88.1")
			Expect(err).NotTo(HaveOccurred())

			err = driver.AttachNetworkInterface(interfaceName, vmName)
			Expect(err).NotTo(HaveOccurred())

			err = driver.StartVM(vmName)
			Expect(err).NotTo(HaveOccurred())

			Expect(driver.VBoxManage("showvminfo", vmName, "--machinereadable")).To(ContainSubstring(interfaceName))
			Expect(driver.VBoxManage("list", "hostonlyifs")).To(ContainSubstring(interfaceName))

			_, err = driver.VBoxManage("controlvm", vmName, "poweroff")
			Expect(err).NotTo(HaveOccurred())

			err = driver.DestroyVM(vmName)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() string {
				exists, err := driver.VBoxManage("list", "hostonlyifs")
				Expect(err).NotTo(HaveOccurred())
				return string(exists)
			}, 120*time.Second).ShouldNot(ContainSubstring(interfaceName))
		})
	})

	Describe("#StartVM", func() {
		Context("when VM with the given name does not exist", func() {
			It("should return an error", func() {
				err := driver.StartVM("some-bad-vm-name")
				Expect(err).To(MatchError("failed to execute 'VBoxManage startvm some-bad-vm-name --type headless': exit status 1"))
			})
		})
	})

	Describe("#VMExists", func() {
		Context("when VM exists", func() {
			It("should return true", func() {
				exists, err := driver.VMExists(vmName)
				Expect(err).NotTo(HaveOccurred())
				Expect(exists).To(BeTrue())
			})
		})

		Context("when VM does not exist", func() {
			It("should return false", func() {
				exists, err := driver.VMExists("does-not-exist")
				Expect(err).NotTo(HaveOccurred())
				Expect(exists).To(BeFalse())
			})
		})
	})

	Describe("#IsVMRunning", func() {
		Context("when VM does not exist", func() {
			It("should return false", func() {
				Expect(driver.IsVMRunning("some-bad-vm")).To(BeFalse())
			})
		})

		Context("when VM is not running", func() {
			It("should return false", func() {
				Expect(driver.IsVMRunning(vmName)).To(BeFalse())
			})
		})

		Context("when VM is running", func() {
			It("should return true", func() {
				sshClient := &ssh.SSH{}

				_, port, err := sshClient.GenerateAddress()
				Expect(err).NotTo(HaveOccurred())

				err = driver.ForwardPort(vmName, "some-rule-name", port, "22")
				Expect(err).NotTo(HaveOccurred())

				err = driver.StartVM(vmName)
				Expect(err).NotTo(HaveOccurred())
				Expect(driver.IsVMRunning(vmName)).To(BeTrue())

				stdout := gbytes.NewBuffer()
				err = sshClient.RunSSHCommand("hostname", port, 5*time.Minute, stdout, ioutil.Discard)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(stdout.Contents())).To(ContainSubstring("ubuntu-core-stable-15"))
			})
		})
	})

	Describe("#StopVM", func() {
		Context("when VM with the given name does not exist", func() {
			It("should return an error", func() {
				err := driver.StopVM("some-bad-vm-name")
				Expect(err).To(MatchError("failed to execute 'VBoxManage controlvm some-bad-vm-name acpipowerbutton': exit status 1"))
			})
		})
	})

	Describe("#PowerOffVM", func() {
		Context("when VM with the given name does not exist", func() {
			It("should return an error", func() {
				Expect(driver.PowerOffVM("some-bad-vm-name")).To(
					MatchError("failed to execute 'VBoxManage controlvm some-bad-vm-name poweroff': exit status 1"),
				)
			})
		})
	})

	Describe("#DestroyVM", func() {
		Context("when VM with the given name does not exist", func() {
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

		It("should create a hostonlyif", func() {
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

	Describe("#AttachInterface", func() {
		BeforeEach(func() {
			_, err := driver.CreateHostOnlyInterface("192.168.77.1")
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			exec.Command("VBoxManage", "hostonlyif", "remove", "vboxnet1").Run()
		})

		It("should attach a hostonlyif to the vm", func() {
			err := driver.AttachNetworkInterface("vboxnet1", vmName)
			Expect(err).NotTo(HaveOccurred())

			showvmInfoCommand := exec.Command("VBoxManage", "showvminfo", vmName, "--machinereadable")
			session, err := gexec.Start(showvmInfoCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
			Expect(session).To(gbytes.Say(`hostonlyadapter2="vboxnet1"`))
			Expect(session).To(gbytes.Say(`nic2="hostonly"`))
		})

		Context("when attaching a hostonlyif fails", func() {
			It("should return an error", func() {
				name := "some-bad-vm"
				err := driver.AttachNetworkInterface("vboxnet1", name)
				Expect(err).To(MatchError("failed to execute 'VBoxManage modifyvm some-bad-vm --nic2 hostonly --hostonlyadapter2 vboxnet1': exit status 1"))
			})
		})
	})

	Describe("#ForwardPort", func() {
		It("should forward guest port to the given host port", func() {
			sshClient := &ssh.SSH{}
			_, port, err := sshClient.GenerateAddress()
			Expect(err).NotTo(HaveOccurred())

			err = driver.ForwardPort(vmName, "some-rule-name", port, "22")
			Expect(err).NotTo(HaveOccurred())
			err = driver.StartVM(vmName)
			Expect(err).NotTo(HaveOccurred())

			stdout := gbytes.NewBuffer()
			err = sshClient.RunSSHCommand("hostname", port, 5*time.Minute, stdout, ioutil.Discard)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(stdout.Contents())).To(ContainSubstring("ubuntu-core-stable-15"))
		})
	})

	Describe("#GetHostForwardPort", func() {
		It("should return the forwarded port on the host", func() {
			sshClient := &ssh.SSH{}
			_, expectedPort, err := sshClient.GenerateAddress()
			Expect(err).NotTo(HaveOccurred())

			err = driver.ForwardPort(vmName, "some-rule-name", expectedPort, "22")
			Expect(err).NotTo(HaveOccurred())

			port, err := driver.GetHostForwardPort(vmName, "some-rule-name")
			Expect(err).NotTo(HaveOccurred())

			Expect(port).To(Equal(expectedPort))
		})

		Context("when no port is forwarded", func() {
			It("should return an error", func() {
				_, err := driver.GetHostForwardPort(vmName, "some-bad-rule-name")
				Expect(err).To(MatchError("could not find forwarded port"))
			})
		})
	})

	Describe("#VMs", func() {
		It("should return a list of VMs", func() {
			Expect(driver.VMs()).To(ContainElement(vmName))

			Expect(driver.StartVM(vmName)).To(Succeed())

			Expect(driver.VMs()).To(ContainElement(vmName))
		})
	})

	Describe("#RunningVMs", func() {
		It("should return a list of running VMs", func() {
			Expect(driver.RunningVMs()).NotTo(ContainElement(vmName))

			Expect(driver.StartVM(vmName)).To(Succeed())

			Expect(driver.RunningVMs()).To(ContainElement(vmName))
		})
	})
})

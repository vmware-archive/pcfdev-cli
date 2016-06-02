package vbox_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os/exec"
	"regexp"
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
	})

	Describe("#VBoxManage", func() {
		It("should execute VBoxManage with given args", func() {
			output, err := driver.VBoxManage("help")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(output)).To(ContainSubstring("Oracle VM VirtualBox Command Line Management Interface"))
		})

		It("should return any errors with their output", func() {
			output, err := driver.VBoxManage("some-bad-command")
			Expect(err).To(HaveOccurred())
			Expect(string(output)).To(ContainSubstring("Syntax error: Invalid command 'some-bad-command'"))
		})
	})

	Describe("#GetVMIP", func() {
		Context("when interface exists", func() {
			var interfaceName string

			BeforeEach(func() {
				var err error
				interfaceName, err = driver.CreateHostOnlyInterface("192.168.88.1")
				Expect(err).NotTo(HaveOccurred())

				err = driver.AttachNetworkInterface(interfaceName, vmName)
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				command := exec.Command("VBoxManage", "hostonlyif", "remove", interfaceName)
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session, 10*time.Second).Should(gexec.Exit(0))
			})

			It("should return the ip of the vm", func() {
				ip, err := driver.GetVMIP(vmName)
				Expect(err).NotTo(HaveOccurred())
				Expect(ip).To(Equal("192.168.88.11"))
			})
		})

		Context("when interface does not exist", func() {
			It("should return an error message", func() {
				_, err := driver.GetVMIP(vmName)
				Expect(err).To(MatchError("there is no attached hostonlyif for " + vmName))
			})
		})

		Context("when VBoxManage command fails", func() {
			It("should return the output of the failed command", func() {
				_, err := driver.GetVMIP("some-bad-vm-name")
				Expect(err).To(MatchError(ContainSubstring("failed to execute 'VBoxManage showvminfo some-bad-vm-name --machinereadable': exit status 1")))
				Expect(err).To(MatchError(ContainSubstring("Could not find a registered machine named 'some-bad-vm-name'")))
			})
		})
	})

	Describe("when starting and stopping and suspending and resuming and destroying the VM", func() {
		It("should start, stop, suspend, resume, and then destroy a VBox VM", func() {
			sshClient := &ssh.SSH{}
			_, port, err := sshClient.GenerateAddress()
			Expect(err).NotTo(HaveOccurred())

			err = driver.ForwardPort(vmName, "some-rule-name", port, "22")
			Expect(err).NotTo(HaveOccurred())

			err = driver.StartVM(vmName)
			Expect(err).NotTo(HaveOccurred())
			Expect(driver.VMState(vmName)).To(Equal(vbox.StateRunning))

			stdout := gbytes.NewBuffer()
			err = sshClient.RunSSHCommand("hostname", port, 5*time.Minute, stdout, ioutil.Discard)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(stdout.Contents())).To(ContainSubstring("ubuntu-core-stable-15"))

			err = driver.StopVM(vmName)
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() (string, error) { return driver.VMState(vmName) }, 120*time.Second).Should(Equal(vbox.StateStopped))

			Expect(driver.StartVM(vmName)).To(Succeed())
			Expect(driver.VMState(vmName)).To(Equal(vbox.StateRunning))

			err = driver.SuspendVM(vmName)
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() (string, error) { return driver.VMState(vmName) }, 120*time.Second).Should(Equal(vbox.StateSaved))

			err = driver.ResumeVM(vmName)
			Expect(err).NotTo(HaveOccurred())
			Expect(driver.VMState(vmName)).To(Equal(vbox.StateRunning))

			Expect(driver.PowerOffVM(vmName)).To(Succeed())
			Expect(driver.VMState(vmName)).To(Equal(vbox.StateStopped))

			err = driver.DestroyVM(vmName)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() bool {
				exists, err := driver.VMExists(vmName)
				Expect(err).NotTo(HaveOccurred())
				return exists
			}, 120*time.Second).Should(BeFalse())
		})
	})

	Describe("#StartVM", func() {
		Context("when VM with the given name does not exist", func() {
			It("should return an error", func() {
				err := driver.StartVM("some-bad-vm-name")
				Expect(err).To(MatchError(ContainSubstring("failed to execute 'VBoxManage startvm some-bad-vm-name --type headless': exit status 1")))
				Expect(err).To(MatchError(ContainSubstring("Could not find a registered machine named 'some-bad-vm-name'")))
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

	Describe("#VMState", func() {
		Context("when the VM is running", func() {
			It("should return StateRunning", func() {
				sshClient := &ssh.SSH{}

				_, port, err := sshClient.GenerateAddress()
				Expect(err).NotTo(HaveOccurred())

				err = driver.ForwardPort(vmName, "some-rule-name", port, "22")
				Expect(err).NotTo(HaveOccurred())

				err = driver.StartVM(vmName)
				Expect(err).NotTo(HaveOccurred())
				state, err := driver.VMState(vmName)
				Expect(err).NotTo(HaveOccurred())
				Expect(state).To(Equal(vbox.StateRunning))
			})
		})

		Context("when the VM is saved", func() {
			It("should return StateSaved", func() {
				sshClient := &ssh.SSH{}

				_, port, err := sshClient.GenerateAddress()
				Expect(err).NotTo(HaveOccurred())

				err = driver.ForwardPort(vmName, "some-rule-name", port, "22")
				Expect(err).NotTo(HaveOccurred())

				err = driver.StartVM(vmName)
				Expect(err).NotTo(HaveOccurred())
				err = driver.SuspendVM(vmName)

				Expect(driver.VMState(vmName)).To(Equal(vbox.StateSaved))
			})
		})

		Context("when the VM is stopped", func() {
			It("should return StateStopped", func() {
				Expect(driver.VMState(vmName)).To(Equal(vbox.StateStopped))
			})
		})

		Context("when VBoxManage command fails", func() {
			It("should return the output of the failed command", func() {
				_, err := driver.VMState("some-bad-vm-name")
				Expect(err).To(MatchError(ContainSubstring("failed to execute 'VBoxManage showvminfo some-bad-vm-name --machinereadable': exit status 1")))
				Expect(err).To(MatchError(ContainSubstring("Could not find a registered machine named 'some-bad-vm-name'")))
			})
		})
	})

	Describe("#StopVM", func() {
		Context("when VM with the given name does not exist", func() {
			It("should return an error", func() {
				err := driver.StopVM("some-bad-vm-name")
				Expect(err).To(MatchError(ContainSubstring("failed to execute 'VBoxManage controlvm some-bad-vm-name acpipowerbutton': exit status 1")))
				Expect(err).To(MatchError(ContainSubstring("Could not find a registered machine named 'some-bad-vm-name'")))
			})
		})
	})

	Describe("#SuspendVM", func() {
		Context("when VM with the given name does not exist", func() {
			It("should return an error", func() {
				err := driver.SuspendVM("some-bad-vm-name")
				Expect(err).To(MatchError(ContainSubstring("failed to execute 'VBoxManage controlvm some-bad-vm-name savestate': exit status 1")))
				Expect(err).To(MatchError(ContainSubstring("Could not find a registered machine named 'some-bad-vm-name'")))
			})
		})
	})

	Describe("#ResumeVM", func() {
		Context("when VM with the given name does not exist", func() {
			It("should return an error", func() {
				err := driver.ResumeVM("some-bad-vm-name")
				Expect(err).To(MatchError(ContainSubstring("failed to execute 'VBoxManage startvm some-bad-vm-name --type headless': exit status 1")))
				Expect(err).To(MatchError(ContainSubstring("Could not find a registered machine named 'some-bad-vm-name'")))
			})
		})
	})

	Describe("#PowerOffVM", func() {
		Context("when VM with the given name does not exist", func() {
			It("should return an error", func() {
				err := driver.PowerOffVM("some-bad-vm-name")
				Expect(err).To(MatchError(ContainSubstring("failed to execute 'VBoxManage controlvm some-bad-vm-name poweroff': exit status 1")))
				Expect(err).To(MatchError(ContainSubstring("Could not find a registered machine named 'some-bad-vm-name'")))
			})
		})
	})

	Describe("#DestroyVM", func() {
		Context("when VM with the given name does not exist", func() {
			It("should return an error", func() {
				err := driver.DestroyVM("some-bad-vm-name")
				Expect(err).To(MatchError(ContainSubstring("failed to execute 'VBoxManage unregistervm some-bad-vm-name --delete': exit status 1")))
				Expect(err).To(MatchError(ContainSubstring("Could not find a registered machine named 'some-bad-vm-name'")))
			})
		})
	})

	Describe("#CreateHostOnlyInterface", func() {
		var interfaceName string

		AfterEach(func() {
			command := exec.Command("VBoxManage", "hostonlyif", "remove", interfaceName)
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		})

		It("should create a hostonlyif", func() {
			var err error
			interfaceName, err = driver.CreateHostOnlyInterface("192.168.77.1")
			Expect(err).NotTo(HaveOccurred())
			listCommand := exec.Command("VBoxManage", "list", "hostonlyifs")
			grepCommand := exec.Command("grep", interfaceName, "-A10")
			var output bytes.Buffer
			grepCommand.Stdin, err = listCommand.StdoutPipe()
			Expect(err).NotTo(HaveOccurred())
			grepCommand.Stdout = &output
			grepCommand.Start()
			err = listCommand.Run()
			Expect(err).NotTo(HaveOccurred())
			err = grepCommand.Wait()
			Expect(err).NotTo(HaveOccurred())

			Expect(output.String()).To(MatchRegexp(`Name:\s+` + interfaceName))
			Expect(output.String()).To(MatchRegexp(`IPAddress:\s+192.168.77.1`))
			Expect(output.String()).To(MatchRegexp(`NetworkMask:\s+255.255.255.0`))
		})
	})

	Describe("#GetHostOnlyInterfaces", func() {
		var interfaceName string
		var expectedIP string

		BeforeEach(func() {
			expectedIP = "192.168.55.55"
			output, err := exec.Command("VBoxManage", "hostonlyif", "create").Output()
			Expect(err).NotTo(HaveOccurred())
			regex := regexp.MustCompile(`Interface '(.*)' was successfully created`)
			matches := regex.FindStringSubmatch(string(output))
			interfaceName = matches[1]
			assignIP := exec.Command("VBoxManage", "hostonlyif", "ipconfig", interfaceName, "--ip", expectedIP, "--netmask", "255.255.255.0")
			session, err := gexec.Start(assignIP, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
		})

		AfterEach(func() {
			command := exec.Command("VBoxManage", "hostonlyif", "remove", interfaceName)
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		})

		It("should return a slice of network.Interfaces representing the list of VBox nets", func() {
			interfaces, err := driver.GetHostOnlyInterfaces()
			Expect(err).NotTo(HaveOccurred())

			for _, iface := range interfaces {
				if iface.Name == interfaceName {
					Expect(iface.IP).To(Equal(expectedIP))
					return
				}
			}
			Fail(fmt.Sprintf("did not create interface with expected name %s", interfaceName))
		})
	})

	Describe("#AttachInterface", func() {
		var interfaceName string

		BeforeEach(func() {
			var err error
			interfaceName, err = driver.CreateHostOnlyInterface("192.168.77.1")
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			command := exec.Command("VBoxManage", "hostonlyif", "remove", interfaceName)
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		})

		It("should attach a hostonlyif to the vm", func() {
			err := driver.AttachNetworkInterface(interfaceName, vmName)
			Expect(err).NotTo(HaveOccurred())

			showvmInfoCommand := exec.Command("VBoxManage", "showvminfo", vmName, "--machinereadable")
			session, err := gexec.Start(showvmInfoCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
			Expect(session).To(gbytes.Say(`hostonlyadapter2="` + interfaceName + `"`))
			Expect(session).To(gbytes.Say(`nic2="hostonly"`))
		})

		Context("when attaching a hostonlyif fails", func() {
			It("should return an error", func() {
				err := driver.AttachNetworkInterface("some-interface-name", "some-bad-vm-name")
				Expect(err).To(MatchError(ContainSubstring("failed to execute 'VBoxManage modifyvm some-bad-vm-name --nic2 hostonly --hostonlyadapter2 some-interface-name': exit status 1")))
				Expect(err).To(MatchError(ContainSubstring("Could not find a registered machine named 'some-bad-vm-name'")))
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

		Context("when forwarding a port fails", func() {
			It("should return an error", func() {
				err := driver.ForwardPort("some-bad-vm-name", "some-rule-name", "some-host-port", "some-guest-port")
				Expect(err).To(MatchError(ContainSubstring("failed to execute 'VBoxManage modifyvm some-bad-vm-name --natpf1 some-rule-name,tcp,127.0.0.1,some-host-port,,some-guest-port': exit status 1")))
				Expect(err).To(MatchError(ContainSubstring("Could not find a registered machine named 'some-bad-vm-name'")))
			})
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

		Context("when VM with the given name does not exist", func() {
			It("should return an error", func() {
				_, err := driver.GetHostForwardPort("some-bad-vm-name", "some-rule-name")
				Expect(err).To(MatchError(ContainSubstring("failed to execute 'VBoxManage showvminfo some-bad-vm-name --machinereadable': exit status 1")))
				Expect(err).To(MatchError(ContainSubstring("Could not find a registered machine named 'some-bad-vm-name'")))
			})
		})
	})

	Describe("#SetMemory", func() {
		It("should set vm memory in mb", func() {
			Expect(driver.SetMemory(vmName, uint64(2048))).To(Succeed())

			showvmInfoCommand := exec.Command("VBoxManage", "showvminfo", vmName, "--machinereadable")
			session, err := gexec.Start(showvmInfoCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
			Expect(session).To(gbytes.Say(`memory=2048`))
		})

		Context("when setting memory fails", func() {
			It("should return an error", func() {
				err := driver.SetMemory("some-bad-vm-name", uint64(0))
				Expect(err).To(MatchError(ContainSubstring("failed to execute 'VBoxManage modifyvm some-bad-vm-name --memory 0': exit status 1")))
				Expect(err).To(MatchError(ContainSubstring("Could not find a registered machine named 'some-bad-vm-name'")))
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

	Describe("#GetVirtualSystemNumbersOfHardDiskImages", func() {
		It("should the virtual system number of the hard disk image", func() {
			ovaPath := "../assets/snappy.ova"
			Expect(driver.GetVirtualSystemNumbersOfHardDiskImages(ovaPath)).To(ConsistOf([]string{"6", "7"}))
		})

		Context("when there is an error getting the virtual system number", func() {
			It("should return an error", func() {
				numbers, err := driver.GetVirtualSystemNumbersOfHardDiskImages("some-bad-ova-path.ova")
				Expect(err).To(MatchError(ContainSubstring("failed to execute 'VBoxManage import some-bad-ova-path.ova -n': exit status 1")))
				Expect(err).To(MatchError(ContainSubstring("Could not open the OVA file")))
				Expect(numbers).To(BeNil())
			})
		})
	})
})

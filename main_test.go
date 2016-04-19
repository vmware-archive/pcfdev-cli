package main_test

import (
	"os"
	"os/exec"
	"regexp"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var context struct {
	cfHome string
}

var _ = BeforeSuite(func() {
	ifconfig := exec.Command("ifconfig")
	session, err := gexec.Start(ifconfig, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Expect(session.Wait().Out.Contents()).NotTo(ContainSubstring("192.168.11.1"))

	uninstallCommand := exec.Command("cf", "uninstall-plugin", "pcfdev")
	session, err = gexec.Start(uninstallCommand, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session, "10s").Should(gexec.Exit())

	pluginPath, err := gexec.Build("github.com/pivotal-cf/pcfdev-cli")
	Expect(err).NotTo(HaveOccurred())
	installCommand := exec.Command("cf", "install-plugin", "-f", pluginPath)
	session, err = gexec.Start(installCommand, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session, "1m").Should(gexec.Exit(0))
})

var _ = AfterSuite(func() {
	uninstallCommand := exec.Command("cf", "uninstall-plugin", "pcfdev")
	session, err := gexec.Start(uninstallCommand, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session, "10s").Should(gexec.Exit(0))
})

var _ = Describe("pcfdev", func() {
	Context("pivnet api token is set in environment", func() {
		It("should start, stop, and destroy a virtualbox instance", func() {
			pcfdevCommand := exec.Command("cf", "dev", "start")
			session, err := gexec.Start(pcfdevCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, "1h").Should(gexec.Exit(0))
			Expect(session).To(gbytes.Say("PCF Dev is now running"))
			Expect(isVMRunning()).To(BeTrue())

			// rerunning start has no effect
			restartCommand := exec.Command("cf", "dev", "start")
			session, err = gexec.Start(restartCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, "1m").Should(gexec.Exit(0))
			Expect(session).To(gbytes.Say("PCF Dev is already running"))
			Expect(isVMRunning()).To(BeTrue())

			Eventually(cf("api", "api.local.pcfdev.io", "--skip-ssl-validation")).Should(gexec.Exit(0))
			Eventually(cf("auth", "admin", "admin")).Should(gexec.Exit(0))

			pcfdevCommand = exec.Command("cf", "dev", "stop")
			session, err = gexec.Start(pcfdevCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, "10m").Should(gexec.Exit(0))
			Expect(session).To(gbytes.Say("PCF Dev is now stopped"))
			Expect(isVMRunning()).NotTo(BeTrue())

			pcfdevCommand = exec.Command("cf", "dev", "destroy")
			session, err = gexec.Start(pcfdevCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, "10m").Should(gexec.Exit(0))
			Expect(session).To(gbytes.Say("PCF Dev VM has been destroyed"))
		})
		It("should respond to pcfdev alias", func() {
			pcfdevCommand := exec.Command("cf", "pcfdev")
			session, err := gexec.Start(pcfdevCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(1))
			Expect(session).To(gbytes.Say("Usage: cf dev import|start|stop"))
		})
		It("should import a VM without starting it", func() {
			pcfdevCommand := exec.Command("cf", "dev", "import")
			session, err := gexec.Start(pcfdevCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, "1h").Should(gexec.Exit(0))

			listVmsCommand := exec.Command("VboxManage", "list", "vms")
			session, err = gexec.Start(listVmsCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
			Expect(session).To(gbytes.Say("pcfdev-2016-03-29_1728"))
		})

		AfterEach(func() {
			output, err := exec.Command("VBoxManage", "showvminfo", "pcfdev-2016-03-29_1728", "--machinereadable").Output()
			if err != nil {
				return
			}

			regex := regexp.MustCompile(`hostonlyadapter2="(.*)"`)
			vboxnet := regex.FindStringSubmatch(string(output))[1]

			exec.Command("VBoxManage", "controlvm", "pcfdev-2016-03-29_1728", "poweroff").Run()
			exec.Command("VBoxManage", "unregistervm", "pcfdev-2016-03-29_1728", "--delete").Run()
			exec.Command("VBoxManage", "hostonlyif", "remove", vboxnet).Run()
		})
	})
})

func loadEnv(name string) string {
	value := os.Getenv(name)
	if value == "" {
		Fail("missing "+name, 1)
	}
	return value
}

func cf(args ...string) *gexec.Session {
	command := exec.Command("cf", args...)
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	return session
}

func isVMRunning() bool {
	vmStatus, err := exec.Command("VBoxManage", "showvminfo", "pcfdev-2016-03-29_1728", "--machinereadable").Output()
	Expect(err).NotTo(HaveOccurred())
	return strings.Contains(string(vmStatus), `VMState="running"`)
}

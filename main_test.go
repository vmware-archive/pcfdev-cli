package main_test

import (
	"os"
	"os/exec"
	"strings"
	"time"

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

	uninstallCommand := exec.Command("cf", "uninstall-plugin", "PCFDev")
	session, err = gexec.Start(uninstallCommand, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session, "10s").Should(gexec.Exit())

	pluginPath, err := gexec.Build("github.com/pivotal-cf/pcfdev-cli")
	Expect(err).NotTo(HaveOccurred())
	installCommand := exec.Command("cf", "install-plugin", "-f", pluginPath)
	session, err = gexec.Start(installCommand, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session).Should(gexec.Exit(0))
})

var _ = AfterSuite(func() {
	uninstallCommand := exec.Command("cf", "uninstall-plugin", "PCFDev")
	session, err := gexec.Start(uninstallCommand, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session, "10s").Should(gexec.Exit(0))
})

var _ = Describe("PCFDev", func() {
	Context("pivnet api token is set in environment", func() {
		XIt("should start and stop a virtualbox instance", func() {
			pcfdevCommand := exec.Command("cf", "dev", "start")
			session, err := gexec.Start(pcfdevCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, "1h").Should(gexec.Exit(0))
			Expect(session).To(gbytes.Say("PCFDev is now running"))
			Expect(isVMRunning()).To(BeTrue())

			// rerunning start has no effect
			restartCommand := exec.Command("cf", "dev", "start")
			session, err = gexec.Start(restartCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, "1m").Should(gexec.Exit(0))
			Expect(session).To(gbytes.Say("PCFDev is already running"))
			Expect(isVMRunning()).To(BeTrue())

			Eventually(cf("api", "api.local.pcfdev.io", "--skip-ssl-validation")).Should(gexec.Exit(0))
			Eventually(cf("auth", "admin", "admin")).Should(gexec.Exit(0))

			pcfdevCommand = exec.Command("cf", "dev", "stop")
			session, err = gexec.Start(pcfdevCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, "10m").Should(gexec.Exit(0))
			Expect(session).To(gbytes.Say("PCFDev is now stopped"))
			Expect(isVMRunning()).NotTo(BeTrue())
		})
		It("should respond to pcfdev alias", func() {
			pcfdevCommand := exec.Command("cf", "pcfdev")
			session, err := gexec.Start(pcfdevCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(1))
			Expect(session).To(gbytes.Say("Usage: cf dev start|stop"))
		})

		AfterEach(func() {
			exec.Command("VBoxManage", "controlvm", "pcfdev-2016-03-29_1728", "acpipowerbutton").Run()

			for attempts := 0; attempts < 30; attempts++ {
				err := exec.Command("VBoxManage", "unregistervm", "pcfdev-2016-03-29_1728", "--delete").Run()
				if err == nil {
					break
				}
				time.Sleep(time.Second)
			}

			exec.Command("VBoxManage", "hostonlyif", "remove", "vboxnet0").Run()
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

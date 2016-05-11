package main_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

const (
	vmName = "pcfdev-test"
)

var (
	tempHome        string
	oldCFHome       string
	oldCFPluginHome string
	oldPCFDevHome   string
)

var _ = BeforeSuite(func() {
	oldCFHome = os.Getenv("CF_HOME")
	oldCFPluginHome = os.Getenv("CF_PLUGIN_HOME")
	oldPCFDevHome = os.Getenv("PCFDev_HOME")

	ifconfig := exec.Command("ifconfig")
	session, err := gexec.Start(ifconfig, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Expect(session.Wait().Out.Contents()).NotTo(ContainSubstring("192.168.11.1"))

	tempHome, err = ioutil.TempDir("", "pcfdev")
	Expect(err).NotTo(HaveOccurred())
	os.Setenv("CF_HOME", tempHome)
	os.Setenv("CF_PLUGIN_HOME", filepath.Join(tempHome, "plugins"))
	os.Setenv("PCFDEV_HOME", tempHome)

	uninstallCommand := exec.Command("cf", "uninstall-plugin", "pcfdev")
	session, err = gexec.Start(uninstallCommand, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session, "10s").Should(gexec.Exit())

	pluginPath, err := gexec.Build("github.com/pivotal-cf/pcfdev-cli", "-ldflags",
		"-X main.vmName="+vmName+
			" -X main.releaseId=1622"+
			" -X main.productFileId=4448"+
			" -X main.md5=05761a420b00028ae5384c6bc460a6ba")
	Expect(err).NotTo(HaveOccurred())

	installCommand := exec.Command("cf", "install-plugin", "-f", pluginPath)
	session, err = gexec.Start(installCommand, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session, "1m").Should(gexec.Exit(0))
})

var _ = AfterSuite(func() {
	Expect(os.RemoveAll(tempHome)).To(Succeed())
	os.Setenv("CF_HOME", oldCFHome)
	os.Setenv("CF_PLUGIN_HOME", oldCFPluginHome)
	os.Setenv("PCFDev_HOME", oldPCFDevHome)
})

var _ = Describe("pcfdev", func() {
	AfterEach(func() {
		output, err := exec.Command("VBoxManage", "showvminfo", vmName, "--machinereadable").Output()
		if err != nil {
			return
		}

		regex := regexp.MustCompile(`hostonlyadapter2="(.*)"`)
		interfaceName := regex.FindStringSubmatch(string(output))[1]

		exec.Command("VBoxManage", "controlvm", vmName, "poweroff").Run()
		exec.Command("VBoxManage", "unregistervm", vmName, "--delete").Run()
		exec.Command("VBoxManage", "hostonlyif", "remove", interfaceName).Run()
	})

	It("should start, stop, and destroy a virtualbox instance", func() {
		pcfdevCommand := exec.Command("cf", "dev", "start")
		session, err := gexec.Start(pcfdevCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session, "10m").Should(gexec.Exit(0))
		Expect(session).To(gbytes.Say("Waiting for services to start..."))
		Expect(session).To(gbytes.Say("Services started"))
		Expect(session).To(gbytes.Say("PCF Dev is now running"))
		Expect(isVMRunning()).To(BeTrue())

		By("re-running 'cf dev start' with no effect")
		restartCommand := exec.Command("cf", "dev", "start")
		session, err = gexec.Start(restartCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session, "2m").Should(gexec.Exit(0))
		Expect(session).To(gbytes.Say("PCF Dev is running"))
		Expect(isVMRunning()).To(BeTrue())

		response, err := getResponseFromFakeServer()
		Expect(err).NotTo(HaveOccurred())
		Expect(response).To(Equal("PCF Dev Test VM"))

		pcfdevCommand = exec.Command("cf", "dev", "stop")
		session, err = gexec.Start(pcfdevCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session, "2m").Should(gexec.Exit(0))
		Expect(session).To(gbytes.Say("PCF Dev is now stopped"))
		Expect(isVMRunning()).NotTo(BeTrue())

		pcfdevCommand = exec.Command("cf", "dev", "destroy")
		session, err = gexec.Start(pcfdevCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session, "2m").Should(gexec.Exit(0))
		Expect(session).To(gbytes.Say("PCF Dev VM has been destroyed"))

		By("re-running destroy with no effect")
		redestroyCommand := exec.Command("cf", "dev", "destroy")
		session, err = gexec.Start(redestroyCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session, "2m").Should(gexec.Exit(0))
		Expect(session).To(gbytes.Say("PCF Dev VM has not been created"))

		By("starting after running destroy")
		pcfdevCommand = exec.Command("cf", "dev", "start")
		session, err = gexec.Start(pcfdevCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session, "10m").Should(gexec.Exit(0))
		Expect(session).To(gbytes.Say("PCF Dev is now running"))
		Expect(isVMRunning()).To(BeTrue())

		response, err = getResponseFromFakeServer()
		Expect(err).NotTo(HaveOccurred())
		Expect(response).To(Equal("PCF Dev Test VM"))
	})

	It("should respond to pcfdev alias", func() {
		pcfdevCommand := exec.Command("cf", "pcfdev")
		session, err := gexec.Start(pcfdevCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session).Should(gexec.Exit(1))
		Expect(session).To(gbytes.Say(`Usage: cf dev download\|start\|status\|stop\|destroy`))
	})

	It("should download a VM without importing it", func() {
		pcfdevCommand := exec.Command("cf", "dev", "download")
		session, err := gexec.Start(pcfdevCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session, "1h").Should(gexec.Exit(0))

		_, err = os.Stat(filepath.Join(os.Getenv("PCFDEV_HOME"), "pcfdev.ova"))
		Expect(err).NotTo(HaveOccurred())

		listVmsCommand := exec.Command("VBoxManage", "list", "vms")
		session, err = gexec.Start(listVmsCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session).Should(gexec.Exit(0))
		Expect(session).NotTo(gbytes.Say(vmName))

		pcfdevCommand = exec.Command("cf", "dev", "download")
		session, err = gexec.Start(pcfdevCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session, "3m").Should(gexec.Exit(0))
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
	vmStatus, err := exec.Command("VBoxManage", "showvminfo", vmName, "--machinereadable").Output()
	Expect(err).NotTo(HaveOccurred())
	return strings.Contains(string(vmStatus), `VMState="running"`)
}

func getResponseFromFakeServer() (string, error) {
	timeoutChan := time.After(30 * time.Second)
	var response *http.Response
	var responseBody []byte
	var err error

	for {
		select {
		case <-timeoutChan:
			return "", fmt.Errorf("connection timed out: %s", err)
		default:
			response, err = http.Get("http://api.local.pcfdev.io")
			if err != nil {
				continue
			}
			defer response.Body.Close()

			responseBody, err = ioutil.ReadAll(response.Body)
			return string(responseBody), err
		}
	}
}

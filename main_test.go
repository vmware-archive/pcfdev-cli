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
	"github.com/pivotal-cf/pcfdev-cli/helpers"
	"github.com/pivotal-cf/pcfdev-cli/ssh"
)

const (
	vmName = "pcfdev-test"
)

var (
	tempHome        string
	oldCFHome       string
	oldCFPluginHome string
	oldPCFDevHome   string
	oldHTTPProxy    string
	oldHTTPSProxy   string
	oldNoProxy      string
	vBoxManagePath  string
)

var _ = BeforeSuite(func() {
	oldCFHome = os.Getenv("CF_HOME")
	oldCFPluginHome = os.Getenv("CF_PLUGIN_HOME")
	oldPCFDevHome = os.Getenv("PCFDEV_HOME")
	oldHTTPProxy = os.Getenv("HTTP_PROXY")
	oldHTTPSProxy = os.Getenv("HTTPS_PROXY")
	oldNoProxy = os.Getenv("NO_PROXY")

	var err error
	tempHome, err = ioutil.TempDir("", "pcfdev")

	Expect(err).NotTo(HaveOccurred())
	os.Setenv("CF_HOME", tempHome)
	os.Setenv("CF_PLUGIN_HOME", filepath.Join(tempHome, "plugins"))
	os.Setenv("PCFDEV_HOME", tempHome)
	oldHTTPSProxy = os.Getenv("HTTPS_PROXY")
	oldNoProxy = os.Getenv("NO_PROXY")

	uninstallCommand := exec.Command("cf", "uninstall-plugin", "pcfdev")
	session, err := gexec.Start(uninstallCommand, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session, "10s").Should(gexec.Exit())

	pluginPath, err := gexec.Build(filepath.Join("github.com", "pivotal-cf", "pcfdev-cli"), "-ldflags",
		"-X main.vmName="+vmName+
			" -X main.releaseId=1622"+
			" -X main.productFileId=4448"+
			" -X main.md5=05761a420b00028ae5384c6bc460a6ba")
	Expect(err).NotTo(HaveOccurred())

	installCommand := exec.Command("cf", "install-plugin", "-f", pluginPath)
	session, err = gexec.Start(installCommand, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session, "1m").Should(gexec.Exit(0))

	vBoxManagePath, err = helpers.VBoxManagePath()
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	Expect(os.RemoveAll(tempHome)).To(Succeed())
	os.Setenv("CF_HOME", oldCFHome)
	os.Setenv("CF_PLUGIN_HOME", oldCFPluginHome)
	os.Setenv("PCFDEV_HOME", oldPCFDevHome)
	os.Setenv("HTTP_PROXY", oldHTTPProxy)
	os.Setenv("HTTPS_PROXY", oldHTTPSProxy)
	os.Setenv("NO_PROXY", oldNoProxy)
})

var _ = Describe("pcfdev", func() {
	AfterEach(func() {
		output, err := exec.Command(vBoxManagePath, "showvminfo", vmName, "--machinereadable").Output()
		if err != nil {
			return
		}

		regex := regexp.MustCompile(`hostonlyadapter2="(.*)"`)
		matches := regex.FindStringSubmatch(string(output))

		exec.Command(vBoxManagePath, "controlvm", vmName, "poweroff").Run()
		exec.Command(vBoxManagePath, "unregistervm", vmName, "--delete").Run()

		if len(matches) > 1 {
			exec.Command(vBoxManagePath, "hostonlyif", "remove", matches[1]).Run()
		}
	})

	It("should start, stop, and destroy a virtualbox instance", func() {
		pcfdevCommand := exec.Command("cf", "dev", "start")
		session, err := gexec.Start(pcfdevCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session, "10m").Should(gexec.Exit(0))
		Expect(session).To(gbytes.Say("Waiting for services to start..."))
		Expect(session).To(gbytes.Say("Services started"))
		Expect(isVMRunning()).To(BeTrue())
		Expect(filepath.Join(tempHome, vmName+"-disk0.vmdk")).To(BeAnExistingFile())

		By("re-running 'cf dev start' with no effect")
		restartCommand := exec.Command("cf", "dev", "start")
		session, err = gexec.Start(restartCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session, "2m").Should(gexec.Exit(0))
		Expect(session).To(gbytes.Say("PCF Dev is running"))
		Expect(isVMRunning()).To(BeTrue())

		By("running 'cf dev suspend' should suspend the vm")
		suspendCommand := exec.Command("cf", "dev", "suspend")
		session, err = gexec.Start(suspendCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session, "2m").Should(gexec.Exit(0))
		Expect(session).To(gbytes.Say("Suspending VM..."))
		Expect(session).To(gbytes.Say("PCF Dev is now suspended"))
		Expect(isVMRunning()).NotTo(BeTrue())

		By("running 'cf dev resume' should resume the vm")
		resumeCommand := exec.Command("cf", "dev", "resume")
		session, err = gexec.Start(resumeCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session, "2m").Should(gexec.Exit(0))
		Expect(session).To(gbytes.Say("Resuming VM..."))
		Expect(isVMRunning()).To(BeTrue())

		output, err := exec.Command(vBoxManagePath, "showvminfo", vmName, "--machinereadable").Output()
		Expect(err).NotTo(HaveOccurred())
		regex := regexp.MustCompile(`hostonlyadapter2="(.*)"`)
		interfaceName := regex.FindStringSubmatch(string(output))[1]

		response, err := getResponseFromFakeServer(interfaceName)
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

		By("leaving up hostonly interface after destroy")
		vboxnets, err := exec.Command(vBoxManagePath, "list", "hostonlyifs").Output()
		Expect(err).NotTo(HaveOccurred())
		Expect(vboxnets).To(ContainSubstring(interfaceName))

		By("re-running destroy with no effect")
		redestroyCommand := exec.Command("cf", "dev", "destroy")
		session, err = gexec.Start(redestroyCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session, "2m").Should(gexec.Exit(0))
		Expect(session).To(gbytes.Say("PCF Dev VM has not been created"))

		By("setting proxy variables")
		os.Setenv("HTTP_PROXY", "192.168.93.23")
		os.Setenv("HTTPS_PROXY", "192.168.38.29")
		os.Setenv("NO_PROXY", "192.168.98.98")

		By("starting after running destroy")
		pcfdevCommand = exec.Command("cf", "dev", "start", "-m", "3456")
		session, err = gexec.Start(pcfdevCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session, "10m").Should(gexec.Exit(0))
		Expect(session).To(gbytes.Say("Waiting for services to start..."))
		Expect(session).To(gbytes.Say("Services started"))
		Expect(isVMRunning()).To(BeTrue())
		Expect(vmMemory()).To(Equal("3456"))

		stdout := gbytes.NewBuffer()
		stderr := gbytes.NewBuffer()
		sshClient := &ssh.SSH{}
		sshClient.RunSSHCommand("echo $HTTP_PROXY", getForwardedPort(), 5*time.Second, stdout, stderr)
		Eventually(stdout).Should(gbytes.Say("192.168.93.23"))
		sshClient.RunSSHCommand("echo $HTTPS_PROXY", getForwardedPort(), 5*time.Second, stdout, stderr)
		Eventually(stdout).Should(gbytes.Say("192.168.38.29"))
		sshClient.RunSSHCommand("echo $NO_PROXY", getForwardedPort(), 5*time.Second, stdout, stderr)
		Eventually(stdout).Should(gbytes.Say("192.168.98.98"))

		response, err = getResponseFromFakeServer(interfaceName)
		Expect(err).NotTo(HaveOccurred())
		Expect(response).To(Equal("PCF Dev Test VM"))
	})

	It("should respond to pcfdev alias", func() {
		pcfdevCommand := exec.Command("cf", "pcfdev", "help")
		session, err := gexec.Start(pcfdevCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session).Should(gexec.Exit(0))
		Expect(session).To(gbytes.Say("cf dev SUBCOMMAND"))
	})

	It("should download a VM without importing it", func() {
		pcfdevCommand := exec.Command("cf", "dev", "download")
		session, err := gexec.Start(pcfdevCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session, "1h").Should(gexec.Exit(0))

		_, err = os.Stat(filepath.Join(os.Getenv("PCFDEV_HOME"), "ova", "pcfdev-test.ova"))
		Expect(err).NotTo(HaveOccurred())

		listVmsCommand := exec.Command(vBoxManagePath, "list", "vms")
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
	vmStatus, err := exec.Command(vBoxManagePath, "showvminfo", vmName, "--machinereadable").Output()
	Expect(err).NotTo(HaveOccurred())
	return strings.Contains(string(vmStatus), `VMState="running"`)
}

func vmMemory() string {
	output, err := exec.Command(vBoxManagePath, "showvminfo", vmName, "--machinereadable").Output()
	Expect(err).NotTo(HaveOccurred())

	regex := regexp.MustCompile(`memory=(\d+)`)
	return regex.FindStringSubmatch(string(output))[1]
}

func getForwardedPort() string {
	output, err := exec.Command(vBoxManagePath, "showvminfo", vmName, "--machinereadable").Output()
	Expect(err).NotTo(HaveOccurred())

	regex := regexp.MustCompile(`Forwarding\(\d+\)="ssh,tcp,127.0.0.1,(.*),,22"`)
	return regex.FindStringSubmatch(string(output))[1]
}

func getResponseFromFakeServer(vboxnetName string) (response string, err error) {
	output, err := exec.Command(vBoxManagePath, "list", "hostonlyifs").Output()
	Expect(err).NotTo(HaveOccurred())

	nameRegex := regexp.MustCompile(`(?m:^Name:\s+(.*))`)
	nameMatches := nameRegex.FindAllStringSubmatch(string(output), -1)

	ipRegex := regexp.MustCompile(`(?m:^IPAddress:\s+(.*))`)
	ipMatches := ipRegex.FindAllStringSubmatch(string(output), -1)

	var hostname string
	for i := 0; i < len(nameMatches); i++ {
		if strings.TrimSpace(nameMatches[i][1]) == vboxnetName {
			ip := strings.TrimSpace(ipMatches[i][1])
			ipRegex := regexp.MustCompile(`192.168.\d(\d).1`)
			digit := ipRegex.FindStringSubmatch(string(ip))[1]

			hostname = fmt.Sprintf("http://api.local%s.pcfdev.io", digit)
			break
		}
	}

	timeoutChan := time.After(2 * time.Minute)
	var httpResponse *http.Response
	var responseBody []byte

	for {
		select {
		case <-timeoutChan:
			return "", fmt.Errorf("connection timed out: %s", err)
		default:
			httpResponse, err = http.Get(hostname)
			if err != nil {
				continue
			}
			defer httpResponse.Body.Close()

			responseBody, err = ioutil.ReadAll(httpResponse.Body)
			return string(responseBody), err
		}
	}
}

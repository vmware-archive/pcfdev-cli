package integration_test

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
	vmName = "pcfdev-test2"
)

var (
	pluginPath      string
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
	Expect(os.Getenv("PIVNET_TOKEN")).NotTo(BeEmpty(), "PIVNET_TOKEN must be set")

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
	os.Setenv("PCFDEV_HOME", filepath.Join(tempHome, "pcfdev"))

	pluginPath, err = gexec.Build(filepath.Join("github.com", "pivotal-cf", "pcfdev-cli"), "-ldflags",
		"-X main.vmName="+vmName+
			" -X main.releaseId=1622"+
			" -X main.productFileId=5113"+
			" -X main.md5=3af8ffc82ff6229cd45c27cd0445161f")
	Expect(err).NotTo(HaveOccurred())

	session, err := gexec.Start(exec.Command(pluginPath), GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session, "1m").Should(gexec.Exit(0))
	Expect(session).To(gbytes.Say("Plugin successfully installed, run: cf dev help"))

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

var _ = Describe("PCF Dev", func() {
	AfterEach(func() {
		for _, vm := range []string{vmName, "pcfdev-custom"} {
			output, err := exec.Command(vBoxManagePath, "showvminfo", vm, "--machinereadable").Output()
			if err != nil {
				continue
			}

			regex := regexp.MustCompile(`hostonlyadapter2="(.*)"`)
			matches := regex.FindStringSubmatch(string(output))

			exec.Command(vBoxManagePath, "controlvm", vm, "poweroff").Run()
			exec.Command(vBoxManagePath, "unregistervm", vm, "--delete").Run()

			if len(matches) > 1 {
				exec.Command(vBoxManagePath, "hostonlyif", "remove", matches[1]).Run()
			}
		}
	})

	Context("when run directly", func() {
		It("should output a helpful usage message when run with help flags", func() {
			pluginCommand := exec.Command(pluginPath, "--help")
			session, err := gexec.Start(pluginCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, "5s").Should(gexec.Exit(0))
			Expect(session).To(gbytes.Say("After installing, run: cf dev help"))
		})

		It("should upgrade the plugin if it is already installed", func() {
			pluginCommand := exec.Command(pluginPath)
			session, err := gexec.Start(pluginCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, "1m").Should(gexec.Exit(0))
			Expect(session).To(gbytes.Say("Plugin successfully upgraded, run: cf dev help"))
		})

		It("should output an error message when the cf CLI in unavailable", func() {
			pluginCommand := exec.Command(pluginPath)
			pluginCommand.Env = []string{}
			session, err := gexec.Start(pluginCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, "1m").Should(gexec.Exit(1))
			Expect(session).To(gbytes.Say("Failed to determine cf CLI version"))
		})
	})

	It("should start, stop, and destroy a virtualbox instance", func() {
		pcfdevCommand := exec.Command("cf", "dev", "start", "-c", "1")
		session, err := gexec.Start(pcfdevCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session, "10m").Should(gexec.Exit(0))
		Expect(session).To(gbytes.Say("Waiting for services to start..."))
		Expect(session).To(gbytes.Say("Services started"))
		pcfdevCommand = exec.Command("cf", "dev", "status")
		session, err = gexec.Start(pcfdevCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session, "2m").Should(gexec.Exit(0))
		Eventually(session).Should(gbytes.Say("Running"))
		Expect(filepath.Join(tempHome, "pcfdev", "vms", vmName, vmName+"-disk1.vmdk")).To(BeAnExistingFile())

		By("re-running 'cf dev start' with no effect")
		restartCommand := exec.Command("cf", "dev", "start")
		session, err = gexec.Start(restartCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session, "2m").Should(gexec.Exit(0))
		Expect(session).To(gbytes.Say("PCF Dev is running"))
		pcfdevCommand = exec.Command("cf", "dev", "status")
		session, err = gexec.Start(pcfdevCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session, "2m").Should(gexec.Exit(0))
		Eventually(session).Should(gbytes.Say("Running"))

		By("running 'cf dev suspend' should suspend the vm")
		suspendCommand := exec.Command("cf", "dev", "suspend")
		session, err = gexec.Start(suspendCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session, "2m").Should(gexec.Exit(0))
		Expect(session).To(gbytes.Say("Suspending VM..."))
		Expect(session).To(gbytes.Say("PCF Dev is now suspended"))
		pcfdevCommand = exec.Command("cf", "dev", "status")
		session, err = gexec.Start(pcfdevCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session, "2m").Should(gexec.Exit(0))
		Eventually(session).Should(gbytes.Say("Suspended"))

		By("running 'cf dev resume' should resume the vm")
		resumeCommand := exec.Command("cf", "dev", "resume")
		session, err = gexec.Start(resumeCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session, "10m").Should(gexec.Exit(0))
		Expect(session).To(gbytes.Say("Resuming VM..."))
		pcfdevCommand = exec.Command("cf", "dev", "status")
		session, err = gexec.Start(pcfdevCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session, "2m").Should(gexec.Exit(0))
		Eventually(session).Should(gbytes.Say("Running"))

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

		pcfdevCommand = exec.Command("cf", "dev", "status")
		session, err = gexec.Start(pcfdevCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session, "2m").Should(gexec.Exit(0))
		Eventually(session).Should(gbytes.Say("Stopped"))

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

		By("setting proxy variables")
		os.Setenv("HTTP_PROXY", "192.168.93.23")
		os.Setenv("HTTPS_PROXY", "192.168.38.29")
		os.Setenv("NO_PROXY", "192.168.98.98")

		By("starting after running destroy")
		pcfdevCommand = exec.Command("cf", "dev", "start", "-m", "3456", "-c", "1", "-o", filepath.Join(os.Getenv("PCFDEV_HOME"), "ova", "pcfdev-test2.ova"))
		session, err = gexec.Start(pcfdevCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session, "10m").Should(gexec.Exit(0))
		Expect(session).To(gbytes.Say("Waiting for services to start..."))
		Expect(session).To(gbytes.Say("Services started"))
		pcfdevCommand = exec.Command("cf", "dev", "status")
		session, err = gexec.Start(pcfdevCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session, "2m").Should(gexec.Exit(0))
		Eventually(session).Should(gbytes.Say("Running"))
		Expect(vmMemory("pcfdev-custom")).To(Equal("3456"))
		Expect(vmCores("pcfdev-custom")).To(Equal("1"))

		stdout := gbytes.NewBuffer()
		stderr := gbytes.NewBuffer()
		sshClient := &ssh.SSH{}
		sshClient.RunSSHCommand("echo $HTTP_PROXY", getForwardedPort("pcfdev-custom"), 5*time.Second, stdout, stderr)
		Eventually(stdout).Should(gbytes.Say("192.168.93.23"))
		sshClient.RunSSHCommand("echo $HTTPS_PROXY", getForwardedPort("pcfdev-custom"), 5*time.Second, stdout, stderr)
		Eventually(stdout).Should(gbytes.Say("192.168.38.29"))
		sshClient.RunSSHCommand("echo $NO_PROXY", getForwardedPort("pcfdev-custom"), 5*time.Second, stdout, stderr)
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

		Expect(filepath.Join(os.Getenv("PCFDEV_HOME"), "ova", "pcfdev-test2.ova")).To(BeAnExistingFile())

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

	It("should provision or not provision the VM depending on the command", func() {
		noProvisionCommand := exec.Command("cf", "dev", "start", "-n")
		session, err := gexec.Start(noProvisionCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session, "1h").Should(gexec.Exit(0))
		Expect(session).To(gbytes.Say("VM will not be provisioned .*"))

		provisionCommand := exec.Command("cf", "dev", "provision")
		session, err = gexec.Start(provisionCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session, "1h").Should(gexec.Exit(0))
		Expect(session).To(gbytes.Say("Provisioning VM..."))
		Expect(session).To(gbytes.Say("Waiting for services to start..."))
		Expect(session).To(gbytes.Say("Services started"))
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

func vmMemory(name string) string {
	output, err := exec.Command(vBoxManagePath, "showvminfo", name, "--machinereadable").Output()
	Expect(err).NotTo(HaveOccurred())

	regex := regexp.MustCompile(`memory=(\d+)`)
	return regex.FindStringSubmatch(string(output))[1]
}

func vmCores(name string) string {
	output, err := exec.Command(vBoxManagePath, "showvminfo", name, "--machinereadable").Output()
	Expect(err).NotTo(HaveOccurred())

	regex := regexp.MustCompile(`cpus=(\d+)`)
	return regex.FindStringSubmatch(string(output))[1]
}

func getForwardedPort(name string) string {
	output, err := exec.Command(vBoxManagePath, "showvminfo", name, "--machinereadable").Output()
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

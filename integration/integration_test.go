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

	"runtime"

	"github.com/kr/pty"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf/pcfdev-cli/helpers"
	"github.com/pivotal-cf/pcfdev-cli/ssh"
)

const (
	vmName                = "pcfdev-test"
	releaseID             = "1622"
	testOvaProductFileID  = "5689"
	testOvaMd5            = "c48e9706abae6f36af9e2473f90b10bd"
	emptyOvaProductFileId = "8883"
	emptyOvaMd5           = "8cfb57f0b6f0305cf6797fe361ed738a"
)

var (
	pluginPath      string
	tempHome        string
	ovaPath         string
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

	var ovaDir string
	if path := os.Getenv("INTEGRATION_TEST_OVA_HOME"); path != "" {
		ovaDir = path
	} else {
		var err error
		ovaDir, err = ioutil.TempDir("", "ova")
		Expect(err).NotTo(HaveOccurred())
	}

	ovaPath = filepath.Join(ovaDir, "ova")
	if !fileExists(ovaPath) {
		Expect(exec.Command(
			"wget",
			"--no-verbose",
			"--header", "Authorization: Token "+os.Getenv("PIVNET_TOKEN"),
			"--post-data", "",
			fmt.Sprintf("https://network.pivotal.io/api/v2/products/pcfdev/releases/%s/product_files/%s/download", releaseID, testOvaProductFileID),
			"-O", ovaPath).Run(),
		).To(Succeed())
	}
})

var _ = BeforeEach(func() {
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

	pluginPath = compileCLI(releaseID, testOvaProductFileID, testOvaMd5)

	vBoxManagePath, err = helpers.VBoxManagePath()
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterEach(func() {
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
			exec.Command(vBoxManagePath, "controlvm", vm, "poweroff").Run()
			exec.Command(vBoxManagePath, "unregistervm", vm, "--delete").Run()
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
			Expect(session).To(gbytes.Say("Plugin successfully upgraded. Current version: 0.0.0. For more info run: cf dev help"))
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

	Context("when downloading from Pivotal Network", func() {
		It("should download from Pivotal Network when cf dev start is specified", func() {
			pcfdevCommand := exec.Command("cf", "dev", "start")
			session, err := gexec.Start(pcfdevCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, "10m").Should(gexec.Exit(0))
			Expect(session).To(gbytes.Say("Services started"))
			Expect(filepath.Join(tempHome, "pcfdev", "vms", vmName, vmName+"-disk1.vmdk")).To(BeAnExistingFile())
		})

		Context("Using a small file", func() {
			BeforeEach(func() {
				pluginPath = compileCLI(releaseID, emptyOvaProductFileId, emptyOvaMd5)
			})

			It("should successfully download", func() {
				By("running download")
				pcfdevCommand := exec.Command("cf", "dev", "download")
				session, err := gexec.Start(pcfdevCommand, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session, "1h").Should(gexec.Exit(0))

				Expect(filepath.Join(os.Getenv("PCFDEV_HOME"), "ova", "pcfdev-test.ova")).To(BeAnExistingFile())

				listVmsCommand := exec.Command(vBoxManagePath, "list", "vms")
				session, err = gexec.Start(listVmsCommand, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))
				Expect(session).NotTo(gbytes.Say(vmName))

				By("rerunning download with no effect")
				pcfdevCommand = exec.Command("cf", "dev", "download")
				session, err = gexec.Start(pcfdevCommand, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session, "3m").Should(gexec.Exit(0))
			})
		})
	})

	It("should allow SSH access and forward interrupts", func() {
		if runtime.GOOS == "windows" {
			Skip("pty is not available on windows")
		}

		const interrupt = "\x03"

		pcfdevCommand := exec.Command("cf", "dev", "start", "-c", "1", "-o", ovaPath)
		session, err := gexec.Start(pcfdevCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session, "10m").Should(gexec.Exit(0))
		Expect(session).To(gbytes.Say("Services started"))

		sshCommand := exec.Command("cf", "dev", "ssh")
		sshPty, err := pty.Start(sshCommand)
		Expect(err).NotTo(HaveOccurred())

		time.Sleep(time.Second)
		fmt.Fprintln(sshPty, "sleep 99999999")
		fmt.Fprintln(sshPty, interrupt)
		fmt.Fprintln(sshPty, "echo 'Program was stopped by interrupt successfully' && >&2 echo 'Printing to stderr also works' && exit")
		time.Sleep(time.Second)
		ptyOutput, err := ioutil.ReadAll(sshPty)
		Expect(ptyOutput).To(ContainSubstring("Program was stopped by interrupt successfully"))
		Expect(ptyOutput).To(ContainSubstring("Printing to stderr also works"))
		Expect(ptyOutput).To(ContainSubstring("logout"))
		Expect(sshCommand.Wait()).To(Succeed())
	})

	It("should start, stop, start again and destroy a virtualbox instance", func() {
		pcfdevCommand := exec.Command("cf", "dev", "start", "-c", "1", "-o", ovaPath)
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
		Expect(filepath.Join(tempHome, "pcfdev", "vms", "pcfdev-custom", "pcfdev-custom-disk1.vmdk")).To(BeAnExistingFile())

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

		output, err := exec.Command(vBoxManagePath, "showvminfo", "pcfdev-custom", "--machinereadable").Output()
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

		pcfdevCommand = exec.Command("cf", "dev", "start")
		session, err = gexec.Start(pcfdevCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session, "5m").Should(gexec.Exit(0))
		Expect(session).To(gbytes.Say("Waiting for services to start..."))
		Expect(session).To(gbytes.Say("Services started"))

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
		pcfdevCommand = exec.Command("cf", "dev", "start",
			"-m", "3456",
			"-c", "1",
			"-o", ovaPath,
			"-i", "192.168.200.138",
			"-d", "192.168.200.138.xip.io",
		)
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

		securePrivateKey, err := ioutil.ReadFile(filepath.Join(os.Getenv("PCFDEV_HOME"), "vms", "key.pem"))
		Expect(err).NotTo(HaveOccurred())

		stdout := gbytes.NewBuffer()
		stderr := gbytes.NewBuffer()
		sshClient := &ssh.SSH{}
		sshPort := getForwardedPort("pcfdev-custom")
		sshClient.RunSSHCommand("echo $HTTP_PROXY", []ssh.SSHAddress{{IP: "127.0.0.1", Port: sshPort}}, securePrivateKey, time.Minute, stdout, stderr)
		Eventually(stdout, "30s").Should(gbytes.Say("192.168.93.23"))
		sshClient.RunSSHCommand("echo $HTTPS_PROXY", []ssh.SSHAddress{{IP: "127.0.0.1", Port: sshPort}}, securePrivateKey, time.Minute, stdout, stderr)
		Eventually(stdout, "10s").Should(gbytes.Say("192.168.38.29"))
		sshClient.RunSSHCommand("echo $NO_PROXY", []ssh.SSHAddress{{IP: "127.0.0.1", Port: sshPort}}, securePrivateKey, time.Minute, stdout, stderr)
		Eventually(stdout, "10s").Should(gbytes.Say("192.168.98.98"))

		response, err = getResponseFromFakeServerWithHostname("http://192.168.200.138.xip.io")
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

	It("should respond to 'version' and '--version' commands", func() {
		output, err := exec.Command("cf", "dev", "version").Output()
		Expect(err).NotTo(HaveOccurred())
		Expect(string(output)).To(Equal("PCF Dev version 0.0.0 (CLI: some-cli-sha, OVA: some-ova-version)\n"))

		output, err = exec.Command("cf", "dev", "--version").Output()
		Expect(err).NotTo(HaveOccurred())
		Expect(string(output)).To(Equal("PCF Dev version 0.0.0 (CLI: some-cli-sha, OVA: some-ova-version)\n"))
	})

	Context("when ova is on pivnet or in a temp dir", func() {
		var (
			tempOVALocation string
			wrongOVA        *os.File
		)

		BeforeEach(func() {
			var err error
			tempOVALocation, err = ioutil.TempDir("", "ova-to-import")
			wrongOVA, err = ioutil.TempFile(tempOVALocation, "wrong-ova.ova")
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			os.RemoveAll(tempOVALocation)
		})

		It("should download or import an ova", func() {
			By("removing")
			os.Rename(filepath.Join(os.Getenv("PCFDEV_HOME"), "ova", "pcfdev-test.ova"), filepath.Join(tempOVALocation, "pcfdev-test.ova"))
			Expect(filepath.Join(os.Getenv("PCFDEV_HOME"), "ova", "pcfdev-test.ova")).NotTo(BeAnExistingFile())

			By("running import")
			importCommand := exec.Command("cf", "dev", "import", ovaPath)
			session, err := gexec.Start(importCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, "5m").Should(gexec.Exit(0))
			Expect(filepath.Join(os.Getenv("PCFDEV_HOME"), "ova", "pcfdev-test.ova")).To(BeAnExistingFile())
			Eventually(session).Should(gbytes.Say("OVA version some-ova-version imported successfully."))

			By("rerunning import")
			importCommand = exec.Command("cf", "dev", "import", ovaPath)
			session, err = gexec.Start(importCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, "3m").Should(gexec.Exit(0))
			Eventually(session).Should(gbytes.Say("PCF Dev OVA is already installed."))

			By("running import with incorrect ova")
			importCommand = exec.Command("cf", "dev", "import", wrongOVA.Name())
			session, err = gexec.Start(importCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, "3m").Should(gexec.Exit(1))
			Eventually(session).Should(gbytes.Say("Error: specified OVA version does not match the expected OVA version \\(some-ova-version\\) for this version of the cf CLI plugin."))

			By("running start without provision")
			noProvisionCommand := exec.Command("cf", "dev", "start", "-n")
			session, err = gexec.Start(noProvisionCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, "1h").Should(gexec.Exit(0))
			Expect(session).To(gbytes.Say("VM will not be provisioned .*"))

			By("running provision")
			provisionCommand := exec.Command("cf", "dev", "start", "-p")
			session, err = gexec.Start(provisionCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, "1h").Should(gexec.Exit(0))
			Expect(session).To(gbytes.Say("Provisioning VM..."))
			Expect(session).To(gbytes.Say("Waiting for services to start..."))
			Expect(session).To(gbytes.Say("Services started"))

			By("running 'cf dev debug'")
			pcfdevCommand := exec.Command("cf", "dev", "debug")
			session, err = gexec.Start(pcfdevCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, "2m").Should(gexec.Exit(0))
			Expect(session).To(gbytes.Say("Debug logs written to pcfdev-debug.tgz.*"))
			Expect("pcfdev-debug.tgz").To(BeAnExistingFile())
			Expect(os.RemoveAll("pcfdev-debug.tgz")).To(Succeed())

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

func getForwardedPort(vmName string) string {
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

	return getResponseFromFakeServerWithHostname(hostname)
}

func getResponseFromFakeServerWithHostname(hostname string) (response string, err error) {
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

func fileExists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		return false
	}
	return true
}

func compileCLI(releaseID, productFileID, md5 string) string {
	insecurePrivateKeyBytes, err := ioutil.ReadFile(filepath.Join("..", "assets", "insecure.key"))
	Expect(err).NotTo(HaveOccurred())

	pluginPath, err := gexec.Build(filepath.Join("github.com", "pivotal-cf", "pcfdev-cli"), "-ldflags",
		"-X main.vmName="+vmName+
			" -X main.buildVersion=0.0.0"+
			" -X main.buildSHA=some-cli-sha"+
			" -X main.ovaBuildVersion=some-ova-version"+
			" -X main.releaseId="+releaseID+
			" -X main.productFileId="+productFileID+
			" -X main.md5="+md5+
			fmt.Sprintf(` -X "main.insecurePrivateKey=%s"`, string(insecurePrivateKeyBytes)))
	Expect(err).NotTo(HaveOccurred())

	session, err := gexec.Start(exec.Command(pluginPath), GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session, "1m").Should(gexec.Exit(0))
	Expect(session).To(gbytes.Say("Plugin successfully .* Current version: 0.0.0. For more info run: cf dev help"))

	return pluginPath
}

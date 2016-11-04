package integration_test

import (
	"io/ioutil"
	"path/filepath"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/pivotal-cf/pcfdev-cli/integration"
	"os"
	"os/exec"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Services", func() {

	var (
		oldCFHome string
		oldCFPluginHome string
		oldPCFDevHome string
		tempHome string
	)
	BeforeEach(func() {
		oldCFHome = os.Getenv("CF_HOME")
		oldCFPluginHome = os.Getenv("CF_PLUGIN_HOME")
		oldPCFDevHome = os.Getenv("PCFDEV_HOME")

		var err error
		tempHome, err = ioutil.TempDir("", "pcfdev")
		Expect(err).NotTo(HaveOccurred())

		os.Setenv("CF_HOME", tempHome)
		os.Setenv("CF_PLUGIN_HOME", filepath.Join(tempHome, "plugins"))
		os.Setenv("PCFDEV_HOME", filepath.Join(tempHome, "pcfdev"))

		SetupOva(ReleaseID, TestOvaProductFileID)
		CompileCLI(ReleaseID, TestOvaProductFileID, TestOvaMd5, VmName)
	})

	AfterEach(func() {
		Expect(os.RemoveAll(tempHome)).To(Succeed())
		os.Setenv("CF_HOME", oldCFHome)
		os.Setenv("CF_PLUGIN_HOME", oldCFPluginHome)
		os.Setenv("PCFDEV_HOME", oldPCFDevHome)
	})

	XIt("should list the availble services across all orgs", func(){
		pcfdevCommand := exec.Command("cf", "dev", "services")
		session, err := gexec.Start(pcfdevCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session).Should(gexec.Exit(0))
		Expect(session).To(gbytes.Say(`Running Service Instances:
	--
	pcfdev-org/pcfdev-space/my-mysql-2, type: p-mysql
	Dashboard URL: https://rabbitmq-management.local.pcfdev.io/#/login/apcfdev/utgs9v182ahdk4aptakhnu0csl
	Service URL: amqp://apcfdev:utgs9v182ahdk4aptakhnu0csl@rabbitmq.local.pcfdev.io/4552769e-f3a9-4856-8f14-3fd98759cb0b
	--
	User-Provided Services:
	--
	No user-provided services present.
	--`))
	})

	FContext("When no services instances or user provided service instances exist", func() {
		It("displays a nice message to the user", func() {
			pcfdevCommand := exec.Command("cf", "dev", "services")
			session, err := gexec.Start(pcfdevCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
			Expect(session).To(gbytes.Say(`Running Service Instances:
--
No service instances present.
--
User-Provided Services:
--
No user-provided services present.
--
`))
		})
	})

})




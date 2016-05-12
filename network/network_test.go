package network_test

import (
	"os/exec"
	"regexp"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf/pcfdev-cli/network"
)

var _ = Describe("network", func() {
	Describe("Interfaces", func() {
		var expectedName string
		var expectedIP string

		BeforeEach(func() {
			expectedIP = "192.168.56.56"
			output, err := exec.Command("VBoxManage", "hostonlyif", "create").Output()
			Expect(err).NotTo(HaveOccurred())
			regex := regexp.MustCompile(`Interface '(.*)' was successfully created`)
			matches := regex.FindStringSubmatch(string(output))
			expectedName = matches[1]
			assignIP := exec.Command("VBoxManage", "hostonlyif", "ipconfig", expectedName, "--ip", expectedIP, "--netmask", "255.255.255.0")
			session, err := gexec.Start(assignIP, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
		})

		AfterEach(func() {
			assignIP := exec.Command("VBoxManage", "hostonlyif", "remove", expectedName)
			session, err := gexec.Start(assignIP, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
		})

		It("should return the network interfaces", func() {
			network := network.Network{}
			interfaces, err := network.Interfaces()
			Expect(err).NotTo(HaveOccurred())
			lastCreatedInterface := interfaces[len(interfaces)-1]
			Expect(lastCreatedInterface.IP).To(Equal(expectedIP))
		})
	})
})

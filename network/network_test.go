package network_test

import (
	"os/exec"
	"regexp"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf/pcfdev-cli/helpers"
	"github.com/pivotal-cf/pcfdev-cli/network"
)

var _ = Describe("network", func() {
	Describe("Interfaces", func() {
		var (
			expectedName            string
			expectedIP              string
			expectedHardwareAddress string
		)

		BeforeEach(func() {
			expectedIP = "192.168.56.56"
			vBoxManagePath, err := helpers.VBoxManagePath()
			Expect(err).NotTo(HaveOccurred())
			output, err := exec.Command(vBoxManagePath, "hostonlyif", "create").Output()
			Expect(err).NotTo(HaveOccurred())

			regex := regexp.MustCompile(`Interface '(.*)' was successfully created`)
			matches := regex.FindStringSubmatch(string(output))
			expectedName = matches[1]

			assignIP := exec.Command(vBoxManagePath, "hostonlyif", "ipconfig", expectedName, "--ip", expectedIP, "--netmask", "255.255.255.0")
			session, err := gexec.Start(assignIP, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, "10s").Should(gexec.Exit(0))

			hardwareAddressOutput, err := exec.Command(vBoxManagePath, "list", "hostonlyifs").Output()
			Expect(err).NotTo(HaveOccurred())

			nameRegex := regexp.MustCompile(`(?m:^Name:\s+(.*))`)
			nameMatches := nameRegex.FindAllStringSubmatch(string(hardwareAddressOutput), -1)
			hardwareAddressRegex := regexp.MustCompile(`(?m:^HardwareAddress:\s+(.*))`)
			hardwareAddressMatches := hardwareAddressRegex.FindAllStringSubmatch(string(hardwareAddressOutput), -1)

			for i := 0; i < len(nameMatches); i++ {
				if strings.TrimSpace(nameMatches[i][1]) == expectedName {
					expectedHardwareAddress = strings.TrimSpace(hardwareAddressMatches[i][1])
				}
			}
		})

		AfterEach(func() {
			vBoxManagePath, err := helpers.VBoxManagePath()
			Expect(err).NotTo(HaveOccurred())
			assignIP := exec.Command(vBoxManagePath, "hostonlyif", "remove", expectedName)
			session, err := gexec.Start(assignIP, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, "10s").Should(gexec.Exit(0))
		})

		It("should return the network interfaces", func() {
			net := network.Network{}
			interfaces, err := net.Interfaces()
			Expect(err).NotTo(HaveOccurred())
			expectedInterface := &network.Interface{IP: expectedIP, HardwareAddress: expectedHardwareAddress, Exists: true}
			Expect(interfaces).To(ContainElement(expectedInterface))
		})
	})
})

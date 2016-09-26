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

var _ = Describe("Network", func() {
	var (
		net                     *network.Network
		expectedName            string
		expectedIP              string
		expectedHardwareAddress string
	)

	BeforeEach(func() {
		net = &network.Network{}
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

	Describe("#Interfaces", func() {
		It("should return the network interfaces", func() {
			interfaces, err := net.Interfaces()
			Expect(err).NotTo(HaveOccurred())
			expectedInterface := &network.Interface{IP: expectedIP, HardwareAddress: expectedHardwareAddress, Exists: true}
			Expect(interfaces).To(ContainElement(expectedInterface))
		})
	})

	Describe("#HasIPCollision", func() {
		It("should return true when ip collides", func() {
			Expect(net.HasIPCollision(expectedIP)).To(BeTrue())
		})

		It("should return false when ip does not collide", func() {
			Expect(net.HasIPCollision("some-non-colliding-ip")).To(BeFalse())
		})
	})

	Describe(".IsIPV4", func() {
		It("should return true when ip is valid", func() {
			Expect(network.IsIPV4("192.168.11.11")).To(BeTrue())
		})

		It("should return false when ip is not valid", func() {
			Expect(network.IsIPV4("some-bad-ip")).To(BeFalse())
		})
	})
})

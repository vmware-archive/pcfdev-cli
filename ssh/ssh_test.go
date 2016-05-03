package ssh_test

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/nu7hatch/gouuid"
	. "github.com/pivotal-cf/pcfdev-cli/ssh"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ssh", func() {
	Describe("GenerateAddress", func() {
		It("Should return a host and free port", func() {
			ssh := &SSH{}
			host, port, err := ssh.GenerateAddress()
			Expect(err).NotTo(HaveOccurred())
			Expect(host).To(Equal("127.0.0.1"))
			Expect(port).To(MatchRegexp("^[\\d]+$"))
		})
	})

	Describe("RunSSHCommand", func() {
		var ssh *SSH

		Context("when SSH is available", func() {
			var (
				vmName string
				port   string
			)

			BeforeEach(func() {
				ssh = &SSH{}
				_, err := os.Stat("../assets/snappy.ova")
				if os.IsNotExist(err) {
					By("Downloading ova...")
					resp, err := http.Get("https://s3.amazonaws.com/pcfdev/ovas/snappy.ova")
					Expect(err).NotTo(HaveOccurred())
					ovaFile, err := os.Create("../assets/snappy.ova")
					Expect(err).NotTo(HaveOccurred())
					defer ovaFile.Close()
					_, err = io.Copy(ovaFile, resp.Body)
				}

				tmpDir := os.Getenv("TMPDIR")
				vmName = "Snappy-" + randomName()
				command := exec.Command("VBoxManage",
					"import",
					"../assets/snappy.ova",
					"--vsys", "0",
					"--vmname", vmName,
					"--unit", "6", "--disk", filepath.Join(tmpDir, vmName+"-disk1_4.vmdk"),
					"--unit", "7", "--disk", filepath.Join(tmpDir, vmName+"-disk2.vmdk"))
				Expect(command.Run()).To(Succeed())

				_, port, err = ssh.GenerateAddress()
				Expect(err).NotTo(HaveOccurred())

				Expect(exec.Command("VBoxManage", "modifyvm", vmName, "--natpf1", fmt.Sprintf("ssh,tcp,127.0.0.1,%s,,22", port)).Run()).To(Succeed())
				Expect(exec.Command("VBoxManage", "startvm", vmName, "--type", "headless").Run()).To(Succeed())
			})

			AfterEach(func() {
				Expect(exec.Command("VBoxManage", "controlvm", vmName, "poweroff").Run()).To(Succeed())
				Expect(exec.Command("VBoxManage", "unregistervm", vmName, "--delete").Run()).To(Succeed())
			})

			Context("when the command succeeds", func() {
				It("should return the output", func() {
					output, err := ssh.RunSSHCommand("echo -n some-output", port, time.Minute)
					Expect(err).NotTo(HaveOccurred())
					Expect(string(output)).To(Equal("some-output"))
				})
			})

			Context("when the command fails", func() {
				It("should return an error", func() {
					_, err := ssh.RunSSHCommand("false", port, time.Minute)
					Expect(err).To(MatchError(ContainSubstring("Process exited with: 1")))
				})
			})
		})

		Context("when SSH connection times out", func() {
			It("should return an error", func() {
				_, err := ssh.RunSSHCommand("echo -n some-output", "some-bad-port", time.Second)
				Expect(err).To(MatchError(ContainSubstring("ssh connection timed out:")))
			})
		})
	})
})

func randomName() string {
	guid, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}

	return guid.String()
}

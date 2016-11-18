package integration

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"

	cMD5 "crypto/md5"
	"io"
	"os"
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

const (
	VmName                = "pcfdev-test"
	ReleaseID             = "1622"
	EmptyOvaProductFileId = "8883"
	EmptyOvaMd5           = "8cfb57f0b6f0305cf6797fe361ed738a"
)

func FileExists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		return false
	}
	return true
}

func DownloadTestOva(releaseID, productFileID, destination string) {
	req, err := http.NewRequest("POST", fmt.Sprintf("https://network.pivotal.io/api/v2/products/pcfdev/releases/%s/product_files/%s/download", releaseID, productFileID), nil)
	Expect(err).NotTo(HaveOccurred())

	req.Header.Set("Authorization", "Token "+os.Getenv("PIVNET_TOKEN"))

	resp, err := http.DefaultClient.Do(req)
	Expect(err).NotTo(HaveOccurred())

	destinationWriter, err := os.Create(destination)
	Expect(err).NotTo(HaveOccurred())

	_, err = io.Copy(destinationWriter, resp.Body)
	Expect(err).NotTo(HaveOccurred())
}

func CompileCLI(releaseID, productFileID, md5 string, vmName string) string {
	insecurePrivateKeyBytes, err := ioutil.ReadFile(filepath.Join("..", "assets", "test-private-key.pem"))
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

func SetupOva(releaseID string, testOvaProductFileID string, md5 string) string {
	Expect(os.Getenv("PIVNET_TOKEN")).NotTo(BeEmpty(), "PIVNET_TOKEN must be set")

	var ovaDir string
	if path := os.Getenv("INTEGRATION_TEST_OVA_HOME"); path != "" {
		ovaDir = path
	} else {
		var err error
		ovaDir, err = ioutil.TempDir("", "ova")
		Expect(err).NotTo(HaveOccurred())
	}

	ovaPath := filepath.Join(ovaDir, "ova")
	if FileExists(ovaPath) {
		matches, err := MD5Matches(ovaPath, md5)
		Expect(err).NotTo(HaveOccurred())

		if !matches {
			os.Remove(ovaPath)
		}
	}

	if !FileExists(ovaPath) {
		DownloadTestOva(releaseID, testOvaProductFileID, ovaPath)
	}
	return ovaPath
}

func MD5Matches(path, md5 string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	hash := cMD5.New()

	if _, err = io.Copy(hash, file); err != nil {
		return false, err
	}

	return fmt.Sprintf("%x", hash.Sum([]byte{})) == md5, nil
}

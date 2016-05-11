package pivnet_test

import (
	"io"
	"io/ioutil"
	"os"

	"github.com/pivotal-cf/pcfdev-cli/pivnet"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Download Reader", func() {
	Describe("when using a passed in reader", func() {
		var (
			reader   *pivnet.DownloadReader
			contents []byte
			stdout   *gbytes.Buffer
		)

		BeforeEach(func() {
			stdout = gbytes.NewBuffer()

			contents = []byte("some-contents")
			Expect(
				ioutil.WriteFile("../assets/some-file", contents, 0644),
			).To(Succeed())

			file, err := os.Open("../assets/some-file")
			Expect(err).NotTo(HaveOccurred())

			reader = &pivnet.DownloadReader{
				ReadCloser:    file,
				Writer:        stdout,
				ContentLength: int64(len(contents)),
			}
		})

		AfterEach(func() {
			Expect(
				os.Remove("../assets/some-file"),
			).To(Succeed())

			stdout.Close()
		})

		It("should display progress of the read", func() {
			_, err := io.Copy(ioutil.Discard, reader)
			Expect(err).NotTo(HaveOccurred())

			Eventually(stdout).Should(gbytes.Say("\rProgress: |>                   | 0%"))
			Eventually(stdout).Should(gbytes.Say("\rProgress: |===================>| 100%"))
		})
	})
})

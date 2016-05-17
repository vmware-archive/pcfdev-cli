package pivnet_test

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pivotal-cf/pcfdev-cli/pivnet"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Download Reader", func() {
	Describe("#Read", func() {
		var (
			contents []byte
			stdout   *gbytes.Buffer
			file     io.ReadCloser
			tmpDir   string
		)

		BeforeEach(func() {
			var err error

			stdout = gbytes.NewBuffer()
			contents = []byte("some-contents")
			tmpDir, err = ioutil.TempDir("", "pcfdev-reader")
			Expect(err).NotTo(HaveOccurred())

			Expect(
				ioutil.WriteFile(filepath.Join(tmpDir, "some-file"), contents, 0644),
			).To(Succeed())

			file, err = os.Open(filepath.Join(tmpDir, "some-file"))
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			Expect(
				os.RemoveAll(tmpDir),
			).To(Succeed())

			stdout.Close()
		})

		It("should display progress of the read", func() {
			reader := &pivnet.DownloadReader{
				ReadCloser:     file,
				Writer:         stdout,
				ContentLength:  int64(len(contents)),
				ExistingLength: 0,
			}

			_, err := io.Copy(ioutil.Discard, reader)
			Expect(err).NotTo(HaveOccurred())

			Eventually(stdout).Should(gbytes.Say(`\rProgress: \|>                    \| 0%`))
			Eventually(stdout).Should(gbytes.Say(`\rProgress: \|====================>\| 100%`))
		})

		Context("when an evenly divisible percentage of the file has been downloaded", func() {
			It("should display progress before starting byte as +s", func() {
				reader := &pivnet.DownloadReader{
					ReadCloser:     file,
					Writer:         stdout,
					ContentLength:  int64(len(contents)),
					ExistingLength: 13,
				}
				_, err := io.Copy(ioutil.Discard, reader)
				Expect(err).NotTo(HaveOccurred())

				Eventually(stdout).Should(gbytes.Say(`\r\QProgress: |++++++++++>          | 50%\E`))
				Eventually(stdout).Should(gbytes.Say(`\r\QProgress: |++++++++++==========>| 100%\E`))
			})
		})

		Context("when an unevenly divisible percentage of the file has been downloaded", func() {
			It("should display progress before starting byte as +s", func() {
				reader := &pivnet.DownloadReader{
					ReadCloser:     file,
					Writer:         stdout,
					ContentLength:  int64(len(contents)),
					ExistingLength: 20,
				}
				_, err := io.Copy(ioutil.Discard, reader)
				Expect(err).NotTo(HaveOccurred())

				Eventually(stdout).Should(gbytes.Say(`\r\QProgress: |+++++++++++++>       | 61%\E`))
				Eventually(stdout).Should(gbytes.Say(`\r\QProgress: |+++++++++++++=======>| 100%\E`))
			})
		})
	})
})

package fs_test

import (
	"io/ioutil"
	"os"
	"strings"

	"github.com/pivotal-cf/pcfdev-cli/fs"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Filesystem", func() {
	fs := &fs.FS{}
	Describe("#Exists", func() {
		Context("File exists", func() {
			BeforeEach(func() {
				_, err := os.Create("../assets/some-file")
				Expect(err).NotTo(HaveOccurred())
			})
			AfterEach(func() {
				err := os.Remove("../assets/some-file")
				Expect(err).NotTo(HaveOccurred())
			})
			It("returns true", func() {
				exists, err := fs.Exists("../assets/some-file")
				Expect(err).NotTo(HaveOccurred())
				Expect(exists).To(BeTrue())
			})
		})
		Context("File does not exist", func() {
			It("returns false", func() {
				exists, err := fs.Exists("../assets/some-bad-file")
				Expect(err).NotTo(HaveOccurred())
				Expect(exists).To(BeFalse())
			})
		})
	})
	Describe("#Write", func() {
		Context("path is valid", func() {
			It("Creates file with path and writes contents", func() {
				readCloser := ioutil.NopCloser(strings.NewReader("some-contents"))
				err := fs.Write("../assets/some-other-file", readCloser)
				Expect(err).NotTo(HaveOccurred())
				data, err := ioutil.ReadFile("../assets/some-other-file")
				Expect(err).NotTo(HaveOccurred())

				Expect(string(data)).To(Equal("some-contents"))
			})
			AfterEach(func() {
				os.Remove("../assets/some-other-file")
			})
		})
		Context("path is invalid", func() {
			It("returns an error", func() {
				readCloser := ioutil.NopCloser(strings.NewReader("some-contents"))
				err := fs.Write("../some-bad-dir/some-other-file", readCloser)
				Expect(err.Error()).To(ContainSubstring("failed to create file:"))
			})
		})
	})
	Describe("#CreateDir", func() {
		Context("directory does not exist", func() {
			It("creates the directory", func() {
				err := fs.CreateDir("../assets/some-dir")
				Expect(err).NotTo(HaveOccurred())
				_, err = os.Stat("../assets/some-dir")
				Expect(err).NotTo(HaveOccurred())
			})
			AfterEach(func() {
				os.Remove("../assets/some-dir")
			})
		})
		Context("directory already exists", func() {
			BeforeEach(func() {
				err := os.Mkdir("../assets/some-dir", 0755)
				Expect(err).NotTo(HaveOccurred())
			})
			It("does nothing", func() {
				err := fs.CreateDir("../assets/some-dir")
				Expect(err).NotTo(HaveOccurred())
				_, err = os.Stat("../assets/some-dir")
				Expect(err).NotTo(HaveOccurred())
			})
			AfterEach(func() {
				os.Remove("../assets/some-dir")
			})
		})
	})
})

package fs_test

import (
	"io/ioutil"
	"os"
	"strings"

	pcfdevfs "github.com/pivotal-cf/pcfdev-cli/fs"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Filesystem", func() {
	var fs *pcfdevfs.FS

	BeforeEach(func() {
		fs = &pcfdevfs.FS{}
	})

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
			AfterEach(func() {
				os.Remove("../assets/some-other-file")
			})

			It("Creates file with path and writes contents", func() {
				readCloser := ioutil.NopCloser(strings.NewReader("some-contents"))
				err := fs.Write("../assets/some-other-file", readCloser)
				Expect(err).NotTo(HaveOccurred())
				data, err := ioutil.ReadFile("../assets/some-other-file")
				Expect(err).NotTo(HaveOccurred())

				Expect(string(data)).To(Equal("some-contents"))
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
			AfterEach(func() {
				os.Remove("../assets/some-dir")
			})

			It("creates the directory", func() {
				err := fs.CreateDir("../assets/some-dir")
				Expect(err).NotTo(HaveOccurred())
				_, err = os.Stat("../assets/some-dir")
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("directory already exists", func() {
			BeforeEach(func() {
				err := os.Mkdir("../assets/some-dir", 0755)
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				os.Remove("../assets/some-dir")
			})

			It("does nothing", func() {
				err := fs.CreateDir("../assets/some-dir")
				Expect(err).NotTo(HaveOccurred())
				_, err = os.Stat("../assets/some-dir")
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("#MD5", func() {
		Context("when the file exists", func() {
			BeforeEach(func() {
				err := ioutil.WriteFile("../assets/some-file", []byte("some-contents"), 0644)
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				err := os.Remove("../assets/some-file")
				Expect(err).NotTo(HaveOccurred())
			})
			It("should return the md5 of the given file", func() {
				md5, err := fs.MD5("../assets/some-file")
				Expect(err).NotTo(HaveOccurred())
				Expect(md5).To(Equal("0b9791ad102b5f5f06ef68cef2aae26e"))
			})
		})

		Context("when the file does not exist", func() {
			It("should return an error", func() {
				md5, err := fs.MD5("../assets/some-non-existent-file")
				Expect(err).To(MatchError(ContainSubstring("could not read ../assets/some-non-existent-file:")))
				Expect(md5).To(Equal(""))
			})
		})
	})

	Describe("#RemoveFile", func() {
		BeforeEach(func() {
			err := ioutil.WriteFile("../assets/some-file", []byte("some-contents"), 0644)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			os.Remove("../assets/some-file")
		})

		It("should remove the given file", func() {
			err := fs.RemoveFile("../assets/some-file")
			Expect(err).NotTo(HaveOccurred())

			_, err = os.Stat("../assets/some-file")
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
	})
})

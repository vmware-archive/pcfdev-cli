package fs_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
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
		Context("when the file exists", func() {
			BeforeEach(func() {
				_, err := os.Create("../assets/some-file")
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				os.Remove("../assets/some-file")
			})

			It("should return true", func() {
				Expect(fs.Exists("../assets/some-file")).To(BeTrue())
			})
		})

		Context("when the file does not exist", func() {
			It("should return false", func() {
				Expect(fs.Exists("../assets/some-bad-file")).To(BeFalse())
			})
		})
	})

	Describe("#Write", func() {
		Context("when path is valid", func() {
			AfterEach(func() {
				os.Remove("../assets/some-file")
			})

			It("should create a file with path and writes contents", func() {
				readCloser := ioutil.NopCloser(strings.NewReader("some-contents"))
				Expect(fs.Write("../assets/some-file", readCloser)).To(Succeed())
				data, err := ioutil.ReadFile("../assets/some-file")
				Expect(err).NotTo(HaveOccurred())
				Expect(string(data)).To(Equal("some-contents"))
			})
		})

		Context("when file exists already", func() {
			BeforeEach(func() {
				Expect(fs.Write("../assets/some-file", ioutil.NopCloser(strings.NewReader("some-")))).To(Succeed())
			})

			AfterEach(func() {
				os.Remove("../assets/some-file")
			})

			It("should append to file", func() {
				readCloser := ioutil.NopCloser(strings.NewReader("contents"))
				Expect(fs.Write("../assets/some-file", readCloser)).To(Succeed())
				data, err := ioutil.ReadFile("../assets/some-file")
				Expect(err).NotTo(HaveOccurred())

				Expect(string(data)).To(Equal("some-contents"))
			})
		})

		Context("when path is invalid", func() {
			It("should return an error", func() {
				readCloser := ioutil.NopCloser(strings.NewReader("some-contents"))
				err := fs.Write("../some-bad-dir/some-other-file", readCloser)
				Expect(err.Error()).To(ContainSubstring("failed to open file:"))
			})
		})
	})

	Describe("#CreateDir", func() {
		Context("when the directory does not exist", func() {
			AfterEach(func() {
				os.Remove("../assets/some-dir")
			})

			It("should create the directory", func() {
				Expect(fs.CreateDir("../assets/some-dir")).To(Succeed())
				_, err := os.Stat("../assets/some-dir")
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when the directory already exists", func() {
			BeforeEach(func() {
				Expect(os.Mkdir("../assets/some-dir", 0755)).To(Succeed())
			})

			AfterEach(func() {
				os.Remove("../assets/some-dir")
			})

			It("should do nothing", func() {
				Expect(fs.CreateDir("../assets/some-dir")).To(Succeed())
				_, err := os.Stat("../assets/some-dir")
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("#DeleteAllExcept", func() {
		Context("when the directory already exists", func() {
			var tmpDir string
			var err error

			BeforeEach(func() {
				tmpDir, err = ioutil.TempDir("", "pcfdev-fs")
				Expect(err).NotTo(HaveOccurred())

				Expect(ioutil.WriteFile(filepath.Join(tmpDir, "some-file-name"), []byte("some-contents"), 0644)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(tmpDir, "not-some-file-name"), []byte("some-contents"), 0644)).To(Succeed())
			})

			AfterEach(func() {
				os.RemoveAll(tmpDir)
			})

			It("should delete files not matching the filenames", func() {
				Expect(fs.DeleteAllExcept(tmpDir, []string{"some-file-name"})).To(Succeed())
				_, err := os.Stat(filepath.Join(tmpDir, "not-some-file-name"))
				Expect(os.IsNotExist(err)).To(BeTrue())
				_, err = os.Stat(filepath.Join(tmpDir, "some-file-name"))
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when the directory does not exist", func() {
			It("should return an error", func() {
				Expect(fs.DeleteAllExcept("some-bad-path", []string{})).To(MatchError(ContainSubstring("failed to list files:")))
			})
		})
	})

	Describe("#MD5", func() {
		Context("when the file exists", func() {
			BeforeEach(func() {
				Expect(ioutil.WriteFile("../assets/some-file", []byte("some-contents"), 0644)).To(Succeed())
			})

			AfterEach(func() {
				os.Remove("../assets/some-file")
			})
			It("should return the md5 of the given file", func() {
				Expect(fs.MD5("../assets/some-file")).To(Equal("0b9791ad102b5f5f06ef68cef2aae26e"))
			})
		})

		Context("when the file does not exist", func() {
			It("should return an error", func() {
				md5, err := fs.MD5("../assets/some-non-existent-file")
				Expect(err).To(MatchError(ContainSubstring("failed to open ../assets/some-non-existent-file:")))
				Expect(md5).To(Equal(""))
			})
		})
	})

	Describe("#Length", func() {
		Context("when the file exists", func() {
			BeforeEach(func() {
				Expect(ioutil.WriteFile("../assets/some-file", []byte("some-contents"), 0644)).To(Succeed())
			})

			AfterEach(func() {
				os.Remove("../assets/some-file")
			})
			It("should return the length of the given file in bytes", func() {
				Expect(fs.Length("../assets/some-file")).To(Equal(int64(13)))
			})
		})

		Context("when the file does not exist", func() {
			It("should return an error", func() {
				length, err := fs.Length("../assets/some-non-existent-file")
				Expect(err).To(MatchError(ContainSubstring("failed to read ../assets/some-non-existent-file:")))
				Expect(length).To(Equal(int64(0)))
			})
		})
	})

	Describe("#RemoveFile", func() {
		BeforeEach(func() {
			Expect(ioutil.WriteFile("../assets/some-file", []byte("some-contents"), 0644)).To(Succeed())
		})

		AfterEach(func() {
			os.Remove("../assets/some-file")
		})

		It("should remove the given file", func() {
			Expect(fs.RemoveFile("../assets/some-file")).To(Succeed())

			_, err := os.Stat("../assets/some-file")
			Expect(os.IsNotExist(err)).To(BeTrue())
		})

		Context("when removing a file fails", func() {
			It("should return an error", func() {
				Expect(fs.RemoveFile("../assets/some-bad-file")).To(MatchError(ContainSubstring("failed to remove file ../assets/some-bad-file:")))
			})
		})
	})

	Describe("#Move", func() {
		Context("when the source exists and destination does not exist", func() {
			BeforeEach(func() {
				Expect(ioutil.WriteFile("../assets/some-file", []byte("some-contents"), 0644)).To(Succeed())
			})

			AfterEach(func() {
				os.Remove("../assets/some-file")
				os.Remove("../assets/some-other-file")
			})

			It("should move the source to the destination", func() {
				fs.Move("../assets/some-file", "../assets/some-other-file")
				Expect(fs.Exists("../assets/some-file")).To(BeFalse())
				data, err := ioutil.ReadFile("../assets/some-other-file")
				Expect(err).NotTo(HaveOccurred())
				Expect(string(data)).To(Equal("some-contents"))
			})
		})

		Context("when the source exists and destination exists", func() {
			BeforeEach(func() {
				Expect(ioutil.WriteFile("../assets/some-file", []byte("some-contents"), 0644)).To(Succeed())
				Expect(ioutil.WriteFile("../assets/some-other-file", []byte("some-other-contents"), 0644)).To(Succeed())
			})

			AfterEach(func() {
				os.Remove("../assets/some-other-file")
			})

			It("should replace the destination file", func() {
				fs.Move("../assets/some-file", "../assets/some-other-file")
				Expect(fs.Exists("../assets/some-file")).To(BeFalse())
				data, err := ioutil.ReadFile("../assets/some-other-file")
				Expect(err).NotTo(HaveOccurred())
				Expect(string(data)).To(Equal("some-contents"))
			})
		})

		Context("when the source does not exist", func() {
			It("should return an error", func() {
				Expect(fs.Move("../assets/some-bad-file", "../assets/some-other-file")).To(MatchError(ContainSubstring("failed to move ../assets/some-bad-file to ../assets/some-other-file:")))
			})
		})
	})
})

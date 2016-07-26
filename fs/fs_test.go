package fs_test

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	pcfdevfs "github.com/pivotal-cf/pcfdev-cli/fs"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Filesystem", func() {
	var (
		fs     *pcfdevfs.FS
		tmpDir string
	)

	BeforeEach(func() {
		fs = &pcfdevfs.FS{}
		var err error
		tmpDir, err = ioutil.TempDir("", "pcfdev-fs")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	Describe("#Read", func() {
		Context("when the file exists", func() {
			It("should return the contents the file", func() {
				Expect(ioutil.WriteFile(filepath.Join(tmpDir, "some-file"), []byte("some-contents"), 0644)).To(Succeed())
				Expect(fs.Read(filepath.Join(tmpDir, "some-file"))).To(Equal([]byte("some-contents")))
			})
		})
	})

	Describe("#Exists", func() {
		Context("when the file exists", func() {
			BeforeEach(func() {
				_, err := os.Create(filepath.Join(tmpDir, "some-file"))
				Expect(err).NotTo(HaveOccurred())
			})

			It("should return true", func() {
				Expect(fs.Exists(filepath.Join(tmpDir, "some-file"))).To(BeTrue())
			})
		})

		Context("when the file does not exist", func() {
			It("should return false", func() {
				Expect(fs.Exists(filepath.Join(tmpDir, "some-bad-file"))).To(BeFalse())
			})
		})
	})

	Describe("#Write", func() {
		Context("when path is valid", func() {
			It("should create a file with path and writes contents", func() {
				readCloser := ioutil.NopCloser(strings.NewReader("some-contents"))
				Expect(fs.Write(filepath.Join(tmpDir, "some-file"), readCloser)).To(Succeed())
				data, err := ioutil.ReadFile(filepath.Join(tmpDir, "some-file"))
				Expect(err).NotTo(HaveOccurred())
				Expect(string(data)).To(Equal("some-contents"))
			})
		})

		Context("when file exists already", func() {
			BeforeEach(func() {
				Expect(fs.Write(filepath.Join(tmpDir, "some-file"), ioutil.NopCloser(strings.NewReader("some-")))).To(Succeed())
			})

			It("should append to file", func() {
				readCloser := ioutil.NopCloser(strings.NewReader("contents"))
				Expect(fs.Write(filepath.Join(tmpDir, "some-file"), readCloser)).To(Succeed())
				data, err := ioutil.ReadFile(filepath.Join(tmpDir, "some-file"))
				Expect(err).NotTo(HaveOccurred())

				Expect(string(data)).To(Equal("some-contents"))
			})
		})

		Context("when path is invalid", func() {
			It("should return an error", func() {
				readCloser := ioutil.NopCloser(strings.NewReader("some-contents"))
				err := fs.Write(filepath.Join("some-bad-dir", "some-other-file"), readCloser)
				Expect(err.Error()).To(ContainSubstring("failed to open file:"))
			})
		})
	})

	Describe("#CreateDir", func() {
		Context("when the directory does not exist", func() {
			It("should create the directory", func() {
				Expect(fs.CreateDir(filepath.Join(tmpDir, "some-dir"))).To(Succeed())
				_, err := os.Stat(filepath.Join(tmpDir, "some-dir"))
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when the directory already exists", func() {
			BeforeEach(func() {
				Expect(os.Mkdir(filepath.Join(tmpDir, "some-dir"), 0755)).To(Succeed())
			})

			It("should do nothing", func() {
				Expect(fs.CreateDir(filepath.Join(tmpDir, "some-dir"))).To(Succeed())
				_, err := os.Stat(filepath.Join(tmpDir, "some-dir"))
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("#DeleteAllExcept", func() {
		Context("when the directory already exists", func() {
			BeforeEach(func() {
				Expect(ioutil.WriteFile(filepath.Join(tmpDir, "some-file-name"), []byte("some-contents"), 0644)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(tmpDir, "not-some-file-name"), []byte("some-contents"), 0644)).To(Succeed())
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
				Expect(ioutil.WriteFile(filepath.Join(tmpDir, "some-file"), []byte("some-contents"), 0644)).To(Succeed())
			})

			It("should return the md5 of the given file", func() {
				Expect(fs.MD5(filepath.Join(tmpDir, "some-file"))).To(Equal("0b9791ad102b5f5f06ef68cef2aae26e"))
			})
		})

		Context("when the file does not exist", func() {
			It("should return an error", func() {
				md5, err := fs.MD5(filepath.Join(tmpDir, "some-non-existent-file"))
				Expect(err).To(MatchError(ContainSubstring(fmt.Sprintf("failed to open %s:", filepath.Join(tmpDir, "some-non-existent-file")))))
				Expect(md5).To(Equal(""))
			})
		})
	})

	Describe("#Length", func() {
		Context("when the file exists", func() {
			BeforeEach(func() {
				Expect(ioutil.WriteFile(filepath.Join(tmpDir, "some-file"), []byte("some-contents"), 0644)).To(Succeed())
			})

			It("should return the length of the given file in bytes", func() {
				Expect(fs.Length(filepath.Join(tmpDir, "some-file"))).To(Equal(int64(13)))
			})
		})

		Context("when the file does not exist", func() {
			It("should return an error", func() {
				length, err := fs.Length(filepath.Join(tmpDir, "some-non-existent-file"))
				Expect(err).To(MatchError(ContainSubstring(fmt.Sprintf("failed to read %s:", filepath.Join(tmpDir, "some-non-existent-file")))))
				Expect(length).To(Equal(int64(0)))
			})
		})
	})

	Describe("#Remove", func() {
		BeforeEach(func() {
			Expect(ioutil.WriteFile(filepath.Join(tmpDir, "some-file"), []byte("some-contents"), 0644)).To(Succeed())
		})

		It("should remove the given file", func() {
			Expect(fs.Remove(filepath.Join(tmpDir, "some-file"))).To(Succeed())

			_, err := os.Stat(filepath.Join(tmpDir, "some-file"))
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
	})

	Describe("#Move", func() {
		Context("when the source exists and destination does not exist", func() {
			BeforeEach(func() {
				Expect(ioutil.WriteFile(filepath.Join(tmpDir, "some-file"), []byte("some-contents"), 0644)).To(Succeed())
			})

			It("should move the source to the destination", func() {
				Expect(fs.Move(filepath.Join(tmpDir, "some-file"), filepath.Join(tmpDir, "some-other-file"))).To(Succeed())
				Expect(fs.Exists(filepath.Join(tmpDir, "some-file"))).To(BeFalse())
				Expect(ioutil.ReadFile(filepath.Join(tmpDir, "some-other-file"))).To(Equal([]byte("some-contents")))
			})
		})

		Context("when the source exists and destination exists", func() {
			BeforeEach(func() {
				Expect(ioutil.WriteFile(filepath.Join(tmpDir, "some-file"), []byte("some-contents"), 0644)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(tmpDir, "some-other-file"), []byte("some-other-contents"), 0644)).To(Succeed())
			})

			It("should replace the destination file", func() {
				Expect(fs.Move(filepath.Join(tmpDir, "some-file"), filepath.Join(tmpDir, "some-other-file"))).To(Succeed())
				Expect(fs.Exists(filepath.Join(tmpDir, "some-file"))).To(BeFalse())
				Expect(ioutil.ReadFile(filepath.Join(tmpDir, "some-other-file"))).To(Equal([]byte("some-contents")))
			})
		})

		Context("when the source does not exist", func() {
			It("should return an error", func() {
				Expect(fs.Move(filepath.Join(tmpDir, "some-bad-file"), filepath.Join(tmpDir, "some-other-file"))).To(MatchError(ContainSubstring(fmt.Sprintf("failed to move %s to %s:", filepath.Join(tmpDir, "some-bad-file"), filepath.Join(tmpDir, "some-other-file")))))
			})
		})
	})

	Describe("#Copy", func() {
		Context("when the source exists and destination does not exist", func() {
			BeforeEach(func() {
				Expect(ioutil.WriteFile(filepath.Join(tmpDir, "some-file"), []byte("some-contents"), 0644)).To(Succeed())
			})

			It("should create the destination directory and copy the file", func() {
				Expect(fs.Copy(filepath.Join(tmpDir, "some-file"), filepath.Join(tmpDir, "some-dir", "some-file"))).To(Succeed())
				Expect(ioutil.ReadFile(filepath.Join(tmpDir, "some-dir", "some-file"))).To(Equal([]byte("some-contents")))
			})
		})

		Context("when the source exists and destination exists", func() {
			BeforeEach(func() {
				Expect(ioutil.WriteFile(filepath.Join(tmpDir, "some-file"), []byte("some-contents"), 0644)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(tmpDir, "some-other-file"), []byte("some-other-contents"), 0644)).To(Succeed())
			})

			It("should replace the destination file", func() {
				Expect(fs.Copy(filepath.Join(tmpDir, "some-file"), filepath.Join(tmpDir, "some-other-file"))).To(Succeed())
				Expect(ioutil.ReadFile(filepath.Join(tmpDir, "some-other-file"))).To(Equal([]byte("some-contents")))
			})
		})

		Context("when the source does not exist", func() {
			It("should return an error", func() {
				Expect(fs.Copy(filepath.Join(tmpDir, "some-bad-file"), filepath.Join(tmpDir, "some-other-file"))).To(MatchError(ContainSubstring(fmt.Sprintf("open %s:", filepath.Join(tmpDir, "some-bad-file")))))
			})
		})

		Context("when the destination cannot be written to", func() {
			BeforeEach(func() {
				Expect(ioutil.WriteFile(filepath.Join(tmpDir, "some-file"), []byte("some-contents"), 0644)).To(Succeed())
				Expect(os.Mkdir(filepath.Join(tmpDir, "some-dir"), 0755)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(tmpDir, "some-dir", "some-other-file"), []byte("some-other-contents"), 0644)).To(Succeed())
			})

			It("should return an error", func() {
				Expect(fs.Copy(filepath.Join(tmpDir, "some-file"), filepath.Join(tmpDir, "some-dir"))).To(MatchError(ContainSubstring(fmt.Sprintf("open %s: is a directory", filepath.Join(tmpDir, "some-dir")))))
			})
		})
	})

	Describe("#Extract", func() {
		BeforeEach(func() {
			file := struct{ Name, Body string }{"some-file.txt", "some-contents"}
			buf := new(bytes.Buffer)
			tarWriter := tar.NewWriter(buf)
			Expect(tarWriter.WriteHeader(&tar.Header{
				Name: "some-other-file.txt",
				Mode: 0600,
				Size: int64(len("some-other-contents")),
			})).To(Succeed())
			_, err := tarWriter.Write([]byte("some-other-contents"))
			Expect(tarWriter.WriteHeader(&tar.Header{
				Name: file.Name,
				Mode: 0600,
				Size: int64(len(file.Body)),
			})).To(Succeed())
			_, err = tarWriter.Write([]byte(file.Body))
			Expect(err).NotTo(HaveOccurred())
			Expect(tarWriter.Close()).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(tmpDir, "some-tar"), buf.Bytes(), 0644)).To(Succeed())
		})

		It("should extract the matching file from the archive to the destination", func() {
			Expect(
				fs.Extract(
					filepath.Join(tmpDir, "some-tar"),
					filepath.Join(tmpDir, "some-file.txt"),
					`some-file\.\w*`),
			).To(Succeed())
			Expect(ioutil.ReadFile(filepath.Join(tmpDir, "some-file.txt"))).To(Equal([]byte("some-contents")))
			_, err := os.Stat(filepath.Join(tmpDir, "some-other-file.txt"))
			Expect(os.IsNotExist(err)).To(BeTrue())
		})

		Context("when no matching file exists in the archive", func() {
			It("should return an error", func() {
				Expect(
					fs.Extract(
						filepath.Join(tmpDir, "some-tar"),
						filepath.Join(tmpDir, "some-bad-file.txt"),
						"some-bad-file.txt"),
				).To(
					MatchError(fmt.Sprintf("could not find file matching some-bad-file.txt in %s", filepath.Join(tmpDir, "some-tar"))))
			})
		})

		Context("when the archive does not exist", func() {
			It("should return an error", func() {
				Expect(
					fs.Extract(
						"some-bad-archive",
						filepath.Join(tmpDir, "some-file.txt"),
						"some-file.txt"),
				).To(
					MatchError(ContainSubstring("failed to open some-bad-archive:")))
			})
		})

		Context("when the archive is malformed", func() {
			BeforeEach(func() {
				Expect(ioutil.WriteFile(filepath.Join(tmpDir, "some-bad-tar"), []byte("not-an-archive"), 0644)).To(Succeed())
			})

			It("should return an error", func() {
				Expect(
					fs.Extract(
						filepath.Join(tmpDir, "some-bad-tar"),
						filepath.Join(tmpDir, "some-file.txt"),
						"some-file.txt"),
				).To(
					MatchError(ContainSubstring(fmt.Sprintf("malformed tar %s:", filepath.Join(tmpDir, "some-bad-tar")))))
			})
		})
	})
})

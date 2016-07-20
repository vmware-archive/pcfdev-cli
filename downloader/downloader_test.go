package downloader_test

import (
	"errors"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/golang/mock/gomock"
	"github.com/pivotal-cf/pcfdev-cli/config"
	dl "github.com/pivotal-cf/pcfdev-cli/downloader"
	"github.com/pivotal-cf/pcfdev-cli/downloader/mocks"
	"github.com/pivotal-cf/pcfdev-cli/pivnet"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Downloader", func() {
	var (
		downloader *dl.Downloader
		mockCtrl   *gomock.Controller
		mockClient *mocks.MockClient
		mockFS     *mocks.MockFS
		mockToken  *mocks.MockToken
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = mocks.NewMockClient(mockCtrl)
		mockFS = mocks.NewMockFS(mockCtrl)
		mockToken = mocks.NewMockToken(mockCtrl)

		downloader = &dl.Downloader{
			PivnetClient: mockClient,
			FS:           mockFS,
			Config: &config.Config{
				OVADir:        "some-ova-dir",
				DefaultVMName: "some-vm",
				ExpectedMD5:   "some-md5",
			},
			Token: mockToken,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("IsOVACurrent", func() {
		Context("when OVA does not exist", func() {
			It("should return false", func() {
				mockFS.EXPECT().Exists(filepath.Join("some-ova-dir", "some-vm.ova")).Return(false, nil)

				current, err := downloader.IsOVACurrent()
				Expect(err).NotTo(HaveOccurred())
				Expect(current).To(BeFalse())

			})
		})
		Context("when OVA exists and has correct MD5", func() {
			It("should return true", func() {
				gomock.InOrder(
					mockFS.EXPECT().Exists(filepath.Join("some-ova-dir", "some-vm.ova")).Return(true, nil),
					mockFS.EXPECT().MD5(filepath.Join("some-ova-dir", "some-vm.ova")).Return("some-md5", nil),
				)

				current, err := downloader.IsOVACurrent()
				Expect(err).NotTo(HaveOccurred())
				Expect(current).To(BeTrue())

			})
		})
		Context("when OVA exists and has incorrect MD5", func() {
			It("should return false", func() {
				gomock.InOrder(
					mockFS.EXPECT().Exists(filepath.Join("some-ova-dir", "some-vm.ova")).Return(true, nil),
					mockFS.EXPECT().MD5(filepath.Join("some-ova-dir", "some-vm.ova")).Return("some-bad-md5", nil),
				)

				current, err := downloader.IsOVACurrent()
				Expect(err).NotTo(HaveOccurred())
				Expect(current).To(BeFalse())

			})
		})

		Context("when checking if the file exists fails", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockFS.EXPECT().Exists(filepath.Join("some-ova-dir", "some-vm.ova")).Return(false, errors.New("some-error")),
				)

				_, err := downloader.IsOVACurrent()
				Expect(err).To(MatchError("some-error"))
			})
		})

		Context("when checking the MD5 fails", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockFS.EXPECT().Exists(filepath.Join("some-ova-dir", "some-vm.ova")).Return(true, nil),
					mockFS.EXPECT().MD5(filepath.Join("some-ova-dir", "some-vm.ova")).Return("", errors.New("some-error")),
				)

				_, err := downloader.IsOVACurrent()
				Expect(err).To(MatchError("some-error"))
			})
		})
	})

	Describe("#Download", func() {
		Context("when file and partial file do not exist", func() {
			It("should download the file", func() {
				readCloser := &pivnet.DownloadReader{ReadCloser: ioutil.NopCloser(strings.NewReader("some-ova-contents"))}
				gomock.InOrder(
					mockFS.EXPECT().CreateDir(filepath.Join("some-ova-dir")).Return(nil),
					mockFS.EXPECT().DeleteAllExcept("some-ova-dir", []string{"some-vm.ova", "some-vm.ova.partial"}).Return(nil),
					mockFS.EXPECT().Exists(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return(false, nil),
					mockClient.EXPECT().DownloadOVA(int64(0)).Return(readCloser, nil),
					mockToken.EXPECT().Save(),
					mockFS.EXPECT().Write(filepath.Join("some-ova-dir", "some-vm.ova.partial"), readCloser).Return(nil),
					mockFS.EXPECT().MD5(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return("some-md5", nil),
					mockFS.EXPECT().Move(filepath.Join("some-ova-dir", "some-vm.ova.partial"), filepath.Join("some-ova-dir", "some-vm.ova")),
				)

				Expect(downloader.Download()).To(Succeed())
			})
		})

		Context("when partial file does exist", func() {
			It("should resume the download of the partial file", func() {
				readCloser := &pivnet.DownloadReader{ReadCloser: ioutil.NopCloser(strings.NewReader("some-ova-contents"))}
				gomock.InOrder(
					mockFS.EXPECT().CreateDir(filepath.Join("some-ova-dir")).Return(nil),
					mockFS.EXPECT().DeleteAllExcept("some-ova-dir", []string{"some-vm.ova", "some-vm.ova.partial"}).Return(nil),
					mockFS.EXPECT().Exists(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return(true, nil),
					mockFS.EXPECT().Length(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return(int64(25), nil),
					mockClient.EXPECT().DownloadOVA(int64(25)).Return(readCloser, nil),
					mockToken.EXPECT().Save(),
					mockFS.EXPECT().Write(filepath.Join("some-ova-dir", "some-vm.ova.partial"), readCloser).Return(nil),
					mockFS.EXPECT().MD5(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return("some-md5", nil),
					mockFS.EXPECT().Move(filepath.Join("some-ova-dir", "some-vm.ova.partial"), filepath.Join("some-ova-dir", "some-vm.ova")),
				)

				Expect(downloader.Download()).To(Succeed())
			})
		})

		Context("when partial file is downloaded but the checksum is not valid and the re-download succeeds", func() {
			It("should move the file to the downloaded path", func() {
				readCloser := &pivnet.DownloadReader{ReadCloser: ioutil.NopCloser(strings.NewReader("some-ova-contents"))}
				gomock.InOrder(
					mockFS.EXPECT().CreateDir(filepath.Join("some-ova-dir")).Return(nil),
					mockFS.EXPECT().DeleteAllExcept("some-ova-dir", []string{"some-vm.ova", "some-vm.ova.partial"}).Return(nil),
					mockFS.EXPECT().Exists(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return(true, nil),
					mockFS.EXPECT().Length(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return(int64(25), nil),
					mockClient.EXPECT().DownloadOVA(int64(25)).Return(readCloser, nil),
					mockToken.EXPECT().Save(),
					mockFS.EXPECT().Write(filepath.Join("some-ova-dir", "some-vm.ova.partial"), readCloser).Return(nil),
					mockFS.EXPECT().MD5(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return("some-bad-md5", nil),
					mockFS.EXPECT().Remove(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return(nil),

					mockClient.EXPECT().DownloadOVA(int64(0)).Return(readCloser, nil),
					mockToken.EXPECT().Save(),
					mockFS.EXPECT().Write(filepath.Join("some-ova-dir", "some-vm.ova.partial"), readCloser).Return(nil),
					mockFS.EXPECT().MD5(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return("some-md5", nil),
					mockFS.EXPECT().Move(filepath.Join("some-ova-dir", "some-vm.ova.partial"), filepath.Join("some-ova-dir", "some-vm.ova")),
				)

				Expect(downloader.Download()).To(Succeed())
			})
		})

		Context("when partial file is downloaded but the checksum is not valid and the re-download fails", func() {
			It("should return an error", func() {
				readCloser := &pivnet.DownloadReader{ReadCloser: ioutil.NopCloser(strings.NewReader("some-ova-contents"))}
				gomock.InOrder(
					mockFS.EXPECT().CreateDir(filepath.Join("some-ova-dir")).Return(nil),
					mockFS.EXPECT().DeleteAllExcept("some-ova-dir", []string{"some-vm.ova", "some-vm.ova.partial"}).Return(nil),
					mockFS.EXPECT().Exists(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return(true, nil),
					mockFS.EXPECT().Length(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return(int64(25), nil),
					mockClient.EXPECT().DownloadOVA(int64(25)).Return(readCloser, nil),
					mockToken.EXPECT().Save(),
					mockFS.EXPECT().Write(filepath.Join("some-ova-dir", "some-vm.ova.partial"), readCloser).Return(nil),
					mockFS.EXPECT().MD5(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return("some-bad-md5", nil),
					mockFS.EXPECT().Remove(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return(nil),

					mockClient.EXPECT().DownloadOVA(int64(0)).Return(readCloser, nil),
					mockToken.EXPECT().Save(),
					mockFS.EXPECT().Write(filepath.Join("some-ova-dir", "some-vm.ova.partial"), readCloser).Return(nil),
					mockFS.EXPECT().MD5(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return("some-bad-md5", nil),
				)

				Expect(downloader.Download()).To(MatchError("download failed"))
			})
		})
	})

	Context("when creating the directory fails", func() {
		It("should return an error", func() {
			gomock.InOrder(
				mockFS.EXPECT().CreateDir(filepath.Join("some-ova-dir")).Return(errors.New("some-error")),
			)

			Expect(downloader.Download()).To(MatchError("some-error"))
		})
	})

	Context("when deleting files fails", func() {
		It("should return an error", func() {
			gomock.InOrder(
				mockFS.EXPECT().CreateDir(filepath.Join("some-ova-dir")).Return(nil),
				mockFS.EXPECT().DeleteAllExcept("some-ova-dir", []string{"some-vm.ova", "some-vm.ova.partial"}).Return(errors.New("some-error")),
			)

			Expect(downloader.Download()).To(MatchError("some-error"))
		})
	})

	Context("when checking if the partial file exists", func() {
		It("should return an error", func() {
			gomock.InOrder(
				mockFS.EXPECT().CreateDir(filepath.Join("some-ova-dir")).Return(nil),
				mockFS.EXPECT().DeleteAllExcept("some-ova-dir", []string{"some-vm.ova", "some-vm.ova.partial"}).Return(nil),
				mockFS.EXPECT().Exists(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return(false, errors.New("some-error")),
			)

			Expect(downloader.Download()).To(MatchError("some-error"))
		})
	})

	Context("when checking the length of the partial file fails", func() {
		It("should return an error", func() {
			gomock.InOrder(
				mockFS.EXPECT().CreateDir(filepath.Join("some-ova-dir")).Return(nil),
				mockFS.EXPECT().DeleteAllExcept("some-ova-dir", []string{"some-vm.ova", "some-vm.ova.partial"}).Return(nil),
				mockFS.EXPECT().Exists(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return(true, nil),
				mockFS.EXPECT().Length(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return(int64(0), errors.New("some-error")),
			)

			Expect(downloader.Download()).To(MatchError("some-error"))
		})
	})

	Context("when downloading the file fails", func() {
		It("should return an error", func() {
			gomock.InOrder(
				mockFS.EXPECT().CreateDir(filepath.Join("some-ova-dir")).Return(nil),
				mockFS.EXPECT().DeleteAllExcept("some-ova-dir", []string{"some-vm.ova", "some-vm.ova.partial"}).Return(nil),
				mockFS.EXPECT().Exists(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return(false, nil),
				mockClient.EXPECT().DownloadOVA(int64(0)).Return(nil, errors.New("some-error")),
			)

			Expect(downloader.Download()).To(MatchError("some-error"))
		})
	})

	Context("when saving the Pivnet API token fails", func() {
		It("should return an error", func() {
			readCloser := &pivnet.DownloadReader{ReadCloser: ioutil.NopCloser(strings.NewReader("some-ova-contents"))}
			gomock.InOrder(
				mockFS.EXPECT().CreateDir(filepath.Join("some-ova-dir")).Return(nil),
				mockFS.EXPECT().DeleteAllExcept("some-ova-dir", []string{"some-vm.ova", "some-vm.ova.partial"}).Return(nil),
				mockFS.EXPECT().Exists(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return(false, nil),
				mockClient.EXPECT().DownloadOVA(int64(0)).Return(readCloser, nil),
				mockToken.EXPECT().Save().Return(errors.New("some-error")),
			)

			Expect(downloader.Download()).To(MatchError("some-error"))
		})
	})

	Context("when writing the downloaded file fails", func() {
		It("should return an error", func() {
			readCloser := &pivnet.DownloadReader{ReadCloser: ioutil.NopCloser(strings.NewReader("some-ova-contents"))}
			gomock.InOrder(
				mockFS.EXPECT().CreateDir(filepath.Join("some-ova-dir")).Return(nil),
				mockFS.EXPECT().DeleteAllExcept("some-ova-dir", []string{"some-vm.ova", "some-vm.ova.partial"}).Return(nil),
				mockFS.EXPECT().Exists(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return(false, nil),
				mockClient.EXPECT().DownloadOVA(int64(0)).Return(readCloser, nil),
				mockToken.EXPECT().Save(),
				mockFS.EXPECT().Write(filepath.Join("some-ova-dir", "some-vm.ova.partial"), readCloser).Return(errors.New("some-error")),
			)

			Expect(downloader.Download()).To(MatchError("some-error"))
		})
	})

	Context("when checking the MD5 of the downloaded file fails", func() {
		It("should return an error", func() {
			readCloser := &pivnet.DownloadReader{ReadCloser: ioutil.NopCloser(strings.NewReader("some-ova-contents"))}
			gomock.InOrder(
				mockFS.EXPECT().CreateDir(filepath.Join("some-ova-dir")).Return(nil),
				mockFS.EXPECT().DeleteAllExcept("some-ova-dir", []string{"some-vm.ova", "some-vm.ova.partial"}).Return(nil),
				mockFS.EXPECT().Exists(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return(false, nil),
				mockClient.EXPECT().DownloadOVA(int64(0)).Return(readCloser, nil),
				mockToken.EXPECT().Save(),
				mockFS.EXPECT().Write(filepath.Join("some-ova-dir", "some-vm.ova.partial"), readCloser).Return(nil),
				mockFS.EXPECT().MD5(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return("", errors.New("some-error")),
			)

			Expect(downloader.Download()).To(MatchError("some-error"))
		})
	})

	Context("when removing the partial file fails", func() {
		It("should return an error", func() {
			readCloser := &pivnet.DownloadReader{ReadCloser: ioutil.NopCloser(strings.NewReader("some-ova-contents"))}
			gomock.InOrder(
				mockFS.EXPECT().CreateDir(filepath.Join("some-ova-dir")).Return(nil),
				mockFS.EXPECT().DeleteAllExcept("some-ova-dir", []string{"some-vm.ova", "some-vm.ova.partial"}).Return(nil),
				mockFS.EXPECT().Exists(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return(true, nil),
				mockFS.EXPECT().Length(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return(int64(25), nil),
				mockClient.EXPECT().DownloadOVA(int64(25)).Return(readCloser, nil),
				mockToken.EXPECT().Save(),
				mockFS.EXPECT().Write(filepath.Join("some-ova-dir", "some-vm.ova.partial"), readCloser).Return(nil),
				mockFS.EXPECT().MD5(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return("some-bad-md5", nil),
				mockFS.EXPECT().Remove(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return(errors.New("some-error")),
			)

			Expect(downloader.Download()).To(MatchError("some-error"))
		})
	})

	Context("when the MD5 of a file download does not match the expected MD5", func() {
		It("should return an error", func() {
			readCloser := &pivnet.DownloadReader{ReadCloser: ioutil.NopCloser(strings.NewReader("some-ova-contents"))}
			gomock.InOrder(
				mockFS.EXPECT().CreateDir(filepath.Join("some-ova-dir")).Return(nil),
				mockFS.EXPECT().DeleteAllExcept("some-ova-dir", []string{"some-vm.ova", "some-vm.ova.partial"}).Return(nil),
				mockFS.EXPECT().Exists(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return(false, nil),
				mockClient.EXPECT().DownloadOVA(int64(0)).Return(readCloser, nil),
				mockToken.EXPECT().Save(),
				mockFS.EXPECT().Write(filepath.Join("some-ova-dir", "some-vm.ova.partial"), readCloser).Return(nil),
				mockFS.EXPECT().MD5(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return("some-bad-md5", nil),
			)

			Expect(downloader.Download()).To(MatchError("download failed"))
		})
	})

	Context("when downloading the file fails after downloading the partial file failed", func() {
		It("should return an error", func() {
			readCloser := &pivnet.DownloadReader{ReadCloser: ioutil.NopCloser(strings.NewReader("some-ova-contents"))}
			gomock.InOrder(
				mockFS.EXPECT().CreateDir(filepath.Join("some-ova-dir")).Return(nil),
				mockFS.EXPECT().DeleteAllExcept("some-ova-dir", []string{"some-vm.ova", "some-vm.ova.partial"}).Return(nil),
				mockFS.EXPECT().Exists(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return(true, nil),
				mockFS.EXPECT().Length(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return(int64(25), nil),
				mockClient.EXPECT().DownloadOVA(int64(25)).Return(readCloser, nil),
				mockToken.EXPECT().Save(),
				mockFS.EXPECT().Write(filepath.Join("some-ova-dir", "some-vm.ova.partial"), readCloser).Return(nil),
				mockFS.EXPECT().MD5(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return("some-bad-md5", nil),
				mockFS.EXPECT().Remove(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return(nil),
				mockClient.EXPECT().DownloadOVA(int64(0)).Return(nil, errors.New("some-error")),
			)

			Expect(downloader.Download()).To(MatchError("some-error"))
		})
	})

	Context("when saving the Pivnet API token fails after downloading the partial file failed", func() {
		It("should return an error", func() {
			readCloser := &pivnet.DownloadReader{ReadCloser: ioutil.NopCloser(strings.NewReader("some-ova-contents"))}
			gomock.InOrder(
				mockFS.EXPECT().CreateDir(filepath.Join("some-ova-dir")).Return(nil),
				mockFS.EXPECT().DeleteAllExcept("some-ova-dir", []string{"some-vm.ova", "some-vm.ova.partial"}).Return(nil),
				mockFS.EXPECT().Exists(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return(true, nil),
				mockFS.EXPECT().Length(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return(int64(25), nil),
				mockClient.EXPECT().DownloadOVA(int64(25)).Return(readCloser, nil),
				mockToken.EXPECT().Save(),
				mockFS.EXPECT().Write(filepath.Join("some-ova-dir", "some-vm.ova.partial"), readCloser).Return(nil),
				mockFS.EXPECT().MD5(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return("some-bad-md5", nil),
				mockFS.EXPECT().Remove(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return(nil),
				mockClient.EXPECT().DownloadOVA(int64(0)).Return(readCloser, nil),
				mockToken.EXPECT().Save().Return(errors.New("some-error")),
			)

			Expect(downloader.Download()).To(MatchError("some-error"))
		})
	})

	Context("when writing the file fails after downloading the partial file failed", func() {
		It("should return an error", func() {
			readCloser := &pivnet.DownloadReader{ReadCloser: ioutil.NopCloser(strings.NewReader("some-ova-contents"))}
			gomock.InOrder(
				mockFS.EXPECT().CreateDir(filepath.Join("some-ova-dir")).Return(nil),
				mockFS.EXPECT().DeleteAllExcept("some-ova-dir", []string{"some-vm.ova", "some-vm.ova.partial"}).Return(nil),
				mockFS.EXPECT().Exists(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return(true, nil),
				mockFS.EXPECT().Length(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return(int64(25), nil),
				mockClient.EXPECT().DownloadOVA(int64(25)).Return(readCloser, nil),
				mockToken.EXPECT().Save(),
				mockFS.EXPECT().Write(filepath.Join("some-ova-dir", "some-vm.ova.partial"), readCloser).Return(nil),
				mockFS.EXPECT().MD5(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return("some-bad-md5", nil),
				mockFS.EXPECT().Remove(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return(nil),
				mockClient.EXPECT().DownloadOVA(int64(0)).Return(readCloser, nil),
				mockToken.EXPECT().Save(),
				mockFS.EXPECT().Write(filepath.Join("some-ova-dir", "some-vm.ova.partial"), readCloser).Return(errors.New("some-error")),
			)

			Expect(downloader.Download()).To(MatchError("some-error"))
		})
	})

	Context("when checking the MD5 of the file fails after downloading the partial file failed", func() {
		It("should return an error", func() {
			readCloser := &pivnet.DownloadReader{ReadCloser: ioutil.NopCloser(strings.NewReader("some-ova-contents"))}
			gomock.InOrder(
				mockFS.EXPECT().CreateDir(filepath.Join("some-ova-dir")).Return(nil),
				mockFS.EXPECT().DeleteAllExcept("some-ova-dir", []string{"some-vm.ova", "some-vm.ova.partial"}).Return(nil),
				mockFS.EXPECT().Exists(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return(true, nil),
				mockFS.EXPECT().Length(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return(int64(25), nil),
				mockClient.EXPECT().DownloadOVA(int64(25)).Return(readCloser, nil),
				mockToken.EXPECT().Save(),
				mockFS.EXPECT().Write(filepath.Join("some-ova-dir", "some-vm.ova.partial"), readCloser).Return(nil),
				mockFS.EXPECT().MD5(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return("some-bad-md5", nil),
				mockFS.EXPECT().Remove(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return(nil),
				mockClient.EXPECT().DownloadOVA(int64(0)).Return(readCloser, nil),
				mockToken.EXPECT().Save(),
				mockFS.EXPECT().Write(filepath.Join("some-ova-dir", "some-vm.ova.partial"), readCloser).Return(nil),
				mockFS.EXPECT().MD5(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return("", errors.New("some-error")),
			)

			Expect(downloader.Download()).To(MatchError("some-error"))
		})
	})

	Context("when moving the file fails", func() {
		It("should return an error", func() {
			readCloser := &pivnet.DownloadReader{ReadCloser: ioutil.NopCloser(strings.NewReader("some-ova-contents"))}
			gomock.InOrder(
				mockFS.EXPECT().CreateDir(filepath.Join("some-ova-dir")).Return(nil),
				mockFS.EXPECT().DeleteAllExcept("some-ova-dir", []string{"some-vm.ova", "some-vm.ova.partial"}).Return(nil),
				mockFS.EXPECT().Exists(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return(false, nil),
				mockClient.EXPECT().DownloadOVA(int64(0)).Return(readCloser, nil),
				mockToken.EXPECT().Save(),
				mockFS.EXPECT().Write(filepath.Join("some-ova-dir", "some-vm.ova.partial"), readCloser).Return(nil),
				mockFS.EXPECT().MD5(filepath.Join("some-ova-dir", "some-vm.ova.partial")).Return("some-md5", nil),
				mockFS.EXPECT().Move(filepath.Join("some-ova-dir", "some-vm.ova.partial"), filepath.Join("some-ova-dir", "some-vm.ova")).Return(errors.New("some-error")),
			)

			Expect(downloader.Download()).To(MatchError("some-error"))
		})
	})
})

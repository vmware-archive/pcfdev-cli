package downloader_test

import (
	"errors"
	"io/ioutil"
	"strings"

	"github.com/golang/mock/gomock"
	"github.com/pivotal-cf/pcfdev-cli/config"
	dl "github.com/pivotal-cf/pcfdev-cli/downloader"
	"github.com/pivotal-cf/pcfdev-cli/downloader/mocks"
	"github.com/pivotal-cf/pcfdev-cli/pivnet"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ConcreteOVADownloader", func() {
	var (
		downloader *dl.ConcreteOVADownloader
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

		downloader = &dl.ConcreteOVADownloader{
			PivnetClient: mockClient,
			FS:           mockFS,
			Config: &config.Config{
				OVADir:         "some-ova-dir",
				OVAPath:        "some-ova-path",
				PartialOVAPath: "some-partial-ova-path",
				DefaultVMName:  "some-vm",
				ExpectedMD5:    "some-md5",
			},
			Token:                mockToken,
			DownloadAttempts:     2,
			DownloadAttemptDelay: 0,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("IsOVACurrent", func() {
		Context("when OVA does not exist", func() {
			It("should return false", func() {
				mockFS.EXPECT().Exists("some-ova-path").Return(false, nil)

				Expect(downloader.IsOVACurrent()).To(BeFalse())
			})
		})

		Context("when OVA exists and has correct MD5", func() {
			It("should return true", func() {
				gomock.InOrder(
					mockFS.EXPECT().Exists("some-ova-path").Return(true, nil),
					mockFS.EXPECT().MD5("some-ova-path").Return("some-md5", nil),
				)

				Expect(downloader.IsOVACurrent()).To(BeTrue())
			})
		})

		Context("when OVA exists and has incorrect MD5", func() {
			It("should return false", func() {
				gomock.InOrder(
					mockFS.EXPECT().Exists("some-ova-path").Return(true, nil),
					mockFS.EXPECT().MD5("some-ova-path").Return("some-bad-md5", nil),
				)

				Expect(downloader.IsOVACurrent()).To(BeFalse())
			})
		})

		Context("when checking if the file exists fails", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockFS.EXPECT().Exists("some-ova-path").Return(false, errors.New("some-error")),
				)

				_, err := downloader.IsOVACurrent()
				Expect(err).To(MatchError("some-error"))
			})
		})

		Context("when checking the MD5 fails", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockFS.EXPECT().Exists("some-ova-path").Return(true, nil),
					mockFS.EXPECT().MD5("some-ova-path").Return("", errors.New("some-error")),
				)

				_, err := downloader.IsOVACurrent()
				Expect(err).To(MatchError("some-error"))
			})
		})
	})

	Describe("#Setup", func() {
		It("should get ready for a download", func() {
			gomock.InOrder(
				mockFS.EXPECT().CreateDir("some-ova-dir"),
				mockFS.EXPECT().DeleteAllExcept("some-ova-dir", []string{"some-vm.ova", "some-vm.ova.partial"}),
			)

			Expect(downloader.Setup()).To(Succeed())
		})

		Context("when create the ova dir fails", func() {
			It("should return an error", func() {
				mockFS.EXPECT().CreateDir("some-ova-dir").Return(errors.New("some-error"))

				Expect(downloader.Setup()).To(MatchError("some-error"))
			})
		})

		Context("when deleting unspecified files in the ova dir", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockFS.EXPECT().CreateDir("some-ova-dir"),
					mockFS.EXPECT().DeleteAllExcept("some-ova-dir", []string{"some-vm.ova", "some-vm.ova.partial"}).Return(errors.New("some-error")),
				)

				Expect(downloader.Setup()).To(MatchError("some-error"))
			})
		})
	})

	Describe("#Download", func() {
		Context("when there is no partial ova present", func() {
			It("should download the file from the beginning", func() {
				readCloser := &pivnet.DownloadReader{ReadCloser: ioutil.NopCloser(strings.NewReader("some-ova-contents"))}
				gomock.InOrder(
					mockFS.EXPECT().Exists("some-partial-ova-path").Return(false, nil),
					mockClient.EXPECT().DownloadOVA(int64(0)).Return(readCloser, nil),
					mockToken.EXPECT().Save(),
					mockFS.EXPECT().Write("some-partial-ova-path", readCloser, true),
					mockFS.EXPECT().MD5("some-partial-ova-path").Return("some-md5", nil),
				)

				md5, err := downloader.Download()
				Expect(err).NotTo(HaveOccurred())
				Expect(md5).To(Equal("some-md5"))
			})

			Context("when there is an issue seeing if the partial ova exists", func() {
				It("should retry the check", func() {
					readCloser := &pivnet.DownloadReader{ReadCloser: ioutil.NopCloser(strings.NewReader("some-ova-contents"))}
					gomock.InOrder(
						mockFS.EXPECT().Exists("some-partial-ova-path").Return(false, errors.New("some-error")),
						mockFS.EXPECT().Exists("some-partial-ova-path").Return(false, nil),
						mockClient.EXPECT().DownloadOVA(int64(0)).Return(readCloser, nil),
						mockToken.EXPECT().Save(),
						mockFS.EXPECT().Write("some-partial-ova-path", readCloser, true),
						mockFS.EXPECT().MD5("some-partial-ova-path").Return("some-md5", nil),
					)

					md5, err := downloader.Download()
					Expect(err).NotTo(HaveOccurred())
					Expect(md5).To(Equal("some-md5"))
				})
			})

			Context("when there is an issue seeing if the partial ova exists twice", func() {
				It("should return an error", func() {
					gomock.InOrder(
						mockFS.EXPECT().Exists("some-partial-ova-path").Return(false, errors.New("some-error")),
						mockFS.EXPECT().Exists("some-partial-ova-path").Return(false, errors.New("some-error")),
					)

					_, err := downloader.Download()
					Expect(err).To(MatchError("some-error"))
				})
			})

			Context("when there is an issue downloading the OVA", func() {
				It("should retry the download again", func() {
					readCloser := &pivnet.DownloadReader{ReadCloser: ioutil.NopCloser(strings.NewReader("some-ova-contents"))}
					gomock.InOrder(
						mockFS.EXPECT().Exists("some-partial-ova-path").Return(false, nil),
						mockClient.EXPECT().DownloadOVA(int64(0)).Return(nil, errors.New("some-error")),

						mockFS.EXPECT().Exists("some-partial-ova-path").Return(false, nil),
						mockClient.EXPECT().DownloadOVA(int64(0)).Return(readCloser, nil),

						mockToken.EXPECT().Save(),
						mockFS.EXPECT().Write("some-partial-ova-path", readCloser, true),
						mockFS.EXPECT().MD5("some-partial-ova-path").Return("some-md5", nil),
					)

					md5, err := downloader.Download()
					Expect(err).NotTo(HaveOccurred())
					Expect(md5).To(Equal("some-md5"))
				})
			})

			Context("when there is an issue downloading the OVA when retrying", func() {
				It("should return an error", func() {
					gomock.InOrder(
						mockFS.EXPECT().Exists("some-partial-ova-path").Return(false, nil),
						mockClient.EXPECT().DownloadOVA(int64(0)).Return(nil, errors.New("some-error")),

						mockFS.EXPECT().Exists("some-partial-ova-path").Return(false, nil),
						mockClient.EXPECT().DownloadOVA(int64(0)).Return(nil, errors.New("some-error")),
					)

					_, err := downloader.Download()
					Expect(err).To(MatchError("some-error"))
				})
			})

			Context("when there is an issue saving the token", func() {
				It("should retry the save again", func() {
					readCloser := &pivnet.DownloadReader{ReadCloser: ioutil.NopCloser(strings.NewReader("some-ova-contents"))}
					gomock.InOrder(
						mockFS.EXPECT().Exists("some-partial-ova-path").Return(false, nil),
						mockClient.EXPECT().DownloadOVA(int64(0)).Return(readCloser, nil),
						mockToken.EXPECT().Save().Return(errors.New("some-error")),

						mockFS.EXPECT().Exists("some-partial-ova-path").Return(false, nil),
						mockClient.EXPECT().DownloadOVA(int64(0)).Return(readCloser, nil),
						mockToken.EXPECT().Save(),

						mockFS.EXPECT().Write("some-partial-ova-path", readCloser, true),
						mockFS.EXPECT().MD5("some-partial-ova-path").Return("some-md5", nil),
					)

					md5, err := downloader.Download()
					Expect(err).NotTo(HaveOccurred())
					Expect(md5).To(Equal("some-md5"))
				})
			})

			Context("when there is an issue saving the token twice", func() {
				It("should return an error", func() {
					readCloser := &pivnet.DownloadReader{ReadCloser: ioutil.NopCloser(strings.NewReader("some-ova-contents"))}
					gomock.InOrder(
						mockFS.EXPECT().Exists("some-partial-ova-path").Return(false, nil),
						mockClient.EXPECT().DownloadOVA(int64(0)).Return(readCloser, nil),
						mockToken.EXPECT().Save().Return(errors.New("some-error")),

						mockFS.EXPECT().Exists("some-partial-ova-path").Return(false, nil),
						mockClient.EXPECT().DownloadOVA(int64(0)).Return(readCloser, nil),
						mockToken.EXPECT().Save().Return(errors.New("some-error")),
					)

					_, err := downloader.Download()
					Expect(err).To(MatchError("some-error"))
				})
			})

			Context("when there is an issue writing the ova", func() {
				It("should retry the write again", func() {
					readCloser := &pivnet.DownloadReader{ReadCloser: ioutil.NopCloser(strings.NewReader("some-ova-contents"))}
					gomock.InOrder(
						mockFS.EXPECT().Exists("some-partial-ova-path").Return(false, nil),
						mockClient.EXPECT().DownloadOVA(int64(0)).Return(readCloser, nil),
						mockToken.EXPECT().Save(),
						mockFS.EXPECT().Write("some-partial-ova-path", readCloser, true).Return(errors.New("some-error")),

						mockFS.EXPECT().Exists("some-partial-ova-path").Return(false, nil),
						mockClient.EXPECT().DownloadOVA(int64(0)).Return(readCloser, nil),
						mockToken.EXPECT().Save(),
						mockFS.EXPECT().Write("some-partial-ova-path", readCloser, true),
						mockFS.EXPECT().MD5("some-partial-ova-path").Return("some-md5", nil),
					)

					md5, err := downloader.Download()
					Expect(err).NotTo(HaveOccurred())
					Expect(md5).To(Equal("some-md5"))
				})
			})

			Context("when there is an issue writing the ova twice", func() {
				It("should return an error", func() {
					readCloser := &pivnet.DownloadReader{ReadCloser: ioutil.NopCloser(strings.NewReader("some-ova-contents"))}
					gomock.InOrder(
						mockFS.EXPECT().Exists("some-partial-ova-path").Return(false, nil),
						mockClient.EXPECT().DownloadOVA(int64(0)).Return(readCloser, nil),
						mockToken.EXPECT().Save(),
						mockFS.EXPECT().Write("some-partial-ova-path", readCloser, true).Return(errors.New("some-error")),

						mockFS.EXPECT().Exists("some-partial-ova-path").Return(false, nil),
						mockClient.EXPECT().DownloadOVA(int64(0)).Return(readCloser, nil),
						mockToken.EXPECT().Save(),
						mockFS.EXPECT().Write("some-partial-ova-path", readCloser, true).Return(errors.New("some-error")),
					)

					_, err := downloader.Download()
					Expect(err).To(MatchError("some-error"))
				})
			})

			Context("when there is an issue checking the md5 of the partial ova", func() {
				It("should return the error", func() {
					readCloser := &pivnet.DownloadReader{ReadCloser: ioutil.NopCloser(strings.NewReader("some-ova-contents"))}
					gomock.InOrder(
						mockFS.EXPECT().Exists("some-partial-ova-path").Return(false, nil),
						mockClient.EXPECT().DownloadOVA(int64(0)).Return(readCloser, nil),
						mockToken.EXPECT().Save(),
						mockFS.EXPECT().Write("some-partial-ova-path", readCloser, true),
						mockFS.EXPECT().MD5("some-partial-ova-path").Return("", errors.New("some-error")),
					)

					md5, err := downloader.Download()
					Expect(err).To(MatchError("some-error"))
					Expect(md5).To(BeEmpty())
				})
			})
		})

		Context("when there is a partial ova present", func() {
			It("should resume the download of the partial ova", func() {
				readCloser := &pivnet.DownloadReader{ReadCloser: ioutil.NopCloser(strings.NewReader("some-ova-contents"))}
				gomock.InOrder(
					mockFS.EXPECT().Exists("some-partial-ova-path").Return(true, nil),
					mockFS.EXPECT().Length("some-partial-ova-path").Return(int64(24), nil),
					mockClient.EXPECT().DownloadOVA(int64(24)).Return(readCloser, nil),
					mockToken.EXPECT().Save(),
					mockFS.EXPECT().Write("some-partial-ova-path", readCloser, true),
					mockFS.EXPECT().MD5("some-partial-ova-path").Return("some-md5", nil),
				)

				md5, err := downloader.Download()
				Expect(err).NotTo(HaveOccurred())
				Expect(md5).To(Equal("some-md5"))
			})

			Context("when something goes wrong saving the file", func() {
				It("should retry the download from where it failed", func() {
					readCloser := &pivnet.DownloadReader{ReadCloser: ioutil.NopCloser(strings.NewReader("some-ova-contents"))}
					gomock.InOrder(
						mockFS.EXPECT().Exists("some-partial-ova-path").Return(true, nil),
						mockFS.EXPECT().Length("some-partial-ova-path").Return(int64(24), nil),
						mockClient.EXPECT().DownloadOVA(int64(24)).Return(readCloser, nil),
						mockToken.EXPECT().Save(),
						mockFS.EXPECT().Write("some-partial-ova-path", readCloser, true).Return(errors.New("some-error")),

						mockFS.EXPECT().Exists("some-partial-ova-path").Return(true, nil),
						mockFS.EXPECT().Length("some-partial-ova-path").Return(int64(48), nil),
						mockClient.EXPECT().DownloadOVA(int64(48)).Return(readCloser, nil),
						mockToken.EXPECT().Save(),
						mockFS.EXPECT().Write("some-partial-ova-path", readCloser, true),
						mockFS.EXPECT().MD5("some-partial-ova-path").Return("some-md5", nil),
					)

					md5, err := downloader.Download()
					Expect(err).NotTo(HaveOccurred())
					Expect(md5).To(Equal("some-md5"))
				})
			})

			Context("when there is an issue getting the bytes of the partial ova", func() {
				It("should return an error", func() {
					gomock.InOrder(
						mockFS.EXPECT().Exists("some-partial-ova-path").Return(true, nil),
						mockFS.EXPECT().Length("some-partial-ova-path").Return(int64(24), errors.New("some-error")),
						mockFS.EXPECT().Exists("some-partial-ova-path").Return(true, nil),
						mockFS.EXPECT().Length("some-partial-ova-path").Return(int64(24), errors.New("some-error")),
					)

					_, err := downloader.Download()
					Expect(err).To(MatchError("some-error"))
				})
			})

		})
	})
})

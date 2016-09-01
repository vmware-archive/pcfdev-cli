package downloader_test

import (
	"errors"

	"github.com/golang/mock/gomock"
	"github.com/pivotal-cf/pcfdev-cli/config"
	dl "github.com/pivotal-cf/pcfdev-cli/downloader"
	"github.com/pivotal-cf/pcfdev-cli/downloader/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PartialDownloader", func() {
	var (
		downloader        *dl.PartialDownloader
		mockCtrl          *gomock.Controller
		mockOVADownloader *mocks.MockOVADownloader
		mockFS            *mocks.MockFS
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockFS = mocks.NewMockFS(mockCtrl)
		mockOVADownloader = mocks.NewMockOVADownloader(mockCtrl)
		downloader = &dl.PartialDownloader{
			FS:         mockFS,
			Downloader: mockOVADownloader,
			Config: &config.Config{
				OVAPath:        "some-ova-path",
				PartialOVAPath: "some-partial-ova-path",
				DefaultVMName:  "some-vm",
				ExpectedMD5:    "some-md5",
			},
		}

	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("#IsOVACurrent", func() {
		Context("when the ova is current", func() {
			It("should return true", func() {
				mockOVADownloader.EXPECT().IsOVACurrent().Return(true, nil)
				Expect(downloader.IsOVACurrent()).To(BeTrue())
			})
		})

		Context("when the ova is not current", func() {
			It("should return false", func() {
				mockOVADownloader.EXPECT().IsOVACurrent().Return(false, nil)
				Expect(downloader.IsOVACurrent()).To(BeFalse())
			})
		})

		Context("when downloader returns an error", func() {
			It("should return an error", func() {
				mockOVADownloader.EXPECT().IsOVACurrent().Return(false, errors.New("some-error"))

				_, err := downloader.IsOVACurrent()
				Expect(err).To(MatchError("some-error"))
			})
		})
	})

	Describe("#Download", func() {
		It("should download the file", func() {
			gomock.InOrder(
				mockOVADownloader.EXPECT().Setup(),
				mockOVADownloader.EXPECT().Download().Return("some-md5", nil),
				mockFS.EXPECT().Move("some-partial-ova-path", "some-ova-path"),
			)

			Expect(downloader.Download()).To(Succeed())
		})

		Context("when the download setup fails", func() {
			It("should return the error", func() {
				gomock.InOrder(
					mockOVADownloader.EXPECT().Setup().Return(errors.New("some-error")),
				)

				Expect(downloader.Download()).To(MatchError("some-error"))
			})
		})

		Context("when the download fails", func() {
			It("should return the error", func() {
				gomock.InOrder(
					mockOVADownloader.EXPECT().Setup(),
					mockOVADownloader.EXPECT().Download().Return("", errors.New("some-error")),
				)

				Expect(downloader.Download()).To(MatchError("some-error"))
			})
		})

		Context("when the downloaded MD5 does not match", func() {
			It("should delete the partially downloaded file and download again", func() {
				gomock.InOrder(
					mockOVADownloader.EXPECT().Setup(),
					mockOVADownloader.EXPECT().Download().Return("some-other-md5", nil),
					mockFS.EXPECT().Remove("some-partial-ova-path"),
					mockOVADownloader.EXPECT().Download().Return("some-md5", nil),
					mockFS.EXPECT().Move("some-partial-ova-path", "some-ova-path"),
				)

				Expect(downloader.Download()).To(Succeed())
			})
		})

		Context("when removing the ova of a corrupted download fails", func() {
			It("return the error", func() {
				gomock.InOrder(
					mockOVADownloader.EXPECT().Setup(),
					mockOVADownloader.EXPECT().Download().Return("some-other-md5", nil),
					mockFS.EXPECT().Remove("some-partial-ova-path").Return(errors.New("some-error")),
				)

				Expect(downloader.Download()).To(MatchError("some-error"))
			})
		})

		Context("when the downloaded MD5 does not match of the redownloaded OVA", func() {
			It("should delete the partially downloaded file and download again", func() {
				gomock.InOrder(
					mockOVADownloader.EXPECT().Setup(),
					mockOVADownloader.EXPECT().Download().Return("some-other-md5", nil),
					mockFS.EXPECT().Remove("some-partial-ova-path"),
					mockOVADownloader.EXPECT().Download().Return("some-other-bad-md5", nil),
				)

				Expect(downloader.Download()).To(MatchError("download failed"))
			})
		})

		Context("when there is an error download the ova a second time", func() {
			It("should return the error", func() {
				gomock.InOrder(
					mockOVADownloader.EXPECT().Setup(),
					mockOVADownloader.EXPECT().Download().Return("some-other-md5", nil),
					mockFS.EXPECT().Remove("some-partial-ova-path"),
					mockOVADownloader.EXPECT().Download().Return("", errors.New("some-error")),
				)

				Expect(downloader.Download()).To(MatchError("some-error"))
			})
		})

		Context("when there is an error moving the partial ova to the ova path", func() {
			It("should return the error", func() {
				gomock.InOrder(
					mockOVADownloader.EXPECT().Setup(),
					mockOVADownloader.EXPECT().Download().Return("some-md5", nil),
					mockFS.EXPECT().Move("some-partial-ova-path", "some-ova-path").Return(errors.New("some-error")),
				)

				Expect(downloader.Download()).To(MatchError("some-error"))
			})
		})
	})
})

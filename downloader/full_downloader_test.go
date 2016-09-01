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

var _ = Describe("FullDownloader", func() {
	var (
		downloader        *dl.FullDownloader
		mockCtrl          *gomock.Controller
		mockOVADownloader *mocks.MockOVADownloader
		mockFS            *mocks.MockFS
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockOVADownloader = mocks.NewMockOVADownloader(mockCtrl)
		mockFS = mocks.NewMockFS(mockCtrl)
		downloader = &dl.FullDownloader{
			Downloader: mockOVADownloader,
			FS:         mockFS,
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

		Context("when setup fails", func() {
			It("should return an error", func() {
				mockOVADownloader.EXPECT().Setup().Return(errors.New("some-error"))

				Expect(downloader.Download()).To(MatchError("some-error"))
			})
		})

		Context("when the downloaded MD5 does not match", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockOVADownloader.EXPECT().Setup(),
					mockOVADownloader.EXPECT().Download().Return("some-other-md5", nil),
				)

				Expect(downloader.Download()).To(MatchError("download failed"))
			})
		})

		Context("when downloading the ova fails", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockOVADownloader.EXPECT().Setup(),
					mockOVADownloader.EXPECT().Download().Return("", errors.New("some-error")),
				)

				Expect(downloader.Download()).To(MatchError("some-error"))
			})
		})

		Context("when moving the partial file fails", func() {
			It("should return an error", func() {
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

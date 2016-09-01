package downloader_test

import (
	"errors"

	"github.com/golang/mock/gomock"
	cfg "github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/downloader"
	"github.com/pivotal-cf/pcfdev-cli/downloader/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DownloaderFactory", func() {
	var (
		factory  *downloader.DownloaderFactory
		mockCtrl *gomock.Controller
		mockFS   *mocks.MockFS
		config   *cfg.Config
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockFS = mocks.NewMockFS(mockCtrl)
		config = &cfg.Config{
			PartialOVAPath: "some-partial-ova-path",
		}

		factory = &downloader.DownloaderFactory{
			FS:     mockFS,
			Config: config,
		}

	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("#Create", func() {
		Context("when there is a partial ova present", func() {
			It("should return a partial ova downloader", func() {
				mockFS.EXPECT().Exists("some-partial-ova-path").Return(true, nil)

				partialDownloader, err := factory.Create()
				Expect(err).NotTo(HaveOccurred())

				switch d := partialDownloader.(type) {
				case *downloader.PartialDownloader:
					Expect(d.FS).To(Equal(mockFS))
					Expect(d.Config).To(Equal(config))
					Expect(d.Downloader).NotTo(BeNil())
				default:
					Fail("wrong type")
				}
			})
		})

		Context("when there is no partial ova present", func() {
			It("should return a full ova downloader", func() {
				mockFS.EXPECT().Exists("some-partial-ova-path").Return(false, nil)

				fullDownloader, err := factory.Create()
				Expect(err).NotTo(HaveOccurred())

				switch d := fullDownloader.(type) {
				case *downloader.FullDownloader:
					Expect(d.FS).To(Equal(mockFS))
					Expect(d.Config).To(Equal(config))
					Expect(d.Downloader).NotTo(BeNil())
				default:
					Fail("wrong type")
				}
			})
		})

		Context("when there is an error seeing if there is a partial ova present", func() {
			It("should return the error", func() {
				mockFS.EXPECT().Exists("some-partial-ova-path").Return(false, errors.New("some-error"))

				partialDownloader, err := factory.Create()
				Expect(err).To(MatchError("some-error"))
				Expect(partialDownloader).To(BeNil())
			})
		})
	})
})

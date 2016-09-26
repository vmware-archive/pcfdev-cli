package cert_test

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/pcfdev-cli/cert"
	"github.com/pivotal-cf/pcfdev-cli/cert/mocks"
)

var _ = Describe("CertStore", func() {
	var (
		certStore       *cert.CertStore
		mockCtrl        *gomock.Controller
		mockFS          *mocks.MockFS
		mockSystemStore *mocks.MockSystemStore
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockFS = mocks.NewMockFS(mockCtrl)
		mockSystemStore = mocks.NewMockSystemStore(mockCtrl)
		certStore = &cert.CertStore{
			FS:          mockFS,
			SystemStore: mockSystemStore,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("#Store", func() {
		It("should add the certificate to the system certificate store", func() {
			gomock.InOrder(
				mockFS.EXPECT().TempDir().Return("some-temp-dir", nil),
				mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "cert"), strings.NewReader("some-cert"), false),
				mockSystemStore.EXPECT().Store(filepath.Join("some-temp-dir", "cert")),
				mockFS.EXPECT().Remove("some-temp-dir"),
			)

			Expect(certStore.Store("some-cert")).To(Succeed())
		})

		Context("when there is an error creating a temp dir", func() {
			It("should return the error", func() {
				mockFS.EXPECT().TempDir().Return("", errors.New("some-error"))

				Expect(certStore.Store("some-cert")).To(MatchError("some-error"))
			})
		})

		Context("when there is an error writing the cert file", func() {
			It("should return the error", func() {
				gomock.InOrder(
					mockFS.EXPECT().TempDir().Return("some-temp-dir", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "cert"), strings.NewReader("some-cert"), false).Return(errors.New("some-error")),
					mockFS.EXPECT().Remove("some-temp-dir"),
				)

				Expect(certStore.Store("some-cert")).To(MatchError("some-error"))
			})
		})

		Context("when there is an error storing the cert", func() {
			It("should return the error", func() {
				gomock.InOrder(
					mockFS.EXPECT().TempDir().Return("some-temp-dir", nil),
					mockFS.EXPECT().Write(filepath.Join("some-temp-dir", "cert"), strings.NewReader("some-cert"), false),
					mockSystemStore.EXPECT().Store(filepath.Join("some-temp-dir", "cert")).Return(errors.New("some-error")),
					mockFS.EXPECT().Remove("some-temp-dir"),
				)

				Expect(certStore.Store("some-cert")).To(MatchError("some-error"))
			})
		})
	})

	Describe("#Unstore", func() {
		It("should remove the certificates from the system certificate store", func() {
			mockSystemStore.EXPECT().Unstore()

			Expect(certStore.Unstore()).To(Succeed())
		})

		Context("when there is an issue removing the certificates from the system certificate store", func() {
			It("should return the error", func() {
				mockSystemStore.EXPECT().Unstore().Return(errors.New("some-error"))

				Expect(certStore.Unstore()).To(MatchError("some-error"))
			})
		})
	})
})

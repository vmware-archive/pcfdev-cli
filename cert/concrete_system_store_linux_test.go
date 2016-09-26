package cert_test

import (
	"bytes"
	"errors"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/pcfdev-cli/cert"
	"github.com/pivotal-cf/pcfdev-cli/cert/mocks"
)

var _ = Describe("ConcreteSystemStore", func() {
	var (
		certStore *cert.ConcreteSystemStore
		mockCtrl  *gomock.Controller
		mockFS    *mocks.MockFS
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockFS = mocks.NewMockFS(mockCtrl)
		certStore = &cert.ConcreteSystemStore{
			FS: mockFS,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("#Store", func() {
		Context("when OS is Ubuntu", func() {
			It("should add the certificate to the system certificate store", func() {
				gomock.InOrder(
					mockFS.EXPECT().Exists("/etc/ssl/certs/ca-certificates.crt").Return(true, nil),
					mockFS.EXPECT().Read("path-to-some-cert").Return([]byte("some-cert"), nil),
					mockFS.EXPECT().Write("/etc/ssl/certs/ca-certificates.crt", bytes.NewReader([]byte("some-cert")), true),
				)

				Expect(certStore.Store("path-to-some-cert")).To(Succeed())
			})
		})

		Context("when OS is Fedora/RHEL", func() {
			It("should add the certificate to the system certificate store", func() {
				gomock.InOrder(
					mockFS.EXPECT().Exists("/etc/ssl/certs/ca-certificates.crt").Return(false, nil),
					mockFS.EXPECT().Exists("/etc/pki/tls/certs/ca-bundle.crt").Return(true, nil),
					mockFS.EXPECT().Read("path-to-some-cert").Return([]byte("some-cert"), nil),
					mockFS.EXPECT().Write("/etc/pki/tls/certs/ca-bundle.crt", bytes.NewReader([]byte("some-cert")), true),
				)

				Expect(certStore.Store("path-to-some-cert")).To(Succeed())
			})
		})

		Context("when OS is OpenSUSE", func() {
			It("should add the certificate to the system certificate store", func() {
				gomock.InOrder(
					mockFS.EXPECT().Exists("/etc/ssl/certs/ca-certificates.crt").Return(false, nil),
					mockFS.EXPECT().Exists("/etc/pki/tls/certs/ca-bundle.crt").Return(false, nil),
					mockFS.EXPECT().Exists("/etc/ssl/ca-bundle.pem").Return(true, nil),
					mockFS.EXPECT().Read("path-to-some-cert").Return([]byte("some-cert"), nil),
					mockFS.EXPECT().Write("/etc/ssl/ca-bundle.pem", bytes.NewReader([]byte("some-cert")), true),
				)

				Expect(certStore.Store("path-to-some-cert")).To(Succeed())
			})
		})

		Context("when OS is OpenELEC", func() {
			It("should add the certificate to the system certificate store", func() {
				gomock.InOrder(
					mockFS.EXPECT().Exists("/etc/ssl/certs/ca-certificates.crt").Return(false, nil),
					mockFS.EXPECT().Exists("/etc/pki/tls/certs/ca-bundle.crt").Return(false, nil),
					mockFS.EXPECT().Exists("/etc/ssl/ca-bundle.pem").Return(false, nil),
					mockFS.EXPECT().Exists("/etc/pki/tls/cacert.pem").Return(true, nil),
					mockFS.EXPECT().Read("path-to-some-cert").Return([]byte("some-cert"), nil),
					mockFS.EXPECT().Write("/etc/pki/tls/cacert.pem", bytes.NewReader([]byte("some-cert")), true),
				)

				Expect(certStore.Store("path-to-some-cert")).To(Succeed())
			})
		})

		Context("when OS is Unexpected", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockFS.EXPECT().Exists("/etc/ssl/certs/ca-certificates.crt").Return(false, nil),
					mockFS.EXPECT().Exists("/etc/pki/tls/certs/ca-bundle.crt").Return(false, nil),
					mockFS.EXPECT().Exists("/etc/ssl/ca-bundle.pem").Return(false, nil),
					mockFS.EXPECT().Exists("/etc/pki/tls/cacert.pem").Return(false, nil),
				)

				Expect(certStore.Store("path-to-some-cert")).To(MatchError("failed to determine path to CA Cert Store"))
			})
		})

		Context("when there is an error checking if a file exists", func() {
			It("should return an error", func() {
				mockFS.EXPECT().Exists("/etc/ssl/certs/ca-certificates.crt").Return(false, errors.New("some-error"))

				Expect(certStore.Store("path-to-some-cert")).To(MatchError("some-error"))
			})
		})

		Context("when there is an error reading the cert file", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockFS.EXPECT().Exists("/etc/ssl/certs/ca-certificates.crt").Return(true, nil),
					mockFS.EXPECT().Read("path-to-some-cert").Return(nil, errors.New("some-error")),
				)

				Expect(certStore.Store("path-to-some-cert")).To(MatchError("some-error"))
			})
		})

		Context("when there is an error writing to the CA cert store", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockFS.EXPECT().Exists("/etc/ssl/certs/ca-certificates.crt").Return(true, nil),
					mockFS.EXPECT().Read("path-to-some-cert").Return([]byte("some-cert"), nil),
					mockFS.EXPECT().Write("/etc/ssl/certs/ca-certificates.crt", bytes.NewReader([]byte("some-cert")), true).Return(errors.New("some-error")),
				)

				Expect(certStore.Store("path-to-some-cert")).To(MatchError("some-error"))
			})
		})
	})
})

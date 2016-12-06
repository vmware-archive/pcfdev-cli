package cert_test

import (
	"errors"

	"fmt"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/pcfdev-cli/cert"
	"github.com/pivotal-cf/pcfdev-cli/cert/mocks"
	"path/filepath"
	"strings"
)

var _ = Describe("ConcreteSystemStore", func() {
	var (
		certStore     *cert.ConcreteSystemStore
		mockCtrl      *gomock.Controller
		mockCmdRunner *mocks.MockCmdRunner
		mockFS        *mocks.MockFS
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockCmdRunner = mocks.NewMockCmdRunner(mockCtrl)
		mockFS = mocks.NewMockFS(mockCtrl)
		certStore = &cert.ConcreteSystemStore{
			CmdRunner: mockCmdRunner,
			FS:        mockFS,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("#Store", func() {
		It("should create a keychain, add the cert to it, and display the keychain", func() {
			gomock.InOrder(
				mockCmdRunner.EXPECT().Run("security", "create-keychain", "-p", "pcfdev", "pcfdev.keychain"),
				mockCmdRunner.EXPECT().Run("security", "list-keychains", "-d", "user", "-s", "login.keychain", "pcfdev.keychain"),
				mockCmdRunner.EXPECT().Run("security", "add-trusted-cert", "-d", "-r", "trustRoot", "-k", gomock.Any(), "some-path"),
			)

			Expect(certStore.Store("some-path")).To(Succeed())
		})

		Context("when there is an error loading the keychain", func() {
			It("should return the error", func() {
				gomock.InOrder(
					mockCmdRunner.EXPECT().Run("security", "create-keychain", "-p", "pcfdev", "pcfdev.keychain"),
					mockCmdRunner.EXPECT().Run("security", "list-keychains", "-d", "user", "-s", "login.keychain", "pcfdev.keychain").Return(nil, errors.New("some-error")),
				)

				Expect(certStore.Store("some-path")).To(MatchError("some-error"))
			})
		})

		Context("when there is an error adding the trusted cert", func() {
			It("should return the error", func() {
				gomock.InOrder(
					mockCmdRunner.EXPECT().Run("security", "create-keychain", "-p", "pcfdev", "pcfdev.keychain"),
					mockCmdRunner.EXPECT().Run("security", "list-keychains", "-d", "user", "-s", "login.keychain", "pcfdev.keychain"),
					mockCmdRunner.EXPECT().Run("security", "add-trusted-cert", "-d", "-r", "trustRoot", "-k", gomock.Any(), "some-path").Return(nil, errors.New("some-error")),
				)

				Expect(certStore.Store("some-path")).To(MatchError("some-error"))
			})
		})
	})

	Describe("#Unstore", func() {
		allowHappyPathInteractions := func() {
			mockFS.EXPECT().TempDir().Return("some-dir", nil).AnyTimes()
			var anys []interface{}
			for i := 0; i <= 10; i++ {
				mockCmdRunner.EXPECT().Run(gomock.Any(), anys...).AnyTimes()
				anys = append(anys, gomock.Any())
			}
			mockFS.EXPECT().Remove(gomock.Any()).AnyTimes()
		}

		It("should remove the certificate and delete the keychain", func() {
			certsPath := filepath.Join("some-dir", "certs.pem")
			gomock.InOrder(
				mockCmdRunner.EXPECT().Run("security", "showkeychaininfo", endsWith("pcfdev.keychain")),
				mockFS.EXPECT().TempDir().Return("some-dir", nil),
				mockCmdRunner.EXPECT().Run("security", "export", "-k", endsWith("pcfdev.keychain"), "-p", "-o", certsPath),
				mockCmdRunner.EXPECT().Run("security", "remove-trusted-cert", "-d", certsPath),
				mockCmdRunner.EXPECT().Run("security", "delete-keychain", "pcfdev.keychain"),
				mockFS.EXPECT().Remove("some-dir"),
			)

			Expect(certStore.Unstore()).To(Succeed())
		})

		Context("when the pcfdev keychain does not exist", func() {
			It("should not return an error", func() {
				mockCmdRunner.EXPECT().Run("security", "showkeychaininfo", endsWith("pcfdev.keychain")).Return(nil, errors.New("some error"))
				Expect(certStore.Unstore()).To(Succeed())
			})
		})

		Context("when there is an error making a temp dir", func() {
			It("should return the error", func() {
				mockFS.EXPECT().TempDir().Return("", errors.New("some-error"))
				allowHappyPathInteractions()
				Expect(certStore.Unstore()).To(MatchError("some-error"))
			})
		})

		Context("when there is an error exporting the certificate", func() {
			It("should return the error", func() {
				mockCmdRunner.EXPECT().Run("security", "export", "-k", gomock.Any(), "-p", "-o", gomock.Any()).Return(nil, errors.New("some-error"))
				allowHappyPathInteractions()
				Expect(certStore.Unstore()).To(MatchError("some-error"))
			})
		})

		Context("when there is an error removing the certificate", func() {
			It("should return the error", func() {
				mockCmdRunner.EXPECT().Run("security", "remove-trusted-cert", "-d", gomock.Any()).Return(nil, errors.New("some-error"))
				allowHappyPathInteractions()
				Expect(certStore.Unstore()).To(MatchError("some-error"))
			})
		})

		Context("when there is an error deleting the keychain", func() {
			It("should return the error", func() {
				mockCmdRunner.EXPECT().Run("security", "delete-keychain", "pcfdev.keychain").Return(nil, errors.New("some-error"))
				allowHappyPathInteractions()
				Expect(certStore.Unstore()).To(MatchError("some-error"))
			})
		})
	})
})

func endsWith(expected string) *endsWithMatcher {
	return &endsWithMatcher{
		ExpectedSuffix: expected,
	}
}

type endsWithMatcher struct {
	ExpectedSuffix string
	actual         string
}

func (k *endsWithMatcher) Matches(x interface{}) bool {
	var isAString bool
	k.actual, isAString = x.(string)
	return isAString && strings.HasSuffix(k.actual, k.ExpectedSuffix)
}

func (k *endsWithMatcher) String() string {
	return fmt.Sprintf(`Expected "%s" to end with "%s"`, k.actual, k.ExpectedSuffix)
}

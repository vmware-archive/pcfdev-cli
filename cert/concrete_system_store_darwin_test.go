package cert_test

import (
	"errors"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/pcfdev-cli/cert"
	"github.com/pivotal-cf/pcfdev-cli/cert/mocks"
)

var _ = Describe("ConcreteSystemStore", func() {
	var (
		certStore     *cert.ConcreteSystemStore
		mockCtrl      *gomock.Controller
		mockCmdRunner *mocks.MockCmdRunner
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockCmdRunner = mocks.NewMockCmdRunner(mockCtrl)
		certStore = &cert.ConcreteSystemStore{
			CmdRunner: mockCmdRunner,
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
})

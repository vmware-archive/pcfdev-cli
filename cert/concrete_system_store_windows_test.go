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
		It("should store PCF Dev certificates", func() {
			mockCmdRunner.EXPECT().Run("certutil", "-addstore", "-f", "ROOT", "some-path")

			Expect(certStore.Store("some-path")).To(Succeed())
		})

		Context("when there is an issue storing the certificates", func() {
			It("should return an error", func() {
				mockCmdRunner.EXPECT().Run("certutil", "-addstore", "-f", "ROOT", "some-path").Return(nil, errors.New("some-error"))

				Expect(certStore.Store("some-path")).To(MatchError("some-error"))
			})
		})
	})

	Describe("#Unstore", func() {
		It("should delete only PCF Dev certificates", func() {
			mockCmdRunner.EXPECT().Run("certutil", "-delstore", "ROOT", "local.pcfdev.io")
			mockCmdRunner.EXPECT().Run("certutil", "-delstore", "ROOT", "local2.pcfdev.io")
			mockCmdRunner.EXPECT().Run("certutil", "-delstore", "ROOT", "local3.pcfdev.io")
			mockCmdRunner.EXPECT().Run("certutil", "-delstore", "ROOT", "local4.pcfdev.io")
			mockCmdRunner.EXPECT().Run("certutil", "-delstore", "ROOT", "local5.pcfdev.io")
			mockCmdRunner.EXPECT().Run("certutil", "-delstore", "ROOT", "local6.pcfdev.io")
			mockCmdRunner.EXPECT().Run("certutil", "-delstore", "ROOT", "local7.pcfdev.io")
			mockCmdRunner.EXPECT().Run("certutil", "-delstore", "ROOT", "local8.pcfdev.io")
			mockCmdRunner.EXPECT().Run("certutil", "-delstore", "ROOT", "local9.pcfdev.io")

			Expect(certStore.Unstore()).To(Succeed())
		})
	})
})

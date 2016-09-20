package cert_test

import (
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

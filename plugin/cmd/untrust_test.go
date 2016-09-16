package cmd_test

import (
	"errors"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/pcfdev-cli/plugin/cmd"
	"github.com/pivotal-cf/pcfdev-cli/plugin/cmd/mocks"
)

var _ = Describe("UntrustCmd", func() {
	var (
		untrustCmd    *cmd.UntrustCmd
		mockCtrl      *gomock.Controller
		mockCertStore *mocks.MockCertStore
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockCertStore = mocks.NewMockCertStore(mockCtrl)
		untrustCmd = &cmd.UntrustCmd{
			CertStore: mockCertStore,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("Parse", func() {
		Context("when the correct number of arguments are passed", func() {
			It("should succeed", func() {
				Expect(untrustCmd.Parse([]string{})).To(Succeed())
			})
		})
		Context("when the wrong number of arguments are passed", func() {
			It("should fail", func() {
				Expect(untrustCmd.Parse([]string{"some-bad-arg"})).NotTo(Succeed())
			})
		})
		Context("when an unknown flag is passed", func() {
			It("should fail", func() {
				Expect(untrustCmd.Parse([]string{"--some-bad-flag"})).NotTo(Succeed())
			})
		})
	})

	Describe("Run", func() {
		It("should call 'Unstore' on the CertStore", func() {
			mockCertStore.EXPECT().Unstore()

			Expect(untrustCmd.Run()).To(Succeed())
		})

		Context("when there is an error running 'Unstore' on the CertStore", func() {
			It("should return the error", func() {
				mockCertStore.EXPECT().Unstore().Return(errors.New("some-error"))

				Expect(untrustCmd.Run()).To(MatchError("some-error"))
			})
		})
	})
})

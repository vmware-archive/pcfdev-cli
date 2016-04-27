package config_test

import (
	"os"

	"github.com/golang/mock/gomock"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/config/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	Context("#GetToken", func() {
		Context("when PIVNET_TOKEN env var is set", func() {
			var savedToken string

			BeforeEach(func() {
				savedToken = os.Getenv("PIVNET_TOKEN")
				os.Setenv("PIVNET_TOKEN", "some-token")
			})

			AfterEach(func() {
				os.Setenv("PIVNET_TOKEN", savedToken)
			})

			It("should return PIVNET_TOKEN env var", func() {
				config := &config.Config{}
				Expect(config.GetToken()).To(Equal("some-token"))
			})
		})

		Context("when PIVNET_TOKEN env var is not set", func() {
			var (
				savedToken string
				mockUI     *mocks.MockUI
				mockCtrl   *gomock.Controller
				cfg        *config.Config
			)

			BeforeEach(func() {
				savedToken = os.Getenv("PIVNET_TOKEN")
				os.Setenv("PIVNET_TOKEN", "")
				mockCtrl = gomock.NewController(GinkgoT())
				mockUI = mocks.NewMockUI(mockCtrl)

				cfg = &config.Config{
					UI: mockUI,
				}
			})

			AfterEach(func() {
				os.Setenv("PIVNET_TOKEN", savedToken)
			})

			It("should prompt the user to enter their PivNet token", func() {
				mockUI.EXPECT().AskForPassword("Enter your Pivotal Network API token:").Return("some-user-provided-token")
				Expect(cfg.GetToken()).To(Equal("some-user-provided-token"))
			})

		})
	})
})

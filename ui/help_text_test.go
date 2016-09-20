package ui_test

import (
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/pivotal-cf/pcfdev-cli/ui"
	"github.com/pivotal-cf/pcfdev-cli/ui/mocks"
)

var _ = Describe("HelpText", func() {
	var (
		mockCtrl *gomock.Controller
		mockUI   *mocks.MockPluginUI
		helpText *ui.HelpText
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockUI = mocks.NewMockPluginUI(mockCtrl)
		helpText = &ui.HelpText{
			UI: mockUI,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("Print", func() {
		Context("when autoTargeting has not occurred", func() {
			It("should succeed", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say(` _______  _______  _______    ______   _______  __   __
|       ||       ||       |  |      | |       ||  | |  |
|    _  ||       ||    ___|  |  _    ||    ___||  |_|  |
|   |_| ||       ||   |___   | | |   ||   |___ |       |
|    ___||      _||    ___|  | |_|   ||    ___||       |
|   |    |     |_ |   |      |       ||   |___  |     |
|___|    |_______||___|      |______| |_______|  |___|
is now running.`),
					mockUI.EXPECT().Say(`To begin using PCF Dev, please run:`),
					mockUI.EXPECT().Say(`   cf login -a https://api.some-domain --skip-ssl-validation
Apps Manager URL: https://some-domain
Admin user => Email: admin / Password: admin
Regular user => Email: user / Password: pass`),
				)

				helpText.Print("some-domain", false)
			})
		})

		Context("when autoTargeting has occurred", func() {
			It("should succeed", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say(` _______  _______  _______    ______   _______  __   __
|       ||       ||       |  |      | |       ||  | |  |
|    _  ||       ||    ___|  |  _    ||    ___||  |_|  |
|   |_| ||       ||   |___   | | |   ||   |___ |       |
|    ___||      _||    ___|  | |_|   ||    ___||       |
|   |    |     |_ |   |      |       ||   |___  |     |
|___|    |_______||___|      |______| |_______|  |___|
is now running.`),
					mockUI.EXPECT().Say(`PCF Dev automatically targeted. To target manually, run:`),
					mockUI.EXPECT().Say(`   cf login -a https://api.some-domain --skip-ssl-validation
Apps Manager URL: https://some-domain
Admin user => Email: admin / Password: admin
Regular user => Email: user / Password: pass`),
				)

				helpText.Print("some-domain", true)
			})
		})
	})
})

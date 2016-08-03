package cmd_test

import (
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/plugin/cmd"
	"github.com/pivotal-cf/pcfdev-cli/plugin/cmd/mocks"
)

var _ = Describe("VersionCmd", func() {
	var (
		versionCmd *cmd.VersionCmd
		mockUI     *mocks.MockUI
		mockCtrl   *gomock.Controller
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockUI = mocks.NewMockUI(mockCtrl)
		versionCmd = &cmd.VersionCmd{
			Config: &config.Config{
				Version: &config.Version{
					BuildVersion:    "some-build-version",
					BuildSHA:        "some-sha",
					OVABuildVersion: "some-ova-version",
				},
			},
			UI: mockUI,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("Parse", func() {
		Context("when the correct number of arguments are passed", func() {
			It("should succeed", func() {
				Expect(versionCmd.Parse([]string{})).To(Succeed())
			})
		})

		Context("when the wrong number of arguments are passed", func() {
			It("should fail", func() {
				Expect(versionCmd.Parse([]string{"some-bad-arg"})).NotTo(Succeed())
				Expect(versionCmd.Parse([]string{"--some-bad-flag"})).NotTo(Succeed())
			})
		})
	})

	Describe("Run", func() {
		It("should print out the versions", func() {
			mockUI.EXPECT().Say("PCF Dev version some-build-version (CLI: some-sha, OVA: some-ova-version)")
			Expect(versionCmd.Run()).To(Succeed())
		})
	})
})

package cmd_test

import (
	"errors"
	"path/filepath"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/plugin/cmd"
	"github.com/pivotal-cf/pcfdev-cli/plugin/cmd/mocks"
)

var _ = Describe("ImportCmd", func() {
	var (
		mockFS         *mocks.MockFS
		mockDownloader *mocks.MockDownloader
		mockUI         *mocks.MockUI
		mockCtrl       *gomock.Controller
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockFS = mocks.NewMockFS(mockCtrl)
		mockUI = mocks.NewMockUI(mockCtrl)
		mockDownloader = mocks.NewMockDownloader(mockCtrl)
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("Parse", func() {
		Context("when the correct number of arguments are passed", func() {
			It("should succeed", func() {
				importCommand := &cmd.ImportCmd{}
				Expect(importCommand.Parse([]string{"some-ova"})).To(Succeed())
				Expect(importCommand.OVAPath).To(Equal("some-ova"))
			})
		})

		Context("when the wrong number of arguments are passed", func() {
			It("should fail", func() {
				importCommand := &cmd.ImportCmd{}
				Expect(importCommand.Parse([]string{})).NotTo(Succeed())
			})
		})

		Context("when an unknown flag is passed", func() {
			It("should fail", func() {
				importCommand := &cmd.ImportCmd{}
				Expect(importCommand.Parse([]string{"some-ova", "--some-bad-flag"})).NotTo(Succeed())
			})
		})
	})

	Describe("Run", func() {
		var importCmd *cmd.ImportCmd

		BeforeEach(func() {
			importCmd = &cmd.ImportCmd{
				OVAPath:    "some-ova-path",
				UI:         mockUI,
				FS:         mockFS,
				Downloader: mockDownloader,
				Config: &config.Config{
					DefaultVMName: "some-vm-name",
					OVADir:        "some-ova-dir",
					ExpectedMD5:   "some-md5",
					Version: &config.Version{
						BuildVersion:    "some-build-version",
						BuildSHA:        "some-build-sha",
						OVABuildVersion: "some-ova-version",
					},
				},
			}
		})

		It("should copy an ova to the specified path", func() {
			gomock.InOrder(
				mockFS.EXPECT().MD5("some-ova-path").Return("some-md5", nil),
				mockDownloader.EXPECT().IsOVACurrent().Return(false, nil),
				mockFS.EXPECT().Copy("some-ova-path", filepath.Join("some-ova-dir", "some-vm-name.ova")),
				mockUI.EXPECT().Say("OVA version some-ova-version imported successfully."),
			)

			Expect(importCmd.Run()).To(Succeed())
		})

		Context("when move returns an error", func() {
			It("should print an error message", func() {
				gomock.InOrder(
					mockFS.EXPECT().MD5("some-ova-path").Return("some-md5", nil),
					mockDownloader.EXPECT().IsOVACurrent().Return(false, nil),
					mockFS.EXPECT().Copy("some-ova-path", filepath.Join("some-ova-dir", "some-vm-name.ova")).Return(errors.New("some-error")),
				)
				Expect(importCmd.Run()).To(MatchError("some-error"))
			})
		})

		Context("when the ova is not the correct ova for the plugin", func() {
			It("should print an error message", func() {
				mockFS.EXPECT().MD5("some-ova-path").Return("some-bad-md5", nil)

				Expect(importCmd.Run()).To(MatchError("specified OVA version does not match the expected OVA version (some-ova-version) for this version of the cf CLI plugin"))
			})
		})

		Context("when the checksum returns an error", func() {
			It("should print an error message", func() {
				mockFS.EXPECT().MD5("some-ova-path").Return("some-bad-md5", errors.New("some-error"))

				Expect(importCmd.Run()).To(MatchError("some-error"))
			})
		})

		Context("when the ova is already installed", func() {
			It("should print a message", func() {
				gomock.InOrder(
					mockFS.EXPECT().MD5("some-ova-path").Return("some-md5", nil),
					mockDownloader.EXPECT().IsOVACurrent().Return(true, nil),

					mockUI.EXPECT().Say("PCF Dev OVA is already installed."),
				)

				Expect(importCmd.Run()).To(Succeed())
			})
		})

		Context("when there is an error checking if the ova is current", func() {
			It("should print an error message", func() {
				gomock.InOrder(
					mockFS.EXPECT().MD5("some-ova-path").Return("some-md5", nil),
					mockDownloader.EXPECT().IsOVACurrent().Return(true, errors.New("some-error")),
				)

				Expect(importCmd.Run()).To(MatchError("some-error"))
			})
		})
	})
})

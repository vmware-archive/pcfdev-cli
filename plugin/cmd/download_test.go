package cmd_test

import (
	"errors"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/plugin/cmd"
	"github.com/pivotal-cf/pcfdev-cli/plugin/cmd/mocks"
)

var _ = Describe("DownloadCmd", func() {
	var (
		mockCtrl       *gomock.Controller
		mockUI         *mocks.MockUI
		mockEULAUI     *mocks.MockEULAUI
		mockVBox       *mocks.MockVBox
		mockFS         *mocks.MockFS
		mockDownloader *mocks.MockDownloader
		mockClient     *mocks.MockClient
		downloadCmd    *cmd.DownloadCmd
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockUI = mocks.NewMockUI(mockCtrl)
		mockEULAUI = mocks.NewMockEULAUI(mockCtrl)
		mockClient = mocks.NewMockClient(mockCtrl)
		mockVBox = mocks.NewMockVBox(mockCtrl)
		mockDownloader = mocks.NewMockDownloader(mockCtrl)
		mockFS = mocks.NewMockFS(mockCtrl)
		downloadCmd = &cmd.DownloadCmd{
			UI:         mockUI,
			EULAUI:     mockEULAUI,
			Client:     mockClient,
			VBox:       mockVBox,
			Downloader: mockDownloader,
			Config: &config.Config{
				DefaultVMName: "some-vm-name",
			},
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("Parse", func() {
		Context("when the correct number of arguments are passed", func() {
			It("should succeed", func() {
				downloadCommand := &cmd.DownloadCmd{}
				Expect(downloadCommand.Parse([]string{})).To(Succeed())
			})
		})
		Context("when the wrong number of arguments are passed", func() {
			It("should fail", func() {
				downloadCommand := &cmd.DownloadCmd{}
				Expect(downloadCommand.Parse([]string{"some-bad-arg"})).NotTo(Succeed())
			})
		})
		Context("when an unknown flag is passed", func() {
			It("should fail", func() {
				downloadCommand := &cmd.DownloadCmd{}
				Expect(downloadCommand.Parse([]string{"--some-bad-flag"})).NotTo(Succeed())
			})
		})
	})

	Describe("Run", func() {
		Context("when OVA is current", func() {
			It("should not download", func() {
				gomock.InOrder(
					mockVBox.EXPECT().GetVMName().Return("", nil),
					mockDownloader.EXPECT().IsOVACurrent().Return(true, nil),
					mockUI.EXPECT().Say("Using existing image."),
				)

				downloadCmd.Run()
			})
		})

		Context("when there is an old vm present", func() {
			It("should tell the user to destroy downloadCmd", func() {
				mockVBox.EXPECT().GetVMName().Return("some-old-downloadCmd-ova", nil)

				Expect(downloadCmd.Run()).To(MatchError("old version of PCF Dev already running, please run `cf dev destroy` to continue"))
			})
		})

		Context("when there is an error checking for an old vm present", func() {
			It("should return the error", func() {
				mockVBox.EXPECT().GetVMName().Return("", errors.New("some-error"))

				Expect(downloadCmd.Run()).To(MatchError("some-error"))
			})
		})

		Context("when calling IsOVACurrent fails", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockVBox.EXPECT().GetVMName().Return("", nil),
					mockDownloader.EXPECT().IsOVACurrent().Return(false, errors.New("some-error")),
				)

				Expect(downloadCmd.Run()).To(MatchError("some-error"))
			})
		})

		Context("when OVA is not current", func() {
			It("should download the OVA", func() {
				gomock.InOrder(
					mockVBox.EXPECT().GetVMName().Return("", nil),
					mockDownloader.EXPECT().IsOVACurrent().Return(false, nil),
					mockClient.EXPECT().IsEULAAccepted().Return(true, nil),
					mockUI.EXPECT().Say("Downloading VM..."),
					mockDownloader.EXPECT().Download(),
					mockUI.EXPECT().Say("\nVM downloaded."),
				)

				downloadCmd.Run()
			})

			Context("when EULA check fails", func() {
				It("should print an error", func() {
					gomock.InOrder(
						mockVBox.EXPECT().GetVMName().Return("", nil),
						mockDownloader.EXPECT().IsOVACurrent().Return(false, nil),
						mockClient.EXPECT().IsEULAAccepted().Return(false, errors.New("some-error")),
					)

					Expect(downloadCmd.Run()).To(MatchError("some-error"))

				})
			})

			Context("when downloading the OVA fails", func() {
				It("should print an error", func() {
					gomock.InOrder(
						mockVBox.EXPECT().GetVMName().Return("", nil),
						mockDownloader.EXPECT().IsOVACurrent().Return(false, nil),
						mockClient.EXPECT().IsEULAAccepted().Return(true, nil),
						mockUI.EXPECT().Say("Downloading VM..."),
						mockDownloader.EXPECT().Download().Return(errors.New("some-error")),
					)

					Expect(downloadCmd.Run()).To(MatchError("some-error"))
				})
			})

			Context("when EULA has not been accepted and user accepts the EULA", func() {
				It("should download the ova", func() {
					gomock.InOrder(
						mockVBox.EXPECT().GetVMName().Return("", nil),
						mockDownloader.EXPECT().IsOVACurrent().Return(false, nil),
						mockClient.EXPECT().IsEULAAccepted().Return(false, nil),
						mockClient.EXPECT().GetEULA().Return("some-eula", nil),
						mockEULAUI.EXPECT().Init(),
						mockEULAUI.EXPECT().ConfirmText("some-eula").Return(true),
						mockEULAUI.EXPECT().Close(),
						mockClient.EXPECT().AcceptEULA().Return(nil),
						mockUI.EXPECT().Say("Downloading VM..."),
						mockDownloader.EXPECT().Download(),
						mockUI.EXPECT().Say("\nVM downloaded."),
					)

					downloadCmd.Run()
				})
			})

			Context("when EULA has not been accepted and user denies the EULA", func() {
				It("should not accept and fail gracefully", func() {
					gomock.InOrder(
						mockVBox.EXPECT().GetVMName().Return("", nil),
						mockDownloader.EXPECT().IsOVACurrent().Return(false, nil),
						mockClient.EXPECT().IsEULAAccepted().Return(false, nil),
						mockClient.EXPECT().GetEULA().Return("some-eula", nil),
						mockEULAUI.EXPECT().Init(),
						mockEULAUI.EXPECT().ConfirmText("some-eula").Return(false),
						mockEULAUI.EXPECT().Close(),
					)

					Expect(downloadCmd.Run()).To(MatchError("you must accept the end user license agreement to use PCF Dev"))
				})
			})

			Context("when EULA has not been accepted and it fails to accept the EULA", func() {
				It("should return the error", func() {
					gomock.InOrder(
						mockVBox.EXPECT().GetVMName().Return("", nil),
						mockDownloader.EXPECT().IsOVACurrent().Return(false, nil),
						mockClient.EXPECT().IsEULAAccepted().Return(false, nil),
						mockClient.EXPECT().GetEULA().Return("some-eula", nil),
						mockEULAUI.EXPECT().Init(),
						mockEULAUI.EXPECT().ConfirmText("some-eula").Return(true),
						mockEULAUI.EXPECT().Close(),
						mockClient.EXPECT().AcceptEULA().Return(errors.New("some-error")),
					)

					Expect(downloadCmd.Run()).To(MatchError("some-error"))
				})
			})

			Context("when EULA fails to close after not being accepted", func() {
				It("should return the error", func() {
					gomock.InOrder(
						mockVBox.EXPECT().GetVMName().Return("", nil),
						mockDownloader.EXPECT().IsOVACurrent().Return(false, nil),
						mockClient.EXPECT().IsEULAAccepted().Return(false, nil),
						mockClient.EXPECT().GetEULA().Return("some-eula", nil),
						mockEULAUI.EXPECT().Init(),
						mockEULAUI.EXPECT().ConfirmText("some-eula").Return(false),
						mockEULAUI.EXPECT().Close().Return(errors.New("some-error")),
					)

					Expect(downloadCmd.Run()).To(MatchError("some-error"))
				})
			})

			Context("when EULA fails to close after being accepted", func() {
				It("should return the error", func() {
					gomock.InOrder(
						mockVBox.EXPECT().GetVMName().Return("", nil),
						mockDownloader.EXPECT().IsOVACurrent().Return(false, nil),
						mockClient.EXPECT().IsEULAAccepted().Return(false, nil),
						mockClient.EXPECT().GetEULA().Return("some-eula", nil),
						mockEULAUI.EXPECT().Init(),
						mockEULAUI.EXPECT().ConfirmText("some-eula").Return(true),
						mockEULAUI.EXPECT().Close().Return(errors.New("some-error")),
					)

					Expect(downloadCmd.Run()).To(MatchError("some-error"))
				})
			})

			Context("when EULA is not accepted and getting the EULA fails", func() {
				It("should print an error", func() {
					gomock.InOrder(
						mockVBox.EXPECT().GetVMName().Return("", nil),
						mockDownloader.EXPECT().IsOVACurrent().Return(false, nil),
						mockClient.EXPECT().IsEULAAccepted().Return(false, nil),
						mockClient.EXPECT().GetEULA().Return("", errors.New("some-error")),
					)

					Expect(downloadCmd.Run()).To(MatchError("some-error"))
				})
			})
		})
	})
})

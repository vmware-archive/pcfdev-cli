package pivnet_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/pivnet"
	"github.com/pivotal-cf/pcfdev-cli/pivnet/mocks"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Pivnet Token", func() {
	var (
		mockCtrl *gomock.Controller
		mockFS   *mocks.MockFS
		mockUI   *mocks.MockUI
		token    *pivnet.Token
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockFS = mocks.NewMockFS(mockCtrl)
		mockUI = mocks.NewMockUI(mockCtrl)
		token = &pivnet.Token{
			Config: &config.Config{
				PCFDevHome: "some-pcfdev-home",
			},
			FS: mockFS,
			UI: mockUI,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("#Get", func() {
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
				mockUI.EXPECT().Say("PIVNET_TOKEN set, ignored saved PivNet API token.")
				Expect(token.Get()).To(Equal("some-token"))
			})
		})

		Context("when PIVNET_TOKEN env var is not set", func() {
			var savedToken string

			BeforeEach(func() {
				savedToken = os.Getenv("PIVNET_TOKEN")
				os.Setenv("PIVNET_TOKEN", "")
			})

			AfterEach(func() {
				os.Setenv("PIVNET_TOKEN", savedToken)
			})

			Context("when a token exists at the token file path", func() {
				It("should return the token from the file path", func() {
					gomock.InOrder(
						mockFS.EXPECT().Exists(filepath.Join("some-pcfdev-home", "token")).Return(true, nil),
						mockFS.EXPECT().Read(filepath.Join("some-pcfdev-home", "token")).Return([]byte("some-saved-token"), nil),
					)

					Expect(token.Get()).To(Equal("some-saved-token"))
				})
			})

			Context("when a token does not exist at the token file path", func() {
				It("should prompt the user to enter their Pivnet token", func() {
					gomock.InOrder(
						mockFS.EXPECT().Exists(filepath.Join("some-pcfdev-home", "token")).Return(false, nil),
						mockUI.EXPECT().Say("Please retrieve your Pivotal Network API token from:"),
						mockUI.EXPECT().Say("https://network.pivotal.io/users/dashboard/edit-profile"),
						mockUI.EXPECT().AskForPassword("API token").Return("some-user-provided-token"),
					)

					Expect(token.Get()).To(Equal("some-user-provided-token"))
				})
			})

			Context("when pivnet token has already been fetched", func() {
				It("should return the same value", func() {
					gomock.InOrder(
						mockFS.EXPECT().Exists(filepath.Join("some-pcfdev-home", "token")).Times(1),
						mockUI.EXPECT().Say("Please retrieve your Pivotal Network API token from:").Times(1),
						mockUI.EXPECT().Say("https://network.pivotal.io/users/dashboard/edit-profile").Times(1),
						mockUI.EXPECT().AskForPassword("API token").Return("some-user-provided-token").Times(1),
					)
					Expect(token.Get()).To(Equal("some-user-provided-token"))
					Expect(token.Get()).To(Equal("some-user-provided-token"))
				})
			})

			Context("when call to determine whether a token's presence fails", func() {
				It("should return an error", func() {
					gomock.InOrder(
						mockFS.EXPECT().Exists(filepath.Join("some-pcfdev-home", "token")).Return(false, errors.New("some-error")),
					)

					_, err := token.Get()
					Expect(err).To(MatchError("some-error"))
				})
			})

			Context("when call to read token file fails", func() {
				It("should return an error", func() {
					gomock.InOrder(
						mockFS.EXPECT().Exists(filepath.Join("some-pcfdev-home", "token")).Return(true, nil),
						mockFS.EXPECT().Read(filepath.Join("some-pcfdev-home", "token")).Return(nil, errors.New("some-error")),
					)

					_, err := token.Get()
					Expect(err).To(MatchError("some-error"))
				})
			})
		})
	})

	Describe("#Save", func() {
		Context("when PIVNET_TOKEN env var is not set", func() {
			var savedToken string

			BeforeEach(func() {
				savedToken = os.Getenv("PIVNET_TOKEN")
				os.Setenv("PIVNET_TOKEN", "")
			})

			AfterEach(func() {
				os.Setenv("PIVNET_TOKEN", savedToken)
			})

			It("should save the token", func() {
				gomock.InOrder(
					mockFS.EXPECT().Exists(filepath.Join("some-pcfdev-home", "token")).Return(true, nil),
					mockFS.EXPECT().Read(filepath.Join("some-pcfdev-home", "token")).Return([]byte("some-user-provided-token"), nil),
					mockFS.EXPECT().Exists(filepath.Join("some-pcfdev-home", "token")).Return(false, nil),
					mockFS.EXPECT().Write(filepath.Join("some-pcfdev-home", "token"), strings.NewReader("some-user-provided-token")),
				)

				token.Get()
				Expect(token.Save()).To(Succeed())
			})

			It("should remove old token if it exists", func() {
				gomock.InOrder(
					mockFS.EXPECT().Exists(filepath.Join("some-pcfdev-home", "token")).Return(true, nil),
					mockFS.EXPECT().Read(filepath.Join("some-pcfdev-home", "token")).Return([]byte("some-user-provided-token"), nil),
					mockFS.EXPECT().Exists(filepath.Join("some-pcfdev-home", "token")).Return(true, nil),
					mockFS.EXPECT().Remove(filepath.Join("some-pcfdev-home", "token")),
					mockFS.EXPECT().Write(filepath.Join("some-pcfdev-home", "token"), strings.NewReader("some-user-provided-token")),
				)

				token.Get()
				Expect(token.Save()).To(Succeed())
			})
		})

		Context("when PIVNET_TOKEN env var is set", func() {
			var savedToken string

			BeforeEach(func() {
				savedToken = os.Getenv("PIVNET_TOKEN")
				os.Setenv("PIVNET_TOKEN", "some-token")
			})

			AfterEach(func() {
				os.Setenv("PIVNET_TOKEN", savedToken)
			})

			It("should not save the token", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("PIVNET_TOKEN set, ignored saved PivNet API token."),
				)

				token.Get()
				Expect(token.Save()).To(Succeed())
			})
		})
	})

	Describe("#Destroy", func() {
		var savedToken string

		BeforeEach(func() {
			savedToken = os.Getenv("PIVNET_TOKEN")
			os.Setenv("PIVNET_TOKEN", "")
		})

		AfterEach(func() {
			os.Setenv("PIVNET_TOKEN", savedToken)
		})

		Context("when the token is saved to file", func() {
			It("should delete the token file", func() {
				gomock.InOrder(
					mockFS.EXPECT().Exists(filepath.Join("some-pcfdev-home", "token")).Return(true, nil),
					mockFS.EXPECT().Remove(filepath.Join("some-pcfdev-home", "token")),
				)
				Expect(token.Destroy()).To(Succeed())
			})
		})

		Context("when the token is not saved to file", func() {
			It("should not throw an error", func() {
				mockFS.EXPECT().Exists(filepath.Join("some-pcfdev-home", "token")).Return(false, nil)
				Expect(token.Destroy()).To(Succeed())
			})
		})

		Context("when there is an error seeing if token exists", func() {
			It("should throw an error", func() {
				mockFS.EXPECT().Exists(filepath.Join("some-pcfdev-home", "token")).Return(false, errors.New("some-error"))
				Expect(token.Destroy()).To(MatchError("some-error"))
			})
		})

		Context("when there is an error removing the token", func() {
			It("should throw an error", func() {
				gomock.InOrder(
					mockFS.EXPECT().Exists(filepath.Join("some-pcfdev-home", "token")).Return(true, nil),
					mockFS.EXPECT().Remove(filepath.Join("some-pcfdev-home", "token")).Return(errors.New("some-error")),
				)
				Expect(token.Destroy()).To(MatchError("some-error"))
			})
		})

		Context("when PIVNET_TOKEN is set", func() {
			BeforeEach(func() {
				os.Setenv("PIVNET_TOKEN", "some-pivnet-token")
			})

			It("should not destroy the token", func() {
				Expect(token.Destroy()).To(Succeed())
			})
		})
	})
})

package config_test

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/mock/gomock"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/config/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	Context("#GetToken", func() {
		var pcfdevHome string

		BeforeEach(func() {
			pcfdevHome = os.Getenv("PCFDEV_HOME")

			os.Setenv("PCFDEV_HOME", "some-pcfdev-home")
		})

		AfterEach(func() {
			os.Setenv("PCFDEV_HOME", pcfdevHome)
		})

		Context("when PIVNET_TOKEN env var is set", func() {
			var (
				savedToken string
				mockUI     *mocks.MockUI
				mockCtrl   *gomock.Controller
			)

			BeforeEach(func() {
				savedToken = os.Getenv("PIVNET_TOKEN")
				mockCtrl = gomock.NewController(GinkgoT())
				mockUI = mocks.NewMockUI(mockCtrl)

				os.Setenv("PIVNET_TOKEN", "some-token")
			})

			AfterEach(func() {
				os.Setenv("PIVNET_TOKEN", savedToken)
				mockCtrl.Finish()
			})

			It("should return PIVNET_TOKEN env var", func() {
				config := &config.Config{
					UI: mockUI,
				}

				mockUI.EXPECT().Say("PIVNET_TOKEN set, ignored saved PivNet API token.")
				Expect(config.GetToken()).To(Equal("some-token"))
			})

			Context("when a token exists at the token file path", func() {
				var tempDir string

				BeforeEach(func() {
					var err error
					tempDir, err = ioutil.TempDir("", "")
					Expect(err).NotTo(HaveOccurred())
					ioutil.WriteFile(filepath.Join(tempDir, "token"), []byte("some-different-token"), 0644)
				})

				AfterEach(func() {
					os.RemoveAll(tempDir)
				})

				It("should return PIVNET_TOKEN env var", func() {
					config := &config.Config{
						UI: mockUI,
					}

					mockUI.EXPECT().Say("PIVNET_TOKEN set, ignored saved PivNet API token.")
					Expect(config.GetToken()).To(Equal("some-token"))
				})
			})
		})

		Context("when PIVNET_TOKEN env var is not set", func() {
			var (
				savedToken string
				mockUI     *mocks.MockUI
				mockCtrl   *gomock.Controller
				mockFS     *mocks.MockFS
				cfg        *config.Config
			)

			BeforeEach(func() {
				savedToken = os.Getenv("PIVNET_TOKEN")
				os.Setenv("PIVNET_TOKEN", "")
				mockCtrl = gomock.NewController(GinkgoT())
				mockFS = mocks.NewMockFS(mockCtrl)
				mockUI = mocks.NewMockUI(mockCtrl)

				cfg = &config.Config{
					UI: mockUI,
					FS: mockFS,
				}
			})

			AfterEach(func() {
				os.Setenv("PIVNET_TOKEN", savedToken)
				mockCtrl.Finish()
			})

			Context("when a token exists at the token file path", func() {
				It("should return PIVNET_TOKEN env var", func() {
					gomock.InOrder(
						mockFS.EXPECT().Exists(filepath.Join("some-pcfdev-home", ".pcfdev", "token")).Return(true, nil),
						mockFS.EXPECT().Read(filepath.Join("some-pcfdev-home", ".pcfdev", "token")).Return([]byte("some-saved-token"), nil),
					)

					Expect(cfg.GetToken()).To(Equal("some-saved-token"))
				})
			})

			Context("when pivnet token has already been fetched", func() {
				It("should return the same value", func() {
					gomock.InOrder(
						mockFS.EXPECT().Exists(filepath.Join("some-pcfdev-home", ".pcfdev", "token")).Times(1),
						mockUI.EXPECT().Say("Please retrieve your Pivotal Network API from:").Times(1),
						mockUI.EXPECT().Say("https://network.pivotal.io/users/dashboard/edit-profile").Times(1),
						mockUI.EXPECT().AskForPassword("API token").Return("some-user-provided-token").Times(1),
					)
					Expect(cfg.GetToken()).To(Equal("some-user-provided-token"))
					Expect(cfg.GetToken()).To(Equal("some-user-provided-token"))
				})
			})

			Context("when a token does not exist at the token file path", func() {
				It("should prompt the user to enter their Pivnet token", func() {
					gomock.InOrder(
						mockFS.EXPECT().Exists(filepath.Join("some-pcfdev-home", ".pcfdev", "token")).Return(false, nil),
						mockUI.EXPECT().Say("Please retrieve your Pivotal Network API from:"),
						mockUI.EXPECT().Say("https://network.pivotal.io/users/dashboard/edit-profile"),
						mockUI.EXPECT().AskForPassword("API token").Return("some-user-provided-token"),
					)

					Expect(cfg.GetToken()).To(Equal("some-user-provided-token"))
				})
			})

			Context("when call to determine whether a token's presence fails", func() {
				It("should return PIVNET_TOKEN env var", func() {
					gomock.InOrder(
						mockFS.EXPECT().Exists(filepath.Join("some-pcfdev-home", ".pcfdev", "token")).Return(false, errors.New("some-error")),
					)

					_, err := cfg.GetToken()
					Expect(err).To(MatchError("some-error"))
				})
			})

			Context("when call to read token file fails", func() {
				It("should return PIVNET_TOKEN env var", func() {
					gomock.InOrder(
						mockFS.EXPECT().Exists(filepath.Join("some-pcfdev-home", ".pcfdev", "token")).Return(true, nil),
						mockFS.EXPECT().Read(filepath.Join("some-pcfdev-home", ".pcfdev", "token")).Return(nil, errors.New("some-error")),
					)

					_, err := cfg.GetToken()
					Expect(err).To(MatchError("some-error"))
				})
			})
		})
	})

	Context("#SaveToken", func() {
		var pcfdevHome string

		BeforeEach(func() {
			pcfdevHome = os.Getenv("PCFDEV_HOME")

			os.Setenv("PCFDEV_HOME", "some-pcfdev-home")
		})

		AfterEach(func() {
			os.Setenv("PCFDEV_HOME", pcfdevHome)
		})

		Context("when PIVNET_TOKEN env var is not set", func() {
			var (
				savedToken string
				mockUI     *mocks.MockUI
				mockFS     *mocks.MockFS
				mockCtrl   *gomock.Controller
				cfg        *config.Config
			)

			BeforeEach(func() {
				savedToken = os.Getenv("PIVNET_TOKEN")
				os.Setenv("PIVNET_TOKEN", "")
				mockCtrl = gomock.NewController(GinkgoT())
				mockUI = mocks.NewMockUI(mockCtrl)
				mockFS = mocks.NewMockFS(mockCtrl)

				cfg = &config.Config{
					FS: mockFS,
					UI: mockUI,
				}
			})

			AfterEach(func() {
				os.Setenv("PIVNET_TOKEN", savedToken)
				mockCtrl.Finish()
			})

			It("should save the token", func() {
				gomock.InOrder(
					mockFS.EXPECT().Exists(filepath.Join("some-pcfdev-home", ".pcfdev", "token")).Return(true, nil),
					mockFS.EXPECT().Read(filepath.Join("some-pcfdev-home", ".pcfdev", "token")).Return([]byte("some-user-provided-token"), nil),
					mockFS.EXPECT().Write(filepath.Join("some-pcfdev-home", ".pcfdev", "token"), strings.NewReader("some-user-provided-token")),
				)

				cfg.GetToken()
				Expect(cfg.SaveToken()).To(Succeed())
			})
		})

		Context("when PIVNET_TOKEN env var is set", func() {
			var (
				savedToken string
				mockUI     *mocks.MockUI
				mockFS     *mocks.MockFS
				mockCtrl   *gomock.Controller
				cfg        *config.Config
			)

			BeforeEach(func() {
				savedToken = os.Getenv("PIVNET_TOKEN")
				os.Setenv("PIVNET_TOKEN", "some-token")
				mockCtrl = gomock.NewController(GinkgoT())
				mockUI = mocks.NewMockUI(mockCtrl)
				mockFS = mocks.NewMockFS(mockCtrl)

				cfg = &config.Config{
					FS: mockFS,
					UI: mockUI,
				}
			})

			AfterEach(func() {
				os.Setenv("PIVNET_TOKEN", savedToken)
				mockCtrl.Finish()
			})

			It("should not save the token", func() {
				mockUI.EXPECT().Say("PIVNET_TOKEN set, ignored saved PivNet API token.")

				cfg.GetToken()
				Expect(cfg.SaveToken()).To(Succeed())
			})
		})
	})

	Context("#DestroyToken", func() {
		var (
			pcfdevHome string
			mockFS     *mocks.MockFS
			mockCtrl   *gomock.Controller
			cfg        *config.Config
		)

		BeforeEach(func() {
			mockCtrl = gomock.NewController(GinkgoT())
			mockFS = mocks.NewMockFS(mockCtrl)

			cfg = &config.Config{
				FS: mockFS,
			}

			pcfdevHome = os.Getenv("PCFDEV_HOME")

			os.Setenv("PCFDEV_HOME", "some-pcfdev-home")
		})

		AfterEach(func() {
			os.Setenv("PCFDEV_HOME", pcfdevHome)
			mockCtrl.Finish()
		})

		Context("when the token is saved to file", func() {
			It("should delete the token file", func() {
				gomock.InOrder(
					mockFS.EXPECT().Exists(filepath.Join("some-pcfdev-home", ".pcfdev", "token")).Return(true, nil),
					mockFS.EXPECT().RemoveFile(filepath.Join("some-pcfdev-home", ".pcfdev", "token")).Return(nil),
				)
				Expect(cfg.DestroyToken()).To(Succeed())
			})
		})

		Context("when the token is not saved to file", func() {
			It("should not throw an error", func() {
				mockFS.EXPECT().Exists(filepath.Join("some-pcfdev-home", ".pcfdev", "token")).Return(false, nil)
				Expect(cfg.DestroyToken()).To(Succeed())
			})
		})
	})

	Context("#GetPCFDevDir", func() {
		Context("when the PCFDEV_HOME Environment variable is set", func() {
			var pcfdevHome string

			BeforeEach(func() {
				pcfdevHome = os.Getenv("PCFDEV_HOME")
			})

			AfterEach(func() {
				os.Setenv("PCFDEV_HOME", pcfdevHome)
			})

			It("should return the PCF Dev directory", func() {
				os.Setenv("PCFDEV_HOME", "some-dir")
				config := &config.Config{}
				Expect(config.GetPCFDevDir()).To(Equal(filepath.Join("some-dir", ".pcfdev")))
			})
		})
	})

	Context("#GetOVAPath", func() {
		Context("when the PCFDEV_HOME Environment variable is set", func() {
			var pcfdevHome string

			BeforeEach(func() {
				pcfdevHome = os.Getenv("PCFDEV_HOME")
			})

			AfterEach(func() {
				os.Setenv("PCFDEV_HOME", pcfdevHome)
			})

			It("should return the PCF Dev directory", func() {
				os.Setenv("PCFDEV_HOME", "some-dir")
				config := &config.Config{
					VMName: "some-vm",
				}
				Expect(config.GetOVAPath()).To(Equal(filepath.Join("some-dir", ".pcfdev", "some-vm.ova")))
			})
		})
	})

	Context("#GetVMName", func() {
		It("should return the VM Name", func() {
			config := &config.Config{
				VMName: "some-vm",
			}
			Expect(config.GetVMName()).To(Equal("some-vm"))
		})
	})

	Describe("#GetHTTPProxy", func() {
		Context("when HTTP_PROXY is set", func() {
			var (
				savedProxy string
			)

			BeforeEach(func() {
				savedProxy = os.Getenv("HTTP_PROXY")
				os.Setenv("HTTP_PROXY", "some-http-proxy")
			})

			AfterEach(func() {
				os.Setenv("HTTP_PROXY", savedProxy)
			})

			It("should return the HTTP_PROXY environment variable", func() {
				cfg := &config.Config{}
				Expect(cfg.GetHTTPProxy()).To(Equal("some-http-proxy"))
			})
		})

		Context("when http_proxy is set", func() {
			var (
				savedProxy string
			)

			BeforeEach(func() {
				savedProxy = os.Getenv("http_proxy")
				os.Setenv("http_proxy", "some-http-proxy")
			})

			AfterEach(func() {
				os.Setenv("http_proxy", savedProxy)
			})

			It("should return the http_proxy environment variable", func() {
				cfg := &config.Config{}
				Expect(cfg.GetHTTPProxy()).To(Equal("some-http-proxy"))
			})
		})

		Context("when HTTP_PROXY and http_proxy are set", func() {
			var (
				savedProxyCaps  string
				savedProxyLower string
			)

			BeforeEach(func() {
				savedProxyCaps = os.Getenv("HTTP_PROXY")
				savedProxyLower = os.Getenv("http_proxy")
				os.Setenv("http_proxy", "some-http-proxy")
				os.Setenv("HTTP_PROXY", "some-other-http-proxy")
			})

			AfterEach(func() {
				os.Setenv("http_proxy", savedProxyLower)
				os.Setenv("HTTP_PROXY", savedProxyCaps)
			})

			It("should return the HTTP_PROXY environment variable", func() {
				cfg := &config.Config{}
				Expect(cfg.GetHTTPProxy()).To(Equal("some-other-http-proxy"))
			})
		})
	})

	Describe("#GetHTTPSProxy", func() {
		Context("when HTTPS_PROXY is set", func() {
			var (
				savedProxy string
			)

			BeforeEach(func() {
				savedProxy = os.Getenv("HTTPS_PROXY")
				os.Setenv("HTTPS_PROXY", "some-https-proxy")
			})

			AfterEach(func() {
				os.Setenv("HTTPS_PROXY", savedProxy)
			})

			It("should return the HTTPS_PROXY environment variable", func() {
				cfg := &config.Config{}
				Expect(cfg.GetHTTPSProxy()).To(Equal("some-https-proxy"))
			})
		})

		Context("when https_proxy is set", func() {
			var (
				savedProxy string
			)

			BeforeEach(func() {
				savedProxy = os.Getenv("https_proxy")
				os.Setenv("https_proxy", "some-https-proxy")
			})

			AfterEach(func() {
				os.Setenv("https_proxy", savedProxy)
			})

			It("should return the https_proxy environment variable", func() {
				cfg := &config.Config{}
				Expect(cfg.GetHTTPSProxy()).To(Equal("some-https-proxy"))
			})
		})

		Context("when HTTPS_PROXY and https_proxy are set", func() {
			var (
				savedProxyCaps  string
				savedProxyLower string
			)

			BeforeEach(func() {
				savedProxyCaps = os.Getenv("HTTPS_PROXY")
				savedProxyLower = os.Getenv("https_proxy")
				os.Setenv("https_proxy", "some-https-proxy")
				os.Setenv("HTTPS_PROXY", "some-other-https-proxy")
			})

			AfterEach(func() {
				os.Setenv("https_proxy", savedProxyLower)
				os.Setenv("HTTPS_PROXY", savedProxyCaps)
			})

			It("should return the HTTPS_PROXY environment variable", func() {
				cfg := &config.Config{}
				Expect(cfg.GetHTTPSProxy()).To(Equal("some-other-https-proxy"))
			})
		})
	})

	Describe("#GetNoProxy", func() {
		Context("when NO_PROXY is set", func() {
			var (
				savedProxy string
			)

			BeforeEach(func() {
				savedProxy = os.Getenv("NO_PROXY")
				os.Setenv("NO_PROXY", "some-no-proxy")
			})

			AfterEach(func() {
				os.Setenv("NO_PROXY", savedProxy)
			})

			It("should return the NO_PROXY environment variable", func() {
				cfg := &config.Config{}
				Expect(cfg.GetNoProxy()).To(Equal("some-no-proxy"))
			})
		})

		Context("when no_proxy is set", func() {
			var (
				savedProxy string
			)

			BeforeEach(func() {
				savedProxy = os.Getenv("no_proxy")
				os.Setenv("no_proxy", "some-no-proxy")
			})

			AfterEach(func() {
				os.Setenv("no_proxy", savedProxy)
			})

			It("should return the no_proxy environment variable", func() {
				cfg := &config.Config{}
				Expect(cfg.GetNoProxy()).To(Equal("some-no-proxy"))
			})
		})

		Context("when NO_PROXY and no_proxy are set", func() {
			var (
				savedProxyCaps  string
				savedProxyLower string
			)

			BeforeEach(func() {
				savedProxyCaps = os.Getenv("NO_PROXY")
				savedProxyLower = os.Getenv("no_proxy")
				os.Setenv("no_proxy", "some-no-proxy")
				os.Setenv("NO_PROXY", "some-other-no-proxy")
			})

			AfterEach(func() {
				os.Setenv("no_proxy", savedProxyLower)
				os.Setenv("NO_PROXY", savedProxyCaps)
			})

			It("should return the NO_PROXY environment variable", func() {
				cfg := &config.Config{}
				Expect(cfg.GetNoProxy()).To(Equal("some-other-no-proxy"))
			})
		})
	})

	Describe("#GetMinMemory", func() {
		It("should return MinMemory", func() {
			cfg := &config.Config{
				MinMemory: uint64(1024),
			}
			Expect(cfg.GetMinMemory()).To(Equal(uint64(1024)))
		})
	})

	Describe("#GetMaxMemory", func() {
		It("should return MaxMemory", func() {
			cfg := &config.Config{
				MaxMemory: uint64(1024),
			}
			Expect(cfg.GetMaxMemory()).To(Equal(uint64(1024)))
		})
	})

	Describe("#GetDesiredMemory", func() {
		var (
			cfg           *config.Config
			savedVMMemory string
		)

		BeforeEach(func() {
			cfg = &config.Config{}
			savedVMMemory = os.Getenv("VM_MEMORY")
		})

		AfterEach(func() {
			os.Setenv("VM_MEMORY", savedVMMemory)
		})

		Context("when VM_MEMORY env var is set", func() {
			It("should return VM_MEMORY", func() {
				os.Setenv("VM_MEMORY", "1024")
				Expect(cfg.GetDesiredMemory()).To(Equal(uint64(1024)))
			})

		})

		Context("when VM_MEMORY is not an integer", func() {
			It("should return an error", func() {
				os.Setenv("VM_MEMORY", "some-string")
				_, err := cfg.GetDesiredMemory()
				Expect(err).To(MatchError(ContainSubstring("could not convert VM_MEMORY \"some-string\" to integer:")))
			})
		})

		Context("when VM_MEMORY env var is not set", func() {
			It("should return VM_MEMORY", func() {
				os.Setenv("VM_MEMORY", "")
				Expect(cfg.GetDesiredMemory()).To(Equal(uint64(0)))
			})
		})
	})
})

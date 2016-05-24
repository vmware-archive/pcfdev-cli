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
	Describe("#GetToken", func() {
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
				mockCtrl.Finish()
			})

			It("should prompt the user to enter their Pivnet token", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Please retrieve your Pivotal Network API from:"),
					mockUI.EXPECT().Say("https://network.pivotal.io/users/dashboard/edit-profile"),
					mockUI.EXPECT().AskForPassword("API token").Return("some-user-provided-token"),
				)

				Expect(cfg.GetToken()).To(Equal("some-user-provided-token"))
			})

			Context("when pivnet token has already been fetched", func() {
				It("should return the same value", func() {
					gomock.InOrder(
						mockUI.EXPECT().Say("Please retrieve your Pivotal Network API from:").Times(1),
						mockUI.EXPECT().Say("https://network.pivotal.io/users/dashboard/edit-profile").Times(1),
						mockUI.EXPECT().AskForPassword("API token").Return("some-user-provided-token").Times(1),
					)
					Expect(cfg.GetToken()).To(Equal("some-user-provided-token"))
					Expect(cfg.GetToken()).To(Equal("some-user-provided-token"))
				})
			})
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

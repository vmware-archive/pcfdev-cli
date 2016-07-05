package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"

	"github.com/golang/mock/gomock"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/config/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	Describe("New", func() {
		var (
			savedPCFDevHome string
			savedHTTPProxy  string
			savedHTTPSProxy string
			savedNoProxy    string
			savedVMMemory   string
			mockCtrl        *gomock.Controller
			mockSystem      *mocks.MockSystem
		)

		BeforeEach(func() {
			savedPCFDevHome = os.Getenv("PCFDEV_HOME")
			savedHTTPProxy = os.Getenv("HTTP_PROXY")
			savedHTTPSProxy = os.Getenv("HTTPS_PROXY")
			savedNoProxy = os.Getenv("NO_PROXY")
			savedVMMemory = os.Getenv("VM_MEMORY")

			os.Setenv("PCFDEV_HOME", "some-pcfdev-home")
			os.Setenv("HTTP_PROXY", "some-http-proxy")
			os.Setenv("HTTPS_PROXY", "some-https-proxy")
			os.Setenv("NO_PROXY", "some-no-proxy")
			os.Setenv("VM_MEMORY", "1024")

			mockCtrl = gomock.NewController(GinkgoT())
			mockSystem = mocks.NewMockSystem(mockCtrl)
		})

		AfterEach(func() {
			os.Setenv("PCFDEV_HOME", savedPCFDevHome)
			os.Setenv("HTTP_PROXY", savedHTTPProxy)
			os.Setenv("HTTPS_PROXY", savedHTTPSProxy)
			os.Setenv("NO_PROXY", savedNoProxy)
			os.Setenv("VM_MEMORY", savedVMMemory)

			mockCtrl.Finish()
		})

		It("should use given values and env vars to set fields", func() {
			mockSystem.EXPECT().FreeMemory().Return(uint64(2000), nil)
			mockSystem.EXPECT().TotalMemory().Return(uint64(1000), nil)
			mockSystem.EXPECT().PhysicalCores().Return(4, nil)
			conf, err := config.New("some-vm", mockSystem)
			Expect(err).NotTo(HaveOccurred())
			Expect(conf.DefaultVMName).To(Equal("some-vm"))
			Expect(conf.PCFDevHome).To(Equal("some-pcfdev-home"))
			Expect(conf.OVADir).To(Equal(filepath.Join("some-pcfdev-home", "ova")))
			Expect(conf.VMDir).To(Equal(filepath.Join("some-pcfdev-home", "vms")))
			Expect(conf.HTTPProxy).To(Equal("some-http-proxy"))
			Expect(conf.HTTPSProxy).To(Equal("some-https-proxy"))
			Expect(conf.NoProxy).To(Equal("some-no-proxy"))
			Expect(conf.MinMemory).To(Equal(uint64(3072)))
			Expect(conf.MaxMemory).To(Equal(uint64(4096)))
			Expect(conf.SpringCloudMemoryIncrease).To(Equal(uint64(2048)))
		})

		Context("when caps proxy env vars are unset", func() {
			var (
				savedLowerHTTPProxy  string
				savedLowerHTTPSProxy string
				savedLowerNoProxy    string
			)

			BeforeEach(func() {
				savedLowerHTTPProxy = os.Getenv("HTTP_PROXY")
				savedLowerHTTPSProxy = os.Getenv("HTTPS_PROXY")
				savedLowerNoProxy = os.Getenv("NO_PROXY")

				os.Setenv("HTTP_PROXY", "")
				os.Setenv("HTTPS_PROXY", "")
				os.Setenv("NO_PROXY", "")

				os.Setenv("http_proxy", "some-other-http-proxy")
				os.Setenv("https_proxy", "some-other-https-proxy")
				os.Setenv("no_proxy", "some-other-no-proxy")
			})

			AfterEach(func() {
				os.Setenv("http_proxy", savedLowerHTTPProxy)
				os.Setenv("https_proxy", savedLowerHTTPSProxy)
				os.Setenv("no_proxy", savedLowerNoProxy)
			})

			It("should use lower case env vars", func() {
				mockSystem.EXPECT().FreeMemory().Return(uint64(2000), nil)
				mockSystem.EXPECT().TotalMemory().Return(uint64(1000), nil)
				mockSystem.EXPECT().PhysicalCores().Return(4, nil)
				conf, err := config.New("some-vm", mockSystem)
				Expect(err).NotTo(HaveOccurred())
				Expect(conf.HTTPProxy).To(Equal("some-other-http-proxy"))
				Expect(conf.HTTPSProxy).To(Equal("some-other-https-proxy"))
				Expect(conf.NoProxy).To(Equal("some-other-no-proxy"))
			})
		})

		Context("when caps and lower proxy env vars are set", func() {
			var (
				savedLowerHTTPProxy  string
				savedLowerHTTPSProxy string
				savedLowerNoProxy    string
			)

			BeforeEach(func() {
				savedLowerHTTPProxy = os.Getenv("HTTP_PROXY")
				savedLowerHTTPSProxy = os.Getenv("HTTPS_PROXY")
				savedLowerNoProxy = os.Getenv("NO_PROXY")

				os.Setenv("http_proxy", "some-other-http-proxy")
				os.Setenv("https_proxy", "some-other-https-proxy")
				os.Setenv("no_proxy", "some-other-no-proxy")
			})

			AfterEach(func() {
				os.Setenv("http_proxy", savedLowerHTTPProxy)
				os.Setenv("https_proxy", savedLowerHTTPSProxy)
				os.Setenv("no_proxy", savedLowerNoProxy)
			})

			It("should prefer caps env vars", func() {
				if runtime.GOOS == "windows" {
					Skip("does not apply on windows - env vars are case-insensitive")
				}
				mockSystem.EXPECT().FreeMemory().Return(uint64(2000), nil)
				mockSystem.EXPECT().TotalMemory().Return(uint64(1000), nil)
				mockSystem.EXPECT().PhysicalCores().Return(4, nil)
				conf, err := config.New("some-vm", mockSystem)
				Expect(err).NotTo(HaveOccurred())
				Expect(conf.HTTPProxy).To(Equal("some-http-proxy"))
				Expect(conf.HTTPSProxy).To(Equal("some-https-proxy"))
				Expect(conf.NoProxy).To(Equal("some-no-proxy"))
			})
		})

		Context("when proxy env vars contain spaces", func() {
			var (
				savedLowerHTTPProxy  string
				savedLowerHTTPSProxy string
				savedLowerNoProxy    string
			)

			BeforeEach(func() {
				savedLowerHTTPProxy = os.Getenv("HTTP_PROXY")
				savedLowerHTTPSProxy = os.Getenv("HTTPS_PROXY")
				savedLowerNoProxy = os.Getenv("NO_PROXY")

				os.Setenv("HTTP_PROXY", "   some http\tproxy\nwith\r\nwhitespace   ")
				os.Setenv("HTTPS_PROXY", "   some https\tproxy\nwith\r\nwhitespace   ")
				os.Setenv("NO_PROXY", "   some no\tproxy\nwith\r\nwhitespace   ")
			})

			AfterEach(func() {
				os.Setenv("HTTP_PROXY", savedLowerHTTPProxy)
				os.Setenv("HTTPS_PROXY", savedLowerHTTPSProxy)
				os.Setenv("NO_PROXY", savedLowerNoProxy)
			})

			It("should strip all whitespace", func() {
				mockSystem.EXPECT().FreeMemory().Return(uint64(2000), nil)
				mockSystem.EXPECT().TotalMemory().Return(uint64(1000), nil)
				mockSystem.EXPECT().PhysicalCores().Return(4, nil)
				conf, err := config.New("some-vm", mockSystem)
				Expect(err).NotTo(HaveOccurred())
				Expect(conf.HTTPProxy).To(Equal("somehttpproxywithwhitespace"))
				Expect(conf.HTTPSProxy).To(Equal("somehttpsproxywithwhitespace"))
				Expect(conf.NoProxy).To(Equal("somenoproxywithwhitespace"))
			})
		})

		Context("when PCFDEV_HOME is not set", func() {
			It("should use a .pcfdev dir within the user's home", func() {
				var expectedHome string
				if runtime.GOOS == "windows" {
					expectedHome = filepath.Join(os.Getenv("HOMEDRIVE"), os.Getenv("HOMEPATH"))
				} else {
					expectedHome = os.Getenv("HOME")
				}

				mockSystem.EXPECT().FreeMemory().Return(uint64(2000), nil)
				mockSystem.EXPECT().TotalMemory().Return(uint64(1000), nil)
				mockSystem.EXPECT().PhysicalCores().Return(4, nil)
				os.Unsetenv("PCFDEV_HOME")

				conf, err := config.New("some-vm", mockSystem)
				Expect(err).NotTo(HaveOccurred())
				Expect(conf.PCFDevHome).To(Equal(filepath.Join(expectedHome, ".pcfdev")))
				Expect(conf.OVADir).To(Equal(filepath.Join(expectedHome, ".pcfdev", "ova")))
			})
		})

		Context("memory", func() {
			It("should set the total system memory", func() {
				mockSystem.EXPECT().FreeMemory().Return(uint64(2000), nil)
				mockSystem.EXPECT().TotalMemory().Return(uint64(1000), nil)
				mockSystem.EXPECT().PhysicalCores().Return(4, nil)

				conf, err := config.New("some-vm", mockSystem)
				Expect(err).NotTo(HaveOccurred())
				Expect(conf.TotalMemory).To(Equal(uint64(1000)))
			})

			It("should set the free system memory", func() {
				mockSystem.EXPECT().FreeMemory().Return(uint64(2000), nil)
				mockSystem.EXPECT().TotalMemory().Return(uint64(1000), nil)
				mockSystem.EXPECT().PhysicalCores().Return(4, nil)

				conf, err := config.New("some-vm", mockSystem)
				Expect(err).NotTo(HaveOccurred())
				Expect(conf.FreeMemory).To(Equal(uint64(2000)))
			})

			Context("when half of the total system memory is between the minimum and maximum", func() {
				It("should give the VM half the total memory", func() {
					mockSystem.EXPECT().FreeMemory().Return(uint64(2000), nil)
					mockSystem.EXPECT().TotalMemory().Return(uint64(7000), nil)
					mockSystem.EXPECT().PhysicalCores().Return(4, nil)

					conf, err := config.New("some-vm", mockSystem)
					Expect(err).NotTo(HaveOccurred())
					Expect(conf.DefaultMemory).To(Equal(uint64(3500)))
				})
			})

			Context("when half of the total system memory is less than the minimum", func() {
				It("should give the VM the minimum amount of memory", func() {
					mockSystem.EXPECT().FreeMemory().Return(uint64(2000), nil)
					mockSystem.EXPECT().TotalMemory().Return(uint64(6000), nil)
					mockSystem.EXPECT().PhysicalCores().Return(4, nil)

					conf, err := config.New("some-vm", mockSystem)
					Expect(err).NotTo(HaveOccurred())
					Expect(conf.DefaultMemory).To(Equal(uint64(3072)))
				})
			})

			Context("when half of the total system memory is more than the maximum", func() {
				It("should give the VM the maximum amount of memory", func() {
					mockSystem.EXPECT().FreeMemory().Return(uint64(2000), nil)
					mockSystem.EXPECT().TotalMemory().Return(uint64(60000), nil)
					mockSystem.EXPECT().PhysicalCores().Return(4, nil)

					conf, err := config.New("some-vm", mockSystem)
					Expect(err).NotTo(HaveOccurred())
					Expect(conf.DefaultMemory).To(Equal(uint64(4096)))
				})
			})

			Context("when getting free memory fails", func() {
				It("should return an error", func() {
					mockSystem.EXPECT().FreeMemory().Return(uint64(0), errors.New("some-error"))

					_, err := config.New("some-vm", mockSystem)
					Expect(err).To(MatchError("some-error"))
				})
			})

			Context("when getting total memory fails", func() {
				It("should return an error", func() {
					mockSystem.EXPECT().FreeMemory().Return(uint64(2000), nil)
					mockSystem.EXPECT().TotalMemory().Return(uint64(0), errors.New("some-error"))

					_, err := config.New("some-vm", mockSystem)
					Expect(err).To(MatchError("some-error"))
				})
			})
		})
		Context("DefaultCPUs", func() {
			It("should use the number of physical cores", func() {
				mockSystem.EXPECT().FreeMemory().Return(uint64(2000), nil)
				mockSystem.EXPECT().TotalMemory().Return(uint64(60000), nil)
				mockSystem.EXPECT().PhysicalCores().Return(4, nil)
				conf, err := config.New("some-vm", mockSystem)
				Expect(err).NotTo(HaveOccurred())
				Expect(conf.DefaultCPUs).To(Equal(4))
			})

			Context("when there is an error getting the number of cores", func() {
				It("should return an error", func() {
					mockSystem.EXPECT().FreeMemory().Return(uint64(2000), nil)
					mockSystem.EXPECT().TotalMemory().Return(uint64(60000), nil)
					mockSystem.EXPECT().PhysicalCores().Return(0, errors.New("some-error"))

					_, err := config.New("some-vm", mockSystem)
					Expect(err).To(MatchError("some-error"))
				})
			})
		})

	})
})

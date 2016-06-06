package config_test

import (
	"os"
	"path/filepath"

	"github.com/pivotal-cf/pcfdev-cli/config"

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
		})

		AfterEach(func() {
			os.Setenv("PCFDEV_HOME", savedPCFDevHome)
			os.Setenv("HTTP_PROXY", savedHTTPProxy)
			os.Setenv("HTTPS_PROXY", savedHTTPSProxy)
			os.Setenv("NO_PROXY", savedNoProxy)
			os.Setenv("VM_MEMORY", savedVMMemory)
		})

		It("should use given values and env vars to set fields", func() {
			conf, err := config.New("some-vm", uint64(1024), uint64(2048))
			Expect(err).NotTo(HaveOccurred())
			Expect(conf.DefaultVMName).To(Equal("some-vm"))
			Expect(conf.PCFDevHome).To(Equal("some-pcfdev-home"))
			Expect(conf.OVADir).To(Equal("some-pcfdev-home/ova"))
			Expect(conf.HTTPProxy).To(Equal("some-http-proxy"))
			Expect(conf.HTTPSProxy).To(Equal("some-https-proxy"))
			Expect(conf.HTTPSProxy).To(Equal("some-https-proxy"))
			Expect(conf.NoProxy).To(Equal("some-no-proxy"))
			Expect(conf.MinMemory).To(Equal(uint64(1024)))
			Expect(conf.MaxMemory).To(Equal(uint64(2048)))
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
				conf, err := config.New("some-vm", uint64(1024), uint64(2048))
				Expect(err).NotTo(HaveOccurred())
				Expect(conf.HTTPProxy).To(Equal("some-other-http-proxy"))
				Expect(conf.HTTPSProxy).To(Equal("some-other-https-proxy"))
				Expect(conf.HTTPSProxy).To(Equal("some-other-https-proxy"))
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
				conf, err := config.New("some-vm", uint64(1024), uint64(2048))
				Expect(err).NotTo(HaveOccurred())
				Expect(conf.HTTPProxy).To(Equal("some-http-proxy"))
				Expect(conf.HTTPSProxy).To(Equal("some-https-proxy"))
				Expect(conf.HTTPSProxy).To(Equal("some-https-proxy"))
			})
		})

		Context("when PCFDEV_HOME is not set", func() {
			It("should use a .pcfdev dir within the user's home", func() {
				os.Unsetenv("PCFDEV_HOME")

				conf, err := config.New("some-vm", uint64(1024), uint64(2048))
				Expect(err).NotTo(HaveOccurred())
				Expect(conf.PCFDevHome).To(Equal(filepath.Join(os.Getenv("HOME"), ".pcfdev")))
				Expect(conf.OVADir).To(Equal(filepath.Join(os.Getenv("HOME"), ".pcfdev", "ova")))
			})
		})
	})
})

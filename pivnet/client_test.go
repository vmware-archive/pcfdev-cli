package pivnet_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	"github.com/pivotal-cf/pcfdev-cli/pivnet"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PivNet Client", func() {
	Describe("#DownloadOVA", func() {
		Context("when then API token is set in env", func() {
			It("should download an ova from network.pivotal.io", func() {
				handler := func(w http.ResponseWriter, r *http.Request) {
					defer GinkgoRecover()
					Expect(r.Method).To(Equal("POST"))
					Expect(r.URL.Path).To(Equal("/api/v2/products/pcfdev/releases/1622/product_files/4149/download"))
					Expect(r.Header["Authorization"][0]).To(Equal("Token some-token"))
					w.Write([]byte("ova contents"))
				}

				ts := httptest.NewServer(http.HandlerFunc(handler))
				client := pivnet.Client{
					Host: ts.URL,
				}
				ova, err := client.DownloadOVA("some-token")
				Expect(err).NotTo(HaveOccurred())
				buf, err := ioutil.ReadAll(ova)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(buf)).To(Equal("ova contents"))
			})

			Context("when PivNet is unreachable", func() {
				It("should return an appropriate error", func() {
					client := pivnet.Client{
						Host: "some-bad-host",
					}
					_, err := client.DownloadOVA("some-token")
					Expect(err).To(MatchError(ContainSubstring("failed to reach Pivotal Network:")))
				})
			})

			Context("when PivNet returns status not OK", func() {
				It("should return an appropriate error", func() {
					handler := func(w http.ResponseWriter, r *http.Request) {
						defer GinkgoRecover()
						w.WriteHeader(400)
					}

					ts := httptest.NewServer(http.HandlerFunc(handler))
					client := pivnet.Client{
						Host: ts.URL,
					}
					_, err := client.DownloadOVA("some-token")
					Expect(err).To(MatchError("Pivotal Network returned: 400 Bad Request"))
				})
			})

			Context("when PivNet returns status 451", func() {
				It("should return an error telling user to agree to eula", func() {
					handler := func(w http.ResponseWriter, r *http.Request) {
						defer GinkgoRecover()
						w.WriteHeader(451)
					}

					ts := httptest.NewServer(http.HandlerFunc(handler))
					client := pivnet.Client{
						Host: ts.URL,
					}
					_, err := client.DownloadOVA("some-token")
					Expect(err).To(MatchError(MatchRegexp("you must accept the EULA before you can download the PCF Dev image: .*/products/pcfdev#/releases/1622")))
				})
			})

			Context("when PivNet returns status 401", func() {
				It("should return an error telling user that their pivnet token is bad", func() {
					handler := func(w http.ResponseWriter, r *http.Request) {
						defer GinkgoRecover()
						w.WriteHeader(401)
					}

					ts := httptest.NewServer(http.HandlerFunc(handler))
					client := pivnet.Client{
						Host: ts.URL,
					}
					_, err := client.DownloadOVA("some-token")
					Expect(err).To(MatchError(MatchRegexp("invalid Pivotal Network API token")))
				})
			})
		})
	})
})

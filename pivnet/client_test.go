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
		var client pivnet.Client

		BeforeEach(func() {
			client = pivnet.Client{
				ReleaseId:     "some-release-id",
				ProductFileId: "some-product-file-id",
			}
		})

		Context("when the ProductFileDownloadURI is not empty", func() {
			It("should download the specified ova from network.pivotal.io", func() {
				handler := func(w http.ResponseWriter, r *http.Request) {
					defer GinkgoRecover()

					switch r.URL.Path {
					case "/api/v2/products/pcfdev/releases/some-release-id/product_files/some-product-file-id/download":
						Expect(r.Method).To(Equal("POST"))
						Expect(r.Header["Authorization"][0]).To(Equal("Token some-token"))
						w.Header().Add("Location", "http://"+r.Host+"/some-path")
						w.WriteHeader(302)
					case "/some-path":
						Expect(r.Method).To(Equal("GET"))
						Expect(r.Header["Range"][0]).To(Equal("bytes=4-"))
						w.WriteHeader(206)
						w.Write([]byte("ova contents"))
					default:
						Fail("unexpected server request")
					}
				}
				client.Host = httptest.NewServer(http.HandlerFunc(handler)).URL

				ova, err := client.DownloadOVA("some-token", int64(4))
				Expect(err).NotTo(HaveOccurred())
				Expect(ova.ExistingLength).To(Equal(int64(4)))
				Expect(ova.ContentLength).To(Equal(int64(12)))
				buf, err := ioutil.ReadAll(ova.ReadCloser)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(buf)).To(Equal("ova contents"))
			})
		})

		Context("when PivNet is unreachable", func() {
			It("should return an appropriate error", func() {
				client.Host = "some-bad-host"

				_, err := client.DownloadOVA("some-token", int64(0))
				Expect(err).To(MatchError(ContainSubstring("failed to reach Pivotal Network:")))
			})
		})

		Context("when PivNet returns status not OK", func() {
			It("should return an appropriate error", func() {
				handler := func(w http.ResponseWriter, r *http.Request) {
					defer GinkgoRecover()
					w.WriteHeader(400)
				}
				client.Host = httptest.NewServer(http.HandlerFunc(handler)).URL

				_, err := client.DownloadOVA("some-token", int64(0))
				Expect(err).To(MatchError("Pivotal Network returned: 400 Bad Request"))
			})
		})

		Context("when PivNet returns status 451", func() {
			It("should return an error telling user to agree to eula", func() {
				handler := func(w http.ResponseWriter, r *http.Request) {
					defer GinkgoRecover()
					w.WriteHeader(451)
				}
				client.Host = httptest.NewServer(http.HandlerFunc(handler)).URL

				_, err := client.DownloadOVA("some-token", int64(0))
				Expect(err).To(MatchError(MatchRegexp("you must accept the EULA before you can download the PCF Dev image: .*/products/pcfdev#/releases/some-release-id")))
			})
		})

		Context("when PivNet returns status 401", func() {
			It("should return an error telling user that their pivnet token is bad", func() {
				handler := func(w http.ResponseWriter, r *http.Request) {
					defer GinkgoRecover()
					w.WriteHeader(401)
				}
				client.Host = httptest.NewServer(http.HandlerFunc(handler)).URL

				_, err := client.DownloadOVA("some-token", int64(0))
				Expect(err).To(MatchError(MatchRegexp("invalid Pivotal Network API token")))
			})
		})
	})
})

package pivnet_test

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	"github.com/golang/mock/gomock"
	"github.com/pivotal-cf/pcfdev-cli/pivnet"
	"github.com/pivotal-cf/pcfdev-cli/pivnet/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Pivnet Client", func() {
	var (
		client    *pivnet.Client
		mockCtrl  *gomock.Controller
		mockToken *mocks.MockPivnetToken
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockToken = mocks.NewMockPivnetToken(mockCtrl)
		client = &pivnet.Client{
			ReleaseId:     "some-release-id",
			ProductFileId: "some-product-file-id",
			Token:         mockToken,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("#DownloadOVA", func() {
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

				mockToken.EXPECT().Get().Return("some-token", nil)
				ova, err := client.DownloadOVA(int64(4))
				Expect(err).NotTo(HaveOccurred())
				Expect(ova.ExistingLength).To(Equal(int64(4)))
				Expect(ova.ContentLength).To(Equal(int64(12)))
				buf, err := ioutil.ReadAll(ova.ReadCloser)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(buf)).To(Equal("ova contents"))
			})

			It("should accept a 200 during download", func() {
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
						w.WriteHeader(200)
						w.Write([]byte("ova contents"))
					default:
						Fail("unexpected server request")
					}
				}
				client.Host = httptest.NewServer(http.HandlerFunc(handler)).URL

				mockToken.EXPECT().Get().Return("some-token", nil)
				ova, err := client.DownloadOVA(int64(4))
				Expect(err).NotTo(HaveOccurred())
				Expect(ova.ExistingLength).To(Equal(int64(4)))
				Expect(ova.ContentLength).To(Equal(int64(12)))
				buf, err := ioutil.ReadAll(ova.ReadCloser)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(buf)).To(Equal("ova contents"))
			})
		})

		Context("when Pivnet is unreachable", func() {
			It("should return an appropriate error", func() {
				client.Host = "some-bad-host"

				mockToken.EXPECT().Get().Return("some-token", nil)
				_, err := client.DownloadOVA(int64(0))
				Expect(err).To(MatchError(ContainSubstring("failed to reach Pivotal Network:")))
			})
		})

		Context("when getting a token returns an error", func() {
			It("should return an appropriate error", func() {
				client.Host = "some-bad-host"

				mockToken.EXPECT().Get().Return("some-token", errors.New("some-error"))
				_, err := client.DownloadOVA(int64(0))
				Expect(err).To(MatchError("some-error"))
			})
		})

		Context("when Pivnet returns status not OK", func() {
			It("should return an appropriate error", func() {
				handler := func(w http.ResponseWriter, r *http.Request) {
					defer GinkgoRecover()
					w.WriteHeader(400)
				}
				client.Host = httptest.NewServer(http.HandlerFunc(handler)).URL

				mockToken.EXPECT().Get().Return("some-token", nil)

				_, err := client.DownloadOVA(int64(0))
				Expect(err).To(MatchError(ContainSubstring("Pivotal Network returned:")))
			})
		})

		Context("when Pivnet returns status 401", func() {
			It("should return an error telling user that their pivnet token is bad", func() {
				handler := func(w http.ResponseWriter, r *http.Request) {
					defer GinkgoRecover()
					w.WriteHeader(401)
				}
				client.Host = httptest.NewServer(http.HandlerFunc(handler)).URL

				gomock.InOrder(
					mockToken.EXPECT().Get().Return("some-bad-token", nil),
					mockToken.EXPECT().Destroy(),
				)
				_, err := client.DownloadOVA(int64(0))
				Expect(err).To(MatchError(MatchRegexp("invalid Pivotal Network API token")))
			})
		})
	})

	Describe("#AcceptEULA", func() {
		It("should accept the EULA", func() {
			handler := func(w http.ResponseWriter, r *http.Request) {
				defer GinkgoRecover()

				switch r.URL.Path {
				case "/api/v2/products/pcfdev/releases/some-release-id/product_files/some-product-file-id/download":
					Expect(r.Method).To(Equal("POST"))
					Expect(r.Header["Authorization"][0]).To(Equal("Token some-token"))
					w.Header().Add("Location", "http://"+r.Host+"/some-path")
					w.WriteHeader(451)
					w.Write([]byte(`{"_links":{"eula_agreement":{"href":"http://` + r.Host + `/api/v2/products/some-product/releases/some-release/eula_acceptance"}}}`))
				case "/api/v2/products/some-product/releases/some-release/eula_acceptance":
					Expect(r.Method).To(Equal("POST"))
					w.WriteHeader(200)
				default:
					Fail("unexpected server request")
				}
			}
			client.Host = httptest.NewServer(http.HandlerFunc(handler)).URL
			mockToken.EXPECT().Get().Return("some-token", nil).Times(2)

			Expect(client.AcceptEULA()).To(Succeed())
		})

		Context("when getting a token returns an error", func() {
			It("should return an appropriate error", func() {
				client.Host = "some-bad-host"

				mockToken.EXPECT().Get().Return("some-token", errors.New("some-error"))

				Expect(client.AcceptEULA()).To(MatchError(ContainSubstring("failed to reach Pivotal Network:")))
			})
		})

		Context("when unmarshalling the EULA fails", func() {
			It("should return the error", func() {
				handler := func(w http.ResponseWriter, r *http.Request) {
					defer GinkgoRecover()

					switch r.URL.Path {
					case "/api/v2/products/pcfdev/releases/some-release-id/product_files/some-product-file-id/download":
						Expect(r.Method).To(Equal("POST"))
						Expect(r.Header["Authorization"][0]).To(Equal("Token some-token"))
						w.Header().Add("Location", "http://"+r.Host+"/some-path")
						w.WriteHeader(451)
						w.Write([]byte(`some-non-json-response`))
					default:
						Fail("unexpected server request")
					}
				}
				client.Host = httptest.NewServer(http.HandlerFunc(handler)).URL

				mockToken.EXPECT().Get().Return("some-token", nil)

				Expect(client.AcceptEULA()).To(MatchError(ContainSubstring("failed to parse network response:")))
			})
		})

		Context("when authentication to Pivotal Network fails", func() {
			It("should return the error", func() {
				handler := func(w http.ResponseWriter, r *http.Request) {
					defer GinkgoRecover()
					w.WriteHeader(401)
				}
				client.Host = httptest.NewServer(http.HandlerFunc(handler)).URL

				gomock.InOrder(
					mockToken.EXPECT().Get().Return("some-bad-token", nil),
					mockToken.EXPECT().Destroy(),
				)
				Expect(client.AcceptEULA()).To(MatchError("invalid Pivotal Network API token"))
			})
		})

		Context("when Pivotal Network returns something unexpected", func() {
			It("should return the error", func() {
				handler := func(w http.ResponseWriter, r *http.Request) {
					defer GinkgoRecover()
					w.WriteHeader(501)
				}
				client.Host = httptest.NewServer(http.HandlerFunc(handler)).URL
				mockToken.EXPECT().Get().Return("some-token", nil)

				Expect(client.AcceptEULA()).To(MatchError("Pivotal Network returned: 501 Not Implemented"))
			})
		})

		Context("when network request to request ova fails", func() {
			It("should return the error", func() {
				client.Host = "some-bad-host"
				mockToken.EXPECT().Get().Return("some-token", nil)

				Expect(client.AcceptEULA()).To(MatchError(ContainSubstring("failed to reach Pivotal Network:")))
			})
		})

		Context("when network request to accept EULA fails", func() {
			It("should return the error", func() {
				handler := func(w http.ResponseWriter, r *http.Request) {
					defer GinkgoRecover()

					switch r.URL.Path {
					case "/api/v2/products/pcfdev/releases/some-release-id/product_files/some-product-file-id/download":
						Expect(r.Method).To(Equal("POST"))
						Expect(r.Header["Authorization"][0]).To(Equal("Token some-token"))
						w.Header().Add("Location", "http://"+r.Host+"/some-path")
						w.WriteHeader(451)
						w.Write([]byte(`{"_links":{"eula_agreement":{"href":"http://some-bad-host/api/v2/products/some-product/releases/some-release/eula_acceptance"}}}`))
					default:
						Fail("unexpected server request")
					}
				}
				client.Host = httptest.NewServer(http.HandlerFunc(handler)).URL
				mockToken.EXPECT().Get().Return("some-token", nil).Times(2)

				Expect(client.AcceptEULA()).To(MatchError(ContainSubstring("failed to reach Pivotal Network:")))
			})
		})

		Context("when authentication for EULA acceptance fails", func() {
			It("should return an error", func() {
				handler := func(w http.ResponseWriter, r *http.Request) {
					defer GinkgoRecover()

					switch r.URL.Path {
					case "/api/v2/products/pcfdev/releases/some-release-id/product_files/some-product-file-id/download":
						Expect(r.Method).To(Equal("POST"))
						Expect(r.Header["Authorization"][0]).To(Equal("Token some-token"))
						w.Header().Add("Location", "http://"+r.Host+"/some-path")
						w.WriteHeader(451)
						w.Write([]byte(`{"_links":{"eula_agreement":{"href":"http://` + r.Host + `/api/v2/products/some-product/releases/some-release/eula_acceptance"}}}`))
					case "/api/v2/products/some-product/releases/some-release/eula_acceptance":
						Expect(r.Method).To(Equal("POST"))
						w.WriteHeader(401)
					default:
						Fail("unexpected server request")
					}
				}
				client.Host = httptest.NewServer(http.HandlerFunc(handler)).URL

				gomock.InOrder(
					mockToken.EXPECT().Get().Return("some-token", nil).Times(2),
					mockToken.EXPECT().Destroy(),
				)
				Expect(client.AcceptEULA()).To(MatchError("invalid Pivotal Network API token"))
			})
		})

		Context("when Pivotal Network returns something unexpected for EULA acceptance", func() {
			It("should return an error", func() {
				handler := func(w http.ResponseWriter, r *http.Request) {
					defer GinkgoRecover()

					switch r.URL.Path {
					case "/api/v2/products/pcfdev/releases/some-release-id/product_files/some-product-file-id/download":
						Expect(r.Method).To(Equal("POST"))
						Expect(r.Header["Authorization"][0]).To(Equal("Token some-token"))
						w.Header().Add("Location", "http://"+r.Host+"/some-path")
						w.WriteHeader(451)
						w.Write([]byte(`{"_links":{"eula_agreement":{"href":"http://` + r.Host + `/api/v2/products/some-product/releases/some-release/eula_acceptance"}}}`))
					case "/api/v2/products/some-product/releases/some-release/eula_acceptance":
						Expect(r.Method).To(Equal("POST"))
						w.WriteHeader(500)
					default:
						Fail("unexpected server request")
					}
				}
				client.Host = httptest.NewServer(http.HandlerFunc(handler)).URL
				mockToken.EXPECT().Get().Return("some-token", nil).Times(2)

				Expect(client.AcceptEULA()).To(MatchError("Pivotal Network returned: 500 Internal Server Error"))
			})
		})
	})

	Describe("#IsEULAAccepted", func() {
		Context("when EULA has been accepted", func() {
			It("should return true", func() {
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
						Expect(r.Header["Range"][0]).To(Equal("bytes=0-0"))
						w.WriteHeader(206)
						w.Write([]byte("o"))
					default:
						Fail("unexpected server request")
					}
				}
				client.Host = httptest.NewServer(http.HandlerFunc(handler)).URL
				mockToken.EXPECT().Get().Return("some-token", nil)
				Expect(client.IsEULAAccepted()).To(BeTrue())
			})
		})

		Context("when EULA returns a 200", func() {
			It("should return true", func() {
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
						Expect(r.Header["Range"][0]).To(Equal("bytes=0-0"))
						w.WriteHeader(200)
						w.Write([]byte("o"))
					default:
						Fail("unexpected server request")
					}
				}
				client.Host = httptest.NewServer(http.HandlerFunc(handler)).URL
				mockToken.EXPECT().Get().Return("some-token", nil)
				Expect(client.IsEULAAccepted()).To(BeTrue())
			})
		})

		Context("when getting the token returns an error", func() {
			It("should return the error", func() {
				mockToken.EXPECT().Get().Return("some-token", errors.New("some-error"))
				_, err := client.IsEULAAccepted()
				Expect(err).To(MatchError("some-error"))
			})
		})

		Context("when getting the EULA returns an unauthorized error", func() {
			It("should return the error and delete the token file", func() {
				handler := func(w http.ResponseWriter, r *http.Request) {
					defer GinkgoRecover()
					w.WriteHeader(401)
				}

				client.Host = httptest.NewServer(http.HandlerFunc(handler)).URL
				mockToken.EXPECT().Get().Return("some-bad-token", nil)
				mockToken.EXPECT().Destroy()
				_, err := client.IsEULAAccepted()
				Expect(err).To(MatchError(&pivnet.InvalidTokenError{}))
			})
		})

		Context("when EULA has not been accepted", func() {
			It("should return false", func() {
				handler := func(w http.ResponseWriter, r *http.Request) {
					defer GinkgoRecover()
					w.WriteHeader(451)
				}

				client.Host = httptest.NewServer(http.HandlerFunc(handler)).URL
				mockToken.EXPECT().Get().Return("some-token", nil)
				Expect(client.IsEULAAccepted()).To(BeFalse())
			})
		})
	})

	Describe("#GetEULA", func() {
		It("should return EULA", func() {
			handler := func(w http.ResponseWriter, r *http.Request) {
				defer GinkgoRecover()

				switch r.URL.Path {
				case "/api/v2/products/pcfdev/releases/some-release-id":
					Expect(r.Method).To(Equal("GET"))
					Expect(r.Header["Authorization"][0]).To(Equal("Token some-token"))
					w.WriteHeader(200)
					w.Write([]byte(`{"eula": {"_links": {"self": {"href": "http://` + r.Host + `/some-eula-path"}}}}`))
				case "/some-eula-path":
					Expect(r.Method).To(Equal("GET"))
					w.WriteHeader(200)
					w.Write([]byte(`{"content": "<p>some-eula-text</p>"}`))
				default:
					Fail("unexpected server request")
				}
			}
			client.Host = httptest.NewServer(http.HandlerFunc(handler)).URL
			mockToken.EXPECT().Get().Return("some-token", nil).Times(2)
			Expect(client.GetEULA()).To(Equal("some-eula-text\n"))
		})

		Context("when getting the token returns an error", func() {
			It("should return the error", func() {
				mockToken.EXPECT().Get().Return("some-token", errors.New("some-error"))
				_, err := client.GetEULA()
				Expect(err).To(MatchError(ContainSubstring("failed to reach Pivotal Network:")))
			})
		})

		Context("when it fails to query release", func() {
			It("should return an error", func() {
				client.Host = "some-bad-host"

				mockToken.EXPECT().Get().Return("some-token", nil)
				_, err := client.GetEULA()
				Expect(err).To(MatchError(ContainSubstring("failed to reach Pivotal Network:")))
			})
		})

		Context("when the release request returns 401", func() {
			It("should return an auth error", func() {
				handler := func(w http.ResponseWriter, r *http.Request) {
					defer GinkgoRecover()
					w.WriteHeader(401)
				}
				client.Host = httptest.NewServer(http.HandlerFunc(handler)).URL

				gomock.InOrder(
					mockToken.EXPECT().Get().Return("some-bad-token", nil),
					mockToken.EXPECT().Destroy(),
				)
				_, err := client.GetEULA()
				Expect(err).To(MatchError(MatchRegexp("invalid Pivotal Network API token")))
			})
		})

		Context("when it release does not return 200 or 451", func() {
			It("should return an error", func() {
				handler := func(w http.ResponseWriter, r *http.Request) {
					defer GinkgoRecover()
					w.WriteHeader(400)
				}
				client.Host = httptest.NewServer(http.HandlerFunc(handler)).URL
				mockToken.EXPECT().Get().Return("some-token", nil)
				_, err := client.GetEULA()
				Expect(err).To(MatchError("Pivotal Network returned: 400 Bad Request"))
			})
		})

		Context("when it fails to unmarshal release", func() {
			It("should return an error", func() {
				handler := func(w http.ResponseWriter, r *http.Request) {
					defer GinkgoRecover()
					w.WriteHeader(200)
					w.Write([]byte(`some-bad-json`))
				}
				client.Host = httptest.NewServer(http.HandlerFunc(handler)).URL
				mockToken.EXPECT().Get().Return("some-token", nil)
				_, err := client.GetEULA()
				Expect(err).To(MatchError(ContainSubstring("failed to parse network response:")))
			})
		})

		Context("when it fails to query the EULA", func() {
			It("should return an auth error", func() {
				handler := func(w http.ResponseWriter, r *http.Request) {
					defer GinkgoRecover()
					switch r.URL.Path {
					case "/api/v2/products/pcfdev/releases/some-release-id":
						w.WriteHeader(200)
						w.Write([]byte(`{"eula": {"_links": {"self": {"href": "http://some-bad-host/some-eula-path"}}}}`))
					default:
						Fail("unexpected server request")
					}
				}
				client.Host = httptest.NewServer(http.HandlerFunc(handler)).URL
				mockToken.EXPECT().Get().Return("some-token", nil).Times(2)
				_, err := client.GetEULA()
				Expect(err).To(MatchError(ContainSubstring("failed to reach Pivotal Network:")))
			})
		})

		Context("when EULA request returns 401", func() {
			It("should return an auth error", func() {
				handler := func(w http.ResponseWriter, r *http.Request) {
					defer GinkgoRecover()
					switch r.URL.Path {
					case "/api/v2/products/pcfdev/releases/some-release-id":
						w.WriteHeader(200)
						w.Write([]byte(`{"eula": {"_links": {"self": {"href": "http://` + r.Host + `/some-eula-path"}}}}`))
					case "/some-eula-path":
						w.WriteHeader(401)
					default:
						Fail("unexpected server request")
					}
				}
				client.Host = httptest.NewServer(http.HandlerFunc(handler)).URL

				gomock.InOrder(
					mockToken.EXPECT().Get().Return("some-token", nil).Times(2),
					mockToken.EXPECT().Destroy(),
				)
				_, err := client.GetEULA()
				Expect(err).To(MatchError(MatchRegexp("invalid Pivotal Network API token")))
			})
		})

		Context("when it EULA request does not return 200 or 451", func() {
			It("should return an error", func() {
				handler := func(w http.ResponseWriter, r *http.Request) {
					defer GinkgoRecover()
					switch r.URL.Path {
					case "/api/v2/products/pcfdev/releases/some-release-id":
						w.WriteHeader(200)
						w.Write([]byte(`{"eula": {"_links": {"self": {"href": "http://` + r.Host + `/some-eula-path"}}}}`))
					case "/some-eula-path":
						w.WriteHeader(400)
					default:
						Fail("unexpected server request")
					}
				}
				client.Host = httptest.NewServer(http.HandlerFunc(handler)).URL
				mockToken.EXPECT().Get().Return("some-token", nil).Times(2)
				_, err := client.GetEULA()
				Expect(err).To(MatchError("Pivotal Network returned: 400 Bad Request"))
			})
		})

		Context("when it fails to unmarshal EULA", func() {
			It("should return an error", func() {
				handler := func(w http.ResponseWriter, r *http.Request) {
					defer GinkgoRecover()

					switch r.URL.Path {
					case "/api/v2/products/pcfdev/releases/some-release-id":
						Expect(r.Method).To(Equal("GET"))
						Expect(r.Header["Authorization"][0]).To(Equal("Token some-token"))
						w.WriteHeader(200)
						w.Write([]byte(`{"eula": {"_links": {"self": {"href": "http://` + r.Host + `/some-eula-path"}}}}`))
					case "/some-eula-path":
						Expect(r.Method).To(Equal("GET"))
						w.WriteHeader(200)
						w.Write([]byte(`{some-bad-json`))
					default:
						Fail("unexpected server request")
					}
				}
				client.Host = httptest.NewServer(http.HandlerFunc(handler)).URL
				mockToken.EXPECT().Get().Return("some-token", nil).Times(2)
				_, err := client.GetEULA()
				Expect(err).To(MatchError(ContainSubstring("failed to parse network response:")))
			})
		})
	})

	Describe("#GetToken", func() {
		It("should return a token", func() {
			handler := func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/v2/api_token" {
					Expect(r.Method).To(Equal("GET"))
					Expect(r.Header["User-Agent"][0]).To(Equal("PCF-Dev-client"))
					Expect(r.URL.RawQuery).To(Equal("password=some-password&username=some-username"))

					w.WriteHeader(200)
					w.Write([]byte(`{"api_token": "some-token"}`))
				} else {
					Fail("unexpected server request")
				}
			}
			client.Host = httptest.NewServer(http.HandlerFunc(handler)).URL

			Expect(client.GetToken("some-username", "some-password")).To(Equal("some-token"))
		})

		Context("when there is an error making the request", func() {
			It("should return an error", func() {
				client.Host = "some-bad-protocol-scheme://"

				_, err := client.GetToken("some-username", "some-password")
				Expect(err).To(MatchError(ContainSubstring("unsupported protocol scheme")))
			})
		})

		Context("when Pivnet returns a non-200 status", func() {
			It("should return an error", func() {
				handler := func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/api/v2/api_token" {
						w.WriteHeader(401)
					} else {
						Fail("unexpected server request")
					}
				}
				client.Host = httptest.NewServer(http.HandlerFunc(handler)).URL

				_, err := client.GetToken("some-bad-username", "some-bad-password")
				Expect(err).To(MatchError("Pivotal Network returned: 401 Unauthorized"))
			})
		})

		Context("when there is an parsing the body", func() {
			It("should return an error", func() {
				handler := func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/api/v2/api_token" {
						w.WriteHeader(200)
						w.Write([]byte(""))
					} else {
						Fail("unexpected server request")
					}
				}
				client.Host = httptest.NewServer(http.HandlerFunc(handler)).URL

				_, err := client.GetToken("some-username", "some-password")
				Expect(err).To(MatchError("unexpected end of JSON input"))
			})
		})
	})
})

package client_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	c "github.com/pivotal-cf/pcfdev-cli/vm/client"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"time"
)

var _ = Describe("Pivnet Client", func() {
	var (
		client *c.Client
	)

	BeforeEach(func() {
		client = &c.Client{
			Timeout: time.Millisecond,
			HttpClient: http.DefaultClient,
		}
	})

	Describe("#Status", func() {
		It("return the status of the VM", func() {
			handler := func(w http.ResponseWriter, r *http.Request) {
				defer GinkgoRecover()

				switch r.URL.Path {
				case "/status":
					Expect(r.Method).To(Equal("GET"))
					w.WriteHeader(200)
					w.Write([]byte(`{"status":"some-status"}`))
				case "/":
					w.WriteHeader(200)
				default:
					Fail("unexpected server request")
				}
			}

			host := httptest.NewServer(http.HandlerFunc(handler)).URL
			Expect(client.Status(host)).To(Equal("some-status"))
		})

		Context("when doing a bad get request", func() {
			It("return an error", func() {
				host := "http://some-bad-host"
				_, err := client.Status(host)
				Expect(err).To(MatchError(ContainSubstring("failed to talk to PCF Dev VM:")))
			})
		})

		Context("when there is invalid JSON", func() {
			It("returns an error", func() {
				handler := func(w http.ResponseWriter, r *http.Request) {
					defer GinkgoRecover()

					switch r.URL.Path {
					case "/status":
						Expect(r.Method).To(Equal("GET"))
						w.WriteHeader(200)
						w.Write([]byte(`some-bad-json`))
					case "/":
						w.WriteHeader(200)
					default:
						Fail("unexpected server request")
					}
				}
				host := httptest.NewServer(http.HandlerFunc(handler)).URL
				_, err := client.Status(host)
				Expect(err).To(MatchError(ContainSubstring("failed to parse JSON response:")))

			})
		})

		Context("when it fails to retrieve status", func() {
			It("returns an error", func() {
				handler := func(w http.ResponseWriter, r *http.Request) {
					defer GinkgoRecover()

					switch r.URL.Path {
					case "/status":
						Expect(r.Method).To(Equal("GET"))
						w.WriteHeader(500)
					case "/":
						w.WriteHeader(200)
					default:
						Fail("unexpected server request")
					}
				}
				host := httptest.NewServer(http.HandlerFunc(handler)).URL
				_, err := client.Status(host)
				Expect(err).To(MatchError(ContainSubstring("failed to retrieve status:")))

			})
		})
	})

	Describe("#ReplaceSecrets", func() {
		It("should replace secrets on the VM", func() {
			handler := func(w http.ResponseWriter, r *http.Request) {
				defer GinkgoRecover()

				switch r.URL.Path {
				case "/replace-secrets":
					Expect(r.Method).To(Equal("PUT"))
					Expect(ioutil.ReadAll(r.Body)).To(Equal([]byte(`{"password":"some-master-password"}`)))
					w.WriteHeader(200)
				case "/":
					w.WriteHeader(200)
				default:
					Fail("unexpected server request")
				}
			}

			host := httptest.NewServer(http.HandlerFunc(handler)).URL
			Expect(client.ReplaceSecrets(host, "some-master-password")).To(Succeed())
		})

		Context("when there is a bad response from the api", func() {
			It("should return an error", func() {
				host := "http://some-bad-host"
				Expect(client.ReplaceSecrets(host, "some-master-password")).To(MatchError(ContainSubstring("failed to talk to PCF Dev VM:")))
			})
		})

		Context("when there is no response from the api", func() {
			It("should return an error", func() {
				handler := func(w http.ResponseWriter, r *http.Request) {
					defer GinkgoRecover()

					switch r.URL.Path {
					default:
						w.WriteHeader(500)
					}
				}

				host := httptest.NewServer(http.HandlerFunc(handler)).URL
				Expect(client.ReplaceSecrets(host, "some-master-password")).To(MatchError(ContainSubstring("failed to replace master password:")))
			})
		})
	})
})

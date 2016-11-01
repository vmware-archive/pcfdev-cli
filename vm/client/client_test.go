package client_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	c "github.com/pivotal-cf/pcfdev-cli/vm/client"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
)

var _ = Describe("Pivnet Client", func() {
	var (
		client *c.Client
	)

	BeforeEach(func() {
		client = &c.Client{}
	})

	AfterEach(func() {
		//mockCtrl.Finish()
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
				default:
					Fail("unexpected server request")
				}
			}

			client.Host = httptest.NewServer(http.HandlerFunc(handler)).URL
			Expect(client.ReplaceSecrets("some-master-password")).To(Succeed())
		})

		Context("when there is a bad response from the api", func() {
			It("should return an error", func() {
				client.Host = "http://example.com/some-bad-host"
				Expect(client.ReplaceSecrets("some-master-password")).To(MatchError(ContainSubstring("failed to talk to PCF Dev VM:")))
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

				client.Host = httptest.NewServer(http.HandlerFunc(handler)).URL
				Expect(client.ReplaceSecrets("some-master-password")).To(MatchError(ContainSubstring("failed to replace master password:")))
			})
		})
	})
})

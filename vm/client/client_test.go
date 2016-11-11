package client_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"time"

	"errors"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/pivotal-cf/pcfdev-cli/ssh"
	c "github.com/pivotal-cf/pcfdev-cli/vm/client"
	"github.com/pivotal-cf/pcfdev-cli/vm/client/mocks"
)

var _ = Describe("PCF Dev Client", func() {
	var (
		mockCtrl *gomock.Controller
		mockSSH  *mocks.MockSSH
		client   *c.Client
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockSSH = mocks.NewMockSSH(mockCtrl)
		client = &c.Client{
			Timeout:    time.Millisecond,
			HttpClient: http.DefaultClient,
			SSHClient:  mockSSH,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
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

			mockSSH.EXPECT().WithSSHTunnel(
				fmt.Sprintf("127.0.0.1:%d", c.APIPort),
				[]ssh.SSHAddress{{IP: "some-ip", Port: "22"}},
				[]byte("some-private-key"),
				time.Minute,
				gomock.Any(),
			).Do(func(_ string, _ []ssh.SSHAddress, _ []byte, _ time.Duration, block func(string)) {
				block(host)
			})

			status, err := client.Status("some-ip", []byte("some-private-key"))
			Expect(err).NotTo(HaveOccurred())
			Expect(status).To(Equal("some-status"))
		})

		Context("when doing a bad get request", func() {
			It("return an error", func() {
				host := "http://some-bad-host"

				mockSSH.EXPECT().WithSSHTunnel(
					fmt.Sprintf("127.0.0.1:%d", c.APIPort),
					[]ssh.SSHAddress{{IP: "some-ip", Port: "22"}},
					[]byte("some-private-key"),
					time.Minute,
					gomock.Any(),
				).Do(func(_ string, _ []ssh.SSHAddress, _ []byte, _ time.Duration, block func(string)) {
					block(host)
				})

				_, err := client.Status("some-ip", []byte("some-private-key"))
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

				mockSSH.EXPECT().WithSSHTunnel(
					fmt.Sprintf("127.0.0.1:%d", c.APIPort),
					[]ssh.SSHAddress{{IP: "some-ip", Port: "22"}},
					[]byte("some-private-key"),
					time.Minute,
					gomock.Any(),
				).Do(func(_ string, _ []ssh.SSHAddress, _ []byte, _ time.Duration, block func(string)) {
					block(host)
				})

				_, err := client.Status("some-ip", []byte("some-private-key"))
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

				mockSSH.EXPECT().WithSSHTunnel(
					fmt.Sprintf("127.0.0.1:%d", c.APIPort),
					[]ssh.SSHAddress{{IP: "some-ip", Port: "22"}},
					[]byte("some-private-key"),
					time.Minute,
					gomock.Any(),
				).Do(func(_ string, _ []ssh.SSHAddress, _ []byte, _ time.Duration, block func(string)) {
					block(host)
				})

				_, err := client.Status("some-ip", []byte("some-private-key"))
				Expect(err).To(MatchError("failed to retrieve status: PCF Dev API returned: 500"))
			})
		})

		Context("when there is an error establishing the SSH tunnel", func() {
			It("should return the error", func() {
				mockSSH.EXPECT().WithSSHTunnel(
					fmt.Sprintf("127.0.0.1:%d", c.APIPort),
					[]ssh.SSHAddress{{IP: "some-ip", Port: "22"}},
					[]byte("some-private-key"),
					time.Minute,
					gomock.Any(),
				).Return(errors.New("some-error"))

				_, err := client.Status("some-ip", []byte("some-private-key"))
				Expect(err).To(MatchError("some-error"))
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

			mockSSH.EXPECT().WithSSHTunnel(
				fmt.Sprintf("127.0.0.1:%d", c.APIPort),
				[]ssh.SSHAddress{{IP: "some-ip", Port: "22"}},
				[]byte("some-private-key"),
				time.Minute,
				gomock.Any(),
			).Do(func(_ string, _ []ssh.SSHAddress, _ []byte, _ time.Duration, block func(string)) {
				block(host)
			})

			Expect(client.ReplaceSecrets("some-ip", "some-master-password", []byte("some-private-key"))).To(Succeed())
		})

		Context("when there is a bad response from the api", func() {
			It("should return an error", func() {
				host := "http://some-bad-host"

				mockSSH.EXPECT().WithSSHTunnel(
					fmt.Sprintf("127.0.0.1:%d", c.APIPort),
					[]ssh.SSHAddress{{IP: "some-ip", Port: "22"}},
					[]byte("some-private-key"),
					time.Minute,
					gomock.Any(),
				).Do(func(_ string, _ []ssh.SSHAddress, _ []byte, _ time.Duration, block func(string)) {
					block(host)
				})

				Expect(client.ReplaceSecrets("some-ip", "some-master-password", []byte("some-private-key"))).To(MatchError(ContainSubstring("failed to talk to PCF Dev VM:")))
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

				mockSSH.EXPECT().WithSSHTunnel(
					fmt.Sprintf("127.0.0.1:%d", c.APIPort),
					[]ssh.SSHAddress{{IP: "some-ip", Port: "22"}},
					[]byte("some-private-key"),
					time.Minute,
					gomock.Any(),
				).Do(func(_ string, _ []ssh.SSHAddress, _ []byte, _ time.Duration, block func(string)) {
					block(host)
				})

				Expect(client.ReplaceSecrets("some-ip", "some-master-password", []byte("some-private-key"))).To(MatchError("failed to replace master password: PCF Dev API returned: 500"))
			})
		})

		Context("when there is an error establishing the SSH tunnel", func() {
			It("should return the error", func() {
				mockSSH.EXPECT().WithSSHTunnel(
					fmt.Sprintf("127.0.0.1:%d", c.APIPort),
					[]ssh.SSHAddress{{IP: "some-ip", Port: "22"}},
					[]byte("some-private-key"),
					time.Minute,
					gomock.Any(),
				).Return(errors.New("some-error"))

				Expect(client.ReplaceSecrets("some-ip", "some-master-password", []byte("some-private-key"))).To(MatchError("some-error"))
			})
		})
	})
})

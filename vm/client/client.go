package client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/pivotal-cf/pcfdev-cli/ssh"
)

//go:generate mockgen -package mocks -destination mocks/ssh.go github.com/pivotal-cf/pcfdev-cli/vm/client SSH
type SSH interface {
	WithSSHTunnel(remoteAddress string, sshAddresses []ssh.SSHAddress, privateKey []byte, timeout time.Duration, block func(forwardingAddress string)) error
}

type Client struct {
	Timeout    time.Duration
	HttpClient *http.Client
	SSHClient  SSH
}

type StatusResponse struct {
	Status string `json:"status"`
}

const APIPort = 8090

func (c *Client) Status(sshIP string, privateKey []byte) (string, error) {
	var resp *http.Response
	var errorInTunnel error
	errorWithTunnel := c.SSHClient.WithSSHTunnel(
		fmt.Sprintf("127.0.0.1:%d", APIPort),
		[]ssh.SSHAddress{{IP: sshIP, Port: "22"}},
		privateKey,
		time.Minute,
		func(host string) {
			canPing := c.waitForPing(host)
			if !canPing {
				errorInTunnel = &PCFDevVmUnreachableError{}
				return
			}

			var err error
			resp, err = c.HttpClient.Get(fmt.Sprintf("%s/status", host))
			if err != nil {
				errorInTunnel = &PCFDevVmUnreachableError{err}
			}
		},
	)

	if errorWithTunnel != nil {
		return "", errorWithTunnel
	}

	if errorInTunnel != nil {
		return "", errorInTunnel
	}

	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}

		statusResponse := &StatusResponse{}
		if err := json.Unmarshal(data, statusResponse); err != nil {
			return "", &InvalidJSONError{err}
		}

		return statusResponse.Status, nil
	default:
		return "", &StatusRetrievalError{fmt.Errorf("PCF Dev API returned: %d", resp.StatusCode)}
	}
}

func (c *Client) ReplaceSecrets(sshIP, password string, privateKey []byte) error {
	var resp *http.Response
	var errorInTunnel error
	errorWithTunnel := c.SSHClient.WithSSHTunnel(
		fmt.Sprintf("127.0.0.1:%d", APIPort),
		[]ssh.SSHAddress{{IP: sshIP, Port: "22"}},
		privateKey,
		time.Minute,
		func(host string) {
			uri := fmt.Sprintf("%s/replace-secrets", host)

			body := strings.NewReader(fmt.Sprintf(`{"password":"%s"}`, password))
			req, err := http.NewRequest("PUT", uri, body)
			if err != nil {
				errorInTunnel = err
				return
			}

			resp, err = http.DefaultClient.Do(req)
			if err != nil {
				errorInTunnel = &PCFDevVmUnreachableError{err}
			}
		},
	)

	if errorWithTunnel != nil {
		return errorWithTunnel
	}

	if errorInTunnel != nil {
		return errorInTunnel
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	default:
		return &ReplaceMasterPasswordError{fmt.Errorf("PCF Dev API returned: %d", resp.StatusCode)}
	}
}

type PCFDevVmUnreachableError struct {
	Err error
}

func (e *PCFDevVmUnreachableError) Error() string {
	return fmt.Sprintf("failed to talk to PCF Dev VM: %s", e.Err)
}

type StatusRetrievalError struct {
	Err error
}

func (e *StatusRetrievalError) Error() string {
	return fmt.Sprintf("failed to retrieve status: %s", e.Err)
}

type InvalidJSONError struct {
	Err error
}

func (e *InvalidJSONError) Error() string {
	return fmt.Sprintf("failed to parse JSON response: %s", e.Err)
}

type ReplaceMasterPasswordError struct {
	Err error
}

func (e *ReplaceMasterPasswordError) Error() string {
	return fmt.Sprintf("failed to replace master password: %s", e.Err)
}

func (c *Client) waitForPing(host string) bool {
	pingChannel := make(chan bool)
	timeoutChannel := time.After(c.Timeout)
	go func() {
		for {
			select {
			case <-timeoutChannel:
				pingChannel <- false
				return
			default:
				if resp, err := c.HttpClient.Get(host); err == nil {
					resp.Body.Close()
					pingChannel <- true
					return
				}
				time.Sleep(10 * time.Millisecond)
			}
		}
	}()
	return <-pingChannel
}

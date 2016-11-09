package client

import (
	"fmt"
	"net/http"
	"strings"
	"io/ioutil"
	"encoding/json"
	"time"
)

type Client struct {
	Timeout time.Duration
	HttpClient *http.Client
}

type StatusResponse struct {
	Status string `json:"status"`
}

const APIPort  = 8090

func (c *Client) Status(host string) (string, error) {
	canPing := c.waitForPing(host)
	if !canPing {
		return "", &PCFDevVmUnreachableError{}
	}

	uri := fmt.Sprintf("%s/status", host)

	req, err := http.NewRequest("GET", uri, nil)

	if err != nil {
		return "", err
	}

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return "", &PCFDevVmUnreachableError{err}
	}

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
		return "", &StatusRetrievalError{err}
	}
}

func (c *Client) ReplaceSecrets(host, password string) error {
	uri := fmt.Sprintf("%s/replace-secrets", host)

	body := strings.NewReader(fmt.Sprintf(`{"password":"%s"}`, password))
	req, err := http.NewRequest("PUT", uri, body)

	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return &PCFDevVmUnreachableError{err}
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	default:
		return &ReplaceMasterPasswordError{err}
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
				if _, err := c.HttpClient.Get(host); err == nil {
					pingChannel <- true
					return
				}
				time.Sleep(10 * time.Millisecond)
			}
		}
	}()
	return <-pingChannel
}

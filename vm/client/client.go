package client

import (
	"net/http"
	"fmt"
	"strings"
)

type Client struct {
	Host string
}

func (c *Client) ReplaceSecrets(password string) error {
	uri := fmt.Sprintf("%s/replace-secrets", c.Host)

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

type ReplaceMasterPasswordError struct {
	Err error
}

func (e *ReplaceMasterPasswordError) Error() string {
	return fmt.Sprintf("failed to replace master password: %s", e.Err)
}

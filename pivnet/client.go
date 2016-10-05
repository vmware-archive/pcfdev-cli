package pivnet

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/kennygrant/sanitize"
)

//go:generate mockgen -package mocks -destination mocks/token.go github.com/pivotal-cf/pcfdev-cli/pivnet PivnetToken
type PivnetToken interface {
	Get() (token string, err error)
	Destroy() error
}

type Client struct {
	Token         PivnetToken
	Host          string
	ReleaseId     string
	ProductFileId string
}

type ReleaseResponse struct {
	EULAS struct {
		Links struct {
			Self struct {
				HREF string `json:"href"`
			} `json:"self"`
		} `json:"_links"`
	} `json:"eula"`
}

type EULAAcceptanceResponse struct {
	Links struct {
		Agreement struct {
			HREF string `json:"href"`
		} `json:"eula_agreement"`
	} `json:"_links"`
}

type EULAResponse struct {
	Content string `json:"content"`
}

func (c *Client) DownloadOVA(startAtByte int64) (ova *DownloadReader, err error) {
	resp, err := c.requestOva(fmt.Sprintf("bytes=%d-", startAtByte))
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusPartialContent, http.StatusOK:
		return &DownloadReader{ReadCloser: resp.Body, Writer: os.Stdout, ContentLength: resp.ContentLength, ExistingLength: startAtByte}, nil
	case http.StatusUnauthorized:
		c.Token.Destroy()
		return nil, &InvalidTokenError{}
	default:
		return nil, c.unexpectedResponseError(resp)
	}
}

func (c *Client) IsEULAAccepted() (bool, error) {
	resp, err := c.requestOva("bytes=0-0")
	if err != nil {
		return false, err
	}

	switch resp.StatusCode {
	case http.StatusPartialContent, http.StatusOK:
		return true, nil
	case 451:
		return false, nil
	case http.StatusUnauthorized:
		c.Token.Destroy()
		return false, &InvalidTokenError{}
	default:
		return false, c.unexpectedResponseError(resp)
	}
}

func (c *Client) GetEULA() (eula string, err error) {
	uri := fmt.Sprintf("%s/api/v2/products/pcfdev/releases/%s", c.Host, c.ReleaseId)
	resp, err := c.makeRequest(uri, "GET", http.DefaultClient)
	if err != nil {
		return "", &PivNetUnreachableError{err}
	}

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		c.Token.Destroy()
		return "", &InvalidTokenError{}
	case 200:
		break
	default:
		return "", c.unexpectedResponseError(resp)
	}

	releaseResponse := &ReleaseResponse{}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if err := json.Unmarshal(data, releaseResponse); err != nil {
		return "", &JSONUnmarshalError{err}
	}

	uri = fmt.Sprintf(releaseResponse.EULAS.Links.Self.HREF)
	resp, err = c.makeRequest(uri, "GET", http.DefaultClient)
	if err != nil {
		return "", &PivNetUnreachableError{err}
	}

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		c.Token.Destroy()
		return "", &InvalidTokenError{}
	case 200:
		break
	default:
		return "", c.unexpectedResponseError(resp)
	}

	eulaResponse := &EULAResponse{}

	response, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if err := json.Unmarshal(response, eulaResponse); err != nil {
		return "", &JSONUnmarshalError{err}
	}

	return sanitize.HTML(eulaResponse.Content), nil
}

func (c *Client) AcceptEULA() error {
	resp, err := c.requestOva("bytes=0-0")
	if err != nil {
		return &PivNetUnreachableError{err}
	}

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		c.Token.Destroy()
		return &InvalidTokenError{}
	case 451, 200:
		break
	default:
		return c.unexpectedResponseError(resp)
	}

	eulaAcceptanceResponse := &EULAAcceptanceResponse{}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, eulaAcceptanceResponse); err != nil {
		return &JSONUnmarshalError{err}
	}

	uri := fmt.Sprintf(eulaAcceptanceResponse.Links.Agreement.HREF)

	resp, err = c.makeRequest(uri, "POST", http.DefaultClient)
	if err != nil {
		return &PivNetUnreachableError{err}
	}

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		c.Token.Destroy()
		return &InvalidTokenError{}
	case 200:
		return nil
	default:
		return c.unexpectedResponseError(resp)
	}
}

func (c *Client) GetToken(username string, password string) (string, error) {
	req, err := http.NewRequest(
		"GET",
		c.Host+"/api/v2/api_token",
		nil,
	)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "PCF-Dev-client")

	query := req.URL.Query()
	query.Add("username", username)
	query.Add("password", password)
	req.URL.RawQuery = query.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", &InvalidCredentialsError{}
	}

	var apiToken struct {
		ApiToken string `json:"api_token"`
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if err := json.Unmarshal(body, &apiToken); err != nil {
		return "", err
	}

	return apiToken.ApiToken, nil
}

func (c *Client) unexpectedResponseError(resp *http.Response) error {
	return &UnexpectedResponseError{fmt.Errorf("Pivotal Network returned: %s", resp.Status)}
}

func (c *Client) requestOva(byteRange string) (*http.Response, error) {
	uri := fmt.Sprintf("%s/api/v2/products/pcfdev/releases/%s/product_files/%s/download", c.Host, c.ReleaseId, c.ProductFileId)
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			req.Header.Set("Range", byteRange)
			return nil
		},
	}
	return c.makeRequest(uri, "POST", client)
}

func (c *Client) makeRequest(uri string, method string, client *http.Client) (*http.Response, error) {
	req, err := http.NewRequest(method, uri, nil)
	if err != nil {
		return nil, err
	}

	token, err := c.Token.Get()
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Token "+token)

	resp, err := client.Do(req)
	if err != nil {
		return nil, &PivNetUnreachableError{err}
	}
	return resp, nil
}

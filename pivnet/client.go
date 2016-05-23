package pivnet

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/kennygrant/sanitize"
)

type Client struct {
	Host          string
	ReleaseId     string
	ProductFileId string
	Config        Config
}

//go:generate mockgen -package mocks -destination mocks/config.go github.com/pivotal-cf/pcfdev-cli/pivnet Config
type Config interface {
	GetToken() string
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
	case http.StatusPartialContent:
		return &DownloadReader{ReadCloser: resp.Body, Writer: os.Stdout, ContentLength: resp.ContentLength, ExistingLength: startAtByte}, nil
	case http.StatusUnauthorized:
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
	case http.StatusPartialContent:
		return true, nil
	case 451:
		return false, nil
	case http.StatusUnauthorized:
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
		return &InvalidTokenError{}
	case 200:
		return nil
	default:
		return c.unexpectedResponseError(resp)
	}
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
	req.Header.Set("Authorization", "Token "+c.Config.GetToken())

	resp, err := client.Do(req)
	if err != nil {
		return nil, &PivNetUnreachableError{err}
	}
	return resp, nil
}

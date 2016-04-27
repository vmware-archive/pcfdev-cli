package pivnet

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

type Client struct {
	Host   string
	Config Config
}

const (
	releaseId     = "1622"
	productFileId = "4149"
	productSlug   = "pcfdev"
)

//go:generate mockgen -package mocks -destination mocks/config.go github.com/pivotal-cf/pcfdev-cli/pivnet Config
type Config interface {
	GetToken() string
}

type ProductFile struct {
	MD5 string `json:"md5"`
}

func (c *Client) DownloadOVA() (io.ReadCloser, error) {
	return c.request("POST", fmt.Sprintf("/api/v2/products/%s/releases/%s/product_files/%s/download", productSlug, releaseId, productFileId))
}

func (c *Client) MD5() (string, error) {
	response, err := c.request("GET", fmt.Sprintf("/api/v2/products/%s/releases/%s/product_files/%s", productSlug, releaseId, productFileId))
	if err != nil {
		return "", err
	}

	productFile := &ProductFile{}
	data, err := ioutil.ReadAll(response)
	if err != nil {
		return "", fmt.Errorf("Unable to read response: %s", err)
	}

	err = json.Unmarshal(data, productFile)
	if err != nil {
		return "", fmt.Errorf("Unable to parse response: %s", err)
	}

	return productFile.MD5, nil
}

func (c *Client) request(method string, uri string) (io.ReadCloser, error) {
	req, err := http.NewRequest(method, c.Host+uri, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Token "+c.Config.GetToken())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to reach Pivotal Network: %s", err)
	}

	switch resp.StatusCode {
	case 200:
		return resp.Body, nil
	case 401:
		return nil, fmt.Errorf("invalid Pivotal Network API token")
	case 451:
		return nil, fmt.Errorf("you must accept the EULA before you can download the PCF Dev image: %s/products/pcfdev#/releases/1622", c.Host)
	default:
		return nil, fmt.Errorf("Pivotal Network returned: %s", resp.Status)
	}
}

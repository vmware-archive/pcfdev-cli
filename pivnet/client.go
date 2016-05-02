package pivnet

import (
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	Host string
}

const (
	releaseId     = "1622"
	productFileId = "4149"
	productSlug   = "pcfdev"
)

type ProductFile struct {
	MD5 string `json:"md5"`
}

func (c *Client) DownloadOVA(token string) (io.ReadCloser, error) {
	return c.request("POST", fmt.Sprintf("/api/v2/products/%s/releases/%s/product_files/%s/download", productSlug, releaseId, productFileId), token)
}

func (c *Client) request(method string, uri string, token string) (io.ReadCloser, error) {
	req, err := http.NewRequest(method, c.Host+uri, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Token "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to reach Pivotal Network: %s", err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return resp.Body, nil
	case http.StatusUnauthorized:
		return nil, fmt.Errorf("invalid Pivotal Network API token")
	case 451:
		return nil, fmt.Errorf("you must accept the EULA before you can download the PCF Dev image: %s/products/pcfdev#/releases/1622", c.Host)
	default:
		return nil, fmt.Errorf("Pivotal Network returned: %s", resp.Status)
	}
}

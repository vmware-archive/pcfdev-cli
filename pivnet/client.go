package pivnet

import (
	"fmt"
	"net/http"
	"os"
)

type Client struct {
	Host          string
	ReleaseId     string
	ProductFileId string
}

func (c *Client) DownloadOVA(token string) (ova *DownloadReader, err error) {
	uri := fmt.Sprintf("%s/api/v2/products/pcfdev/releases/%s/product_files/%s/download", c.Host, c.ReleaseId, c.ProductFileId)
	req, err := http.NewRequest("POST", uri, nil)
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
		return &DownloadReader{ReadCloser: resp.Body, Writer: os.Stdout, ContentLength: resp.ContentLength}, nil
	case http.StatusUnauthorized:
		return nil, fmt.Errorf("invalid Pivotal Network API token")
	case 451:
		return nil, fmt.Errorf("you must accept the EULA before you can download the PCF Dev image: %s/products/pcfdev#/releases/%s", c.Host, c.ReleaseId)
	default:
		return nil, fmt.Errorf("Pivotal Network returned: %s", resp.Status)
	}
}

package pivnet

import (
	"errors"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	Host  string
	Token string
}

func (c *Client) DownloadOVA() (io.ReadCloser, error) {
	client := http.DefaultClient
	if c.Token == "" {
		return nil, errors.New("missing Pivotal Network Api token, please set PIVNET_TOKEN environment variable")
	}

	req, err := http.NewRequest("POST", c.Host+"/api/v2/products/pcfdev/releases/1622/product_files/4149/download", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Token "+c.Token)
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.New("failed to reach Pivotal Network")
	}

	if resp.StatusCode == 451 {
		return nil, fmt.Errorf("you must accept the eula before you can download the pcfdev image: %s/products/pcfdev#/releases/1622", c.Host)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Pivotal Network returned: %s", resp.Status)
	}

	return resp.Body, nil
}

package pivnet

import (
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	Host   string
	Config Config
}

//go:generate mockgen -package mocks -destination mocks/config.go github.com/pivotal-cf/pcfdev-cli/pivnet Config
type Config interface {
	GetToken() string
}

func (c *Client) DownloadOVA() (io.ReadCloser, error) {
	req, err := http.NewRequest("POST", c.Host+"/api/v2/products/pcfdev/releases/1622/product_files/4149/download", nil)
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

package headway

import (
	"fmt"
	"github.com/pkg/errors"
	"net/http"
)

type Client struct {
	name   string
	host   string
	client http.Client
}

func NewClient(host string, name string) *Client {
	return &Client{
		name:   name,
		host:   host,
		client: http.Client{},
	}
}

func (c *Client) Send(current, total float64, comment string) error {
	req, err := http.NewRequest(http.MethodPut, c.host, nil)
	if err != nil {
		return err
	}
	req.URL.Query().Set("name", c.name)
	req.URL.Query().Set("current", fmt.Sprintf("%f", current))
	req.URL.Query().Set("total", fmt.Sprintf("%f", total))
	req.URL.Query().Set("comment", comment)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("invalid request: %d", resp.StatusCode))
	}

	return nil
}

func (c *Client) SetTotal(total float64) error {
	return c.Send(-1, total, "")
}

func (c *Client) UpdateWithComment(current float64, comment string) error {
	return
}

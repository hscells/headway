package headway

import (
	"fmt"
	"github.com/pkg/errors"
	"net/http"
)

type Client struct {
	host   string
	secret string
	client http.Client
}

func NewClient(host, secret string) *Client {
	return &Client{
		secret: secret,
		host:   host,
		client: http.Client{},
	}
}

func (c *Client) Send(current, total float64, name, comment string) error {
	req, err := http.NewRequest(http.MethodPut, c.host, nil)
	if err != nil {
		return err
	}
	q := req.URL.Query()
	q.Add("name", name)
	q.Add("current", fmt.Sprintf("%f", current))
	q.Add("total", fmt.Sprintf("%f", total))
	q.Add("comment", comment)
	q.Add("secret", c.secret)
	req.URL.RawQuery = q.Encode()

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("invalid request: %d", resp.StatusCode))
	}

	return nil
}

func (c *Client) Message(message string) error {
	req, err := http.NewRequest(http.MethodPut, c.host, nil)
	if err != nil {
		return err
	}
	q := req.URL.Query()
	q.Add("message", message)
	q.Add("secret", c.secret)
	req.URL.RawQuery = q.Encode()

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("invalid request: %d", resp.StatusCode))
	}

	return nil
}

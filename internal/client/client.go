package client

import (
	"net/http"
)

type Client struct {
	Client http.Client
	URL    string
}

func NewClient(url string, rt http.RoundTripper) *Client {
	// Small wrapper
	c := &http.Client{
		Transport: rt,
	}

	return &Client{
		URL:    url,
		Client: *c,
	}
}

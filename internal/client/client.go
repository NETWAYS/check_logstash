package client

import (
	"net/http"
)

type Client struct {
	Client http.Client
	Url    string
}

func NewClient(url string, rt http.RoundTripper) *Client {
	// Small wrapper
	c := &http.Client{
		Transport: rt,
	}

	return &Client{
		Url:    url,
		Client: *c,
	}
}

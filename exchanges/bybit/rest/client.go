package rest

import (
	"net/http"
	"time"

	"github.com/pulsoats/core/lib/logx"
)

const BybitV5URL = "https://api.bybit.com"

type Client struct {
	client     *http.Client
	apiKey     string
	apiSecret  string
	recvWindow string
	baseURL    string
	log        logx.Logger
}

type Option func(*Client)

func NewClient(key, secret string, timeout time.Duration, opts ...Option) *Client {
	c := &Client{
		client: &http.Client{
			Timeout: timeout,
		},
		apiKey:     key,
		apiSecret:  secret,
		recvWindow: "5000",
		baseURL:    BybitV5URL,
		log:        logx.Nop(),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func WithLogger(l logx.Logger) Option {
	return func(c *Client) {
		if l == nil {
			l = logx.Nop()
		}
		c.log = l
	}
}

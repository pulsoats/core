package rest

import (
	"log/slog"
	"net/http"
	"time"
)

const BybitV5URL = "https://api.bybit.com"

type Client struct {
	client     *http.Client
	apiKey     string
	apiSecret  string
	recvWindow string
	baseURL    string
	log        *slog.Logger
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
		log:        slog.New(slog.DiscardHandler),
	}
	for _, opt := range opts {
		opt(c)
	}
	c.log = c.log.With("component", "bybit.rest")
	return c
}

func WithLogger(l *slog.Logger) Option {
	return func(c *Client) {
		if l == nil {
			l = slog.New(slog.DiscardHandler)
		}
		c.log = l
	}
}

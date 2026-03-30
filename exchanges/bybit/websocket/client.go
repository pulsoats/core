package websocket

import (
	"log/slog"
	"sync"

	"github.com/pulsoats/core/transport/websocket"
	"github.com/pulsoats/core/transport/websocket/router"
)

type Client struct {
	mu     sync.RWMutex
	conns  map[string]*conn
	apiKey string
	secret string
	log    *slog.Logger
}

type conn struct {
	stream *websocket.Stream
	router *router.Router
}

type Option func(*Client)

func NewWebSocketClient(apiKey, secret string, opts ...Option) *Client {
	c := &Client{
		mu:     sync.RWMutex{},
		conns:  make(map[string]*conn),
		apiKey: apiKey,
		secret: secret,
		log:    slog.New(slog.DiscardHandler),
	}
	for _, opt := range opts {
		opt(c)
	}
	c.log = c.log.With("component", "bybit.ws")
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

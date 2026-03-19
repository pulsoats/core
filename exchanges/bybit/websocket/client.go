package websocket

import (
	"sync"

	"github.com/pulsoats/core/lib/logx"
	"github.com/pulsoats/core/transport/websocket"
	"github.com/pulsoats/core/transport/websocket/router"
)

type Client struct {
	mu     sync.RWMutex
	conns  map[string]*conn
	apiKey string
	secret string
	log    logx.Logger
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
		log:    logx.Nop(),
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

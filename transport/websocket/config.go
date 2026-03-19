package websocket

import (
	"context"
	"encoding/json"
	"time"

	"github.com/coder/websocket"
)

type StreamOption func(*streamCfg) error

type streamCfg struct {
	dialOptions *websocket.DialOptions
	cmds        chan Command

	auth        func(ctx context.Context) (any, error)
	dispatch    func(ctx context.Context, raw json.RawMessage) error
	onReconnect func(ctx context.Context) error
	outBuf      int
	backoffMin  time.Duration
	backoffMax  time.Duration
	pingEvery   time.Duration
	logger      Logger
}

func WithDialOptions(opt *websocket.DialOptions) StreamOption {
	return func(c *streamCfg) error {
		c.dialOptions = opt
		return nil
	}
}

func WithAuth(fn func(ctx context.Context) (any, error)) StreamOption {
	return func(c *streamCfg) error {
		c.auth = fn
		return nil
	}
}

func WithDispatch(fn func(ctx context.Context, raw json.RawMessage) error) StreamOption {
	return func(c *streamCfg) error {
		c.dispatch = fn
		return nil
	}
}

func WithOnReconnect(fn func(ctx context.Context) error) StreamOption {
	return func(c *streamCfg) error {
		c.onReconnect = fn
		return nil
	}
}

func WithOutBuf(n int) StreamOption {
	return func(c *streamCfg) error {
		c.outBuf = n
		return nil
	}
}

func WithReconnect(backoffStart, backoffMax time.Duration) StreamOption {
	return func(c *streamCfg) error {
		c.backoffMin = backoffStart
		c.backoffMax = backoffMax
		return nil
	}
}

func WithPingEvery(d time.Duration) StreamOption {
	return func(c *streamCfg) error {
		c.pingEvery = d
		return nil
	}
}

func WithLogger(l Logger) StreamOption {
	return func(c *streamCfg) error {
		if l == nil {
			l = nopLogger
		}
		c.logger = l
		return nil
	}
}

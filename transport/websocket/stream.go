package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/coder/websocket"
	"github.com/pulsoats/core/errorsx"
)

type Stream struct {
	url         string
	dialOptions *websocket.DialOptions
	cmds        chan Command

	auth       func(ctx context.Context) (any, error)
	clientPing func(ctx context.Context, c *websocket.Conn) error

	dispatch    func(ctx context.Context, raw json.RawMessage) error
	onReconnect func(ctx context.Context) error
	outBuf      int

	backoffStart time.Duration
	backoffMax   time.Duration
	pingEvery    time.Duration

	log Logger
}

func NewStream(url string, cmds chan Command, opts ...StreamOption) (*Stream, error) {
	if url == "" {
		return nil, fmt.Errorf("websocket stream: url: %w", errorsx.ErrRequired)
	}

	c := &streamCfg{
		backoffMin: time.Second,
		backoffMax: 30 * time.Second,
		outBuf:     256,
		pingEvery:  0,
		logger:     nopLogger,
	}

	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}

	c.cmds = cmds

	// --- Валидация обязательных полей ---
	if cmds == nil {
		return nil, fmt.Errorf("websocket stream: cmds channel: %w", errorsx.ErrRequired)
	}

	if c.backoffMin <= 0 {
		c.backoffMin = time.Second
	}
	if c.backoffMax <= 0 {
		c.backoffMax = 30 * time.Second
	}
	if c.pingEvery < 0 {
		c.pingEvery = 0
	}
	if c.dispatch == nil && c.outBuf <= 0 {
		c.outBuf = 256
	}

	return &Stream{
		url:          url,
		dialOptions:  c.dialOptions,
		cmds:         c.cmds,
		auth:         c.auth,
		dispatch:     c.dispatch,
		onReconnect:  c.onReconnect,
		outBuf:       c.outBuf,
		backoffStart: c.backoffMin,
		backoffMax:   c.backoffMax,
		pingEvery:    c.pingEvery,
		log:          c.logger,
	}, nil
}

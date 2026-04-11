package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/coder/websocket"
	"github.com/pulsoats/core/errorsx"
)

type Stream struct {
	url         string // websocket url
	dialOptions *websocket.DialOptions
	cmds        chan Command // канал команд

	auth        func(ctx context.Context) (any, error)               // Функция авторизации: должна реализовать биржа
	dispatch    func(ctx context.Context, raw json.RawMessage) error // Функция
	onReconnect func(ctx context.Context) error

	outBuf       int
	backoffStart time.Duration
	backoffMax   time.Duration
	pingEvery    time.Duration
	pingMsg      any

	log *slog.Logger
}

func NewStream(cfg StreamConfig) (*Stream, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("NewStream: url: %w", errorsx.ErrRequired)
	}
	if cfg.Cmds == nil {
		return nil, fmt.Errorf("NewStream: cmds: %w", errorsx.ErrRequired)
	}

	if cfg.BackoffStart <= 0 {
		cfg.BackoffStart = time.Second
	}
	if cfg.BackoffMax <= 0 {
		cfg.BackoffMax = 30 * time.Second
	}
	if cfg.PingEvery < 0 {
		cfg.PingEvery = 0
	}
	if cfg.Dispatch == nil && cfg.OutBuf <= 0 {
		cfg.OutBuf = 256
	}
	if cfg.Logger == nil {
		cfg.Logger = nopLogger
	}

	return &Stream{
		url:          cfg.URL,
		dialOptions:  cfg.DialOptions,
		cmds:         cfg.Cmds,
		auth:         cfg.Auth,
		dispatch:     cfg.Dispatch,
		onReconnect:  cfg.OnReconnect,
		outBuf:       cfg.OutBuf,
		backoffStart: cfg.BackoffStart,
		backoffMax:   cfg.BackoffMax,
		pingEvery:    cfg.PingEvery,
		pingMsg:      cfg.PingMsg,
		log:          cfg.Logger.With("component", "ws.stream"),
	}, nil
}

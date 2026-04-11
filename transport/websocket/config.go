package websocket

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/coder/websocket"
)

// StreamConfig holds all configuration for a Stream.
// Zero values are valid: defaults are applied in NewStream.
type StreamConfig struct {
	URL         string
	Cmds        chan Command
	DialOptions *websocket.DialOptions

	// Auth should be provided by every exchange
	Auth        func(ctx context.Context) (any, error)
	Dispatch    func(ctx context.Context, raw json.RawMessage) error
	OnReconnect func(ctx context.Context) error

	// OutBuf is the size of the output channel when Dispatch is nil. Default: 256.
	OutBuf int

	// BackoffStart and BackoffMax control reconnect delay. Defaults: 1s / 30s.
	BackoffStart time.Duration
	BackoffMax   time.Duration

	// PingEvery enables periodic heartbeats. Zero disables.
	PingEvery time.Duration
	// PingMsg, when non-nil, sends a JSON application-level ping instead of a
	// WebSocket protocol PING control frame (e.g. exchanges that require {"op":"ping"}).
	PingMsg any

	Logger *slog.Logger
}

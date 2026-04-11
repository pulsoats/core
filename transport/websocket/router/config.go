package router

import (
	"log/slog"
	"time"

	"github.com/pulsoats/core/transport/websocket"
)

// Config holds all configuration for a Router.
// Zero values are valid: defaults are applied in NewRouter.
type Config struct {
	Cmds       chan websocket.Command
	MsgBuilder MsgBuilder
	MsgDecoder MsgDecoder

	// PipeBuf is the buffer size of each topic channel. Default: 64.
	PipeBuf int
	// TopicsPerReq is the max topics per subscribe/unsubscribe request. Default: 10.
	TopicsPerReq int
	// ReqPerSec is the max requests per second (rate limit). Default: 10.
	ReqPerSec int
	// PendingTTL is the TTL for pending requests before the cleaner evicts them.
	// Zero disables the cleaner.
	PendingTTL time.Duration

	// MaxTopics is the maximum number of distinct topics per connection.
	// Zero means unlimited.
	MaxTopics int

	Logger *slog.Logger
}

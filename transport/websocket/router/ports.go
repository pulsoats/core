package router

import (
	"encoding/json"

	"github.com/pulsoats/core/transport/websocket"
)

type Deps struct {
	Cmds       chan websocket.Command
	MsgBuilder MsgBuilder
	MsgDecoder MsgDecoder
}

type MsgBuilder interface {
	Build(reqID string, op Op, topics []string) (any, error)
}

type MsgDecoder interface {
	Decode(raw json.RawMessage) (*StreamMsg, error)
}

// Лёгкий логгер, чтобы не тянуть slog/zap в домен.
type Logger interface {
	Debug(msg string, kv ...any)
	Info(msg string, kv ...any)
	Warn(msg string, kv ...any)
	Error(msg string, kv ...any)
}
type nopLogger struct{}

func (nopLogger) Debug(string, ...any) {}
func (nopLogger) Info(string, ...any)  {}
func (nopLogger) Warn(string, ...any)  {}
func (nopLogger) Error(string, ...any) {}

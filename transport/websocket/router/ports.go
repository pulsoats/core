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

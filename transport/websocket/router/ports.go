package router

import (
	"context"
	"encoding/json"
)

type MsgBuilder interface {
	Build(ctx context.Context, reqID string, op Op, topics []string) (any, error)
}

type MsgDecoder interface {
	Decode(ctx context.Context, raw json.RawMessage) (*StreamMsg, error)
}

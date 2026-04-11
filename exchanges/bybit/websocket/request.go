package websocket

import (
	"context"

	"github.com/pulsoats/core/transport/websocket/router"
)

type request struct {
	ReqID string   `json:"req_id"`
	Op    string   `json:"op"`
	Args  []string `json:"args"`
}

type bybitMsgBuilder struct{}

func (bybitMsgBuilder) Build(_ context.Context, reqID string, op router.Op, topics []string) (any, error) {
	req := request{
		ReqID: reqID,
		Op:    string(op),
		Args:  topics,
	}

	return req, nil
}

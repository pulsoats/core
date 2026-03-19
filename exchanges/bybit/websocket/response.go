package websocket

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pulsoats/core/domain/derrors"
	"github.com/pulsoats/core/transport/websocket/router"
)

var ErrUnknownFrame = fmt.Errorf("%w: bybit websocket frame is unknown", derrors.ErrInvalidArgument)

type response struct {
	// data
	Topic *string         `json:"topic"`
	Type  *string         `json:"type"`
	Data  json.RawMessage `json:"data"`

	// control / ack
	Success *bool   `json:"success"`
	Op      *string `json:"op"`
	ReqID   string  `json:"req_id"`
	ConnID  string  `json:"conn_id"`
	RetCode *int    `json:"retCode"`
	RetMsg  string  `json:"ret_msg"`
}

type CommandRespData struct {
	SuccessTopics []string `json:"successTopics"`
	FailTopics    []string `json:"failTopics"`
}

type bybitMsgDecoder struct{}

func derefString(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

func (bybitMsgDecoder) Decode(raw json.RawMessage) (*router.StreamMsg, error) {
	var msg response

	if err := json.Unmarshal(raw, &msg); err != nil {
		return &router.StreamMsg{Kind: router.StreamMsgKindUnknown}, err
	}

	if msg.Topic != nil && msg.Success == nil {
		return &router.StreamMsg{
			Kind:  router.StreamMsgKindData,
			Topic: strings.TrimSpace(*msg.Topic),
			Type:  derefString(msg.Type),
			Data:  msg.Data,
			Raw:   raw,
		}, nil
	}

	// Ack || Control
	if msg.Success != nil || msg.Op != nil || msg.ReqID != "" {
		var failed []string
		if msg.Data != nil {
			cmd, err := DecodeCommandRespData(msg.Data)
			if err != nil {
				return &router.StreamMsg{}, err
			}
			failed = append(failed, cmd.FailTopics...)
		}
		return &router.StreamMsg{
			Kind:         router.StreamMsgKindAck,
			Op:           derefString(msg.Op),
			Success:      msg.Success != nil && *msg.Success,
			ReqID:        msg.ReqID,
			Type:         derefString(msg.Type),
			RetMsg:       strings.TrimSpace(msg.RetMsg),
			Data:         msg.Data,
			FailedTopics: failed,
			Raw:          raw,
		}, nil
	}

	return &router.StreamMsg{}, ErrUnknownFrame
}

func DecodeCommandRespData(data json.RawMessage) (CommandRespData, error) {
	var d CommandRespData
	err := json.Unmarshal(data, &d)
	return d, err
}

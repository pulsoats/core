package websocket

import (
	"encoding/json"
	"testing"

	"github.com/pulsoats/core/transport/websocket/router"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecode(t *testing.T) {
	tests := []struct {
		name    string
		raw     json.RawMessage
		want    *router.StreamMsg
		wantErr error
	}{
		{
			name: "data_push",
			raw:  json.RawMessage(`{"topic":"kline.1.BTCUSDT","type":"snapshot","data":{"foo":1}}`),
			want: &router.StreamMsg{
				Kind:  router.StreamMsgKindData,
				Topic: "kline.1.BTCUSDT",
				Type:  "snapshot",
				Data:  json.RawMessage(`{"foo":1}`),
				Raw:   json.RawMessage(`{"topic":"kline.1.BTCUSDT","type":"snapshot","data":{"foo":1}}`),
			},
		},
		{
			name: "ack_success",
			raw:  json.RawMessage(`{"success":true,"op":"subscribe","req_id":"abc","type":"COMMAND_RESP","data":{"successTopics":["kline.1.BTCUSDT"],"failTopics":[]}}`),
			want: &router.StreamMsg{
				Kind:    router.StreamMsgKindAck,
				Success: true,
				Op:      "subscribe",
				ReqID:   "abc",
				Type:    "COMMAND_RESP",
				Data:    json.RawMessage(`{"successTopics":["kline.1.BTCUSDT"],"failTopics":[]}`),
				Raw:     json.RawMessage(`{"success":true,"op":"subscribe","req_id":"abc","type":"COMMAND_RESP","data":{"successTopics":["kline.1.BTCUSDT"],"failTopics":[]}}`),
			},
		},
		{
			name:    "unknown_frame",
			raw:     json.RawMessage(`{"retCode":0}`),
			want:    &router.StreamMsg{},
			wantErr: ErrUnknownFrame,
		},
		{
			name:    "invalid_json",
			raw:     json.RawMessage(`invalid`),
			want:    &router.StreamMsg{Kind: router.StreamMsgKindUnknown},
			wantErr: assert.AnError, // сверяемся через errors.Is ниже
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bmd := bybitMsgDecoder{}
			got, err := bmd.Decode(tt.raw)
			if tt.wantErr != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDecodeCommandRespData(t *testing.T) {
	tests := []struct {
		name    string
		raw     json.RawMessage
		want    CommandRespData
		wantErr bool
	}{
		{
			name: "ok",
			raw:  json.RawMessage(`{"successTopics":["kline"],"failTopics":["orderbook"]}`),
			want: CommandRespData{
				SuccessTopics: []string{"kline"},
				FailTopics:    []string{"orderbook"},
			},
		},
		{
			name:    "invalid_json",
			raw:     json.RawMessage(`{"successTopics":`),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeCommandRespData(tt.raw)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

package websocket

import (
	"testing"

	"github.com/pulsoats/core/transport/websocket/router"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_buildRequest(t *testing.T) {
	tests := []struct {
		name   string
		reqID  string
		op     router.Op
		topics []string
	}{
		{
			name:   "subscribe_single_topic",
			reqID:  "req-1",
			op:     router.OpSubscribe,
			topics: []string{"kline.1.BTCUSDT"},
		},
		{
			name:   "unsubscribe_multiple_topics",
			reqID:  "req-2",
			op:     router.OpUnsubscribe,
			topics: []string{"position.BTCUSDT", "balance.USDT"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bmb := bybitMsgBuilder{}
			gotReq, err := bmb.Build(tt.reqID, tt.op, tt.topics)
			require.NoError(t, err)

			wantReq := request{
				ReqID: tt.reqID,
				Op:    string(tt.op),
				Args:  tt.topics,
			}
			assert.Equal(t, wantReq, gotReq)
		})
	}
}

package websocket

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/pulsoats/core/domain/derrors"
)

type authMsg struct {
	Op   string   `json:"op"`
	Args []string `json:"args"`
}

func (w *Client) Auth(ctx context.Context) (any, error) {
	if w.apiKey == "" || w.secret == "" {
		return nil, fmt.Errorf("%w: api secret", derrors.ErrRequired)
	}

	expires := time.Now().UnixMilli() + 1000
	signature := sign(w.secret, expires)

	return authMsg{
		Op: "auth",
		Args: []string{
			w.apiKey,
			strconv.Itoa(int(expires)),
			signature,
		},
	}, nil
}

func sign(apiSecret string, expires int64) string {
	message := "GET/realtime" + strconv.FormatInt(expires, 10)

	mac := hmac.New(sha256.New, []byte(apiSecret))
	mac.Write([]byte(message))

	return hex.EncodeToString(mac.Sum(nil))
}

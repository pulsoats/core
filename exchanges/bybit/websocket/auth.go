package websocket

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/pulsoats/core/errorsx"
)

type authMsg struct {
	Op   string   `json:"op"`
	Args []string `json:"args"`
}

// Auth возвращает сообщение авторизации, которое содержит API-ключ, expires и подпись для WebSocket-соединения Bybit.
func (w *Client) Auth() (any, error) {
	if w.apiKey == "" || w.secret == "" {
		return nil, fmt.Errorf("bybit websocket: auth api secret: %w", errorsx.ErrRequired)
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

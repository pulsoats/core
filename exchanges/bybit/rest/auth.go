package rest

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/pulsoats/core/errorsx"
)

const recvWindow = "5000"

func sign(apiKey, apiSecret, recvWindow, payloadSuffix string, timestampMs int64) string {
	payload := strconv.FormatInt(timestampMs, 10) + apiKey + recvWindow + payloadSuffix

	mac := hmac.New(sha256.New, []byte(apiSecret))
	mac.Write([]byte(payload))

	return hex.EncodeToString(mac.Sum(nil))
}

func setAuthHeaders(apiKey, apiSecret, payloadSuffix string, t time.Time, req *http.Request) error {
	if t.IsZero() {
		return fmt.Errorf("bybit rest: auth timestamp is zero: %w", errorsx.ErrInvalidArgument)
	}

	tsMs := t.UnixMilli()

	sign := sign(apiKey, apiSecret, recvWindow, payloadSuffix, tsMs)

	req.Header.Set("X-BAPI-API-KEY", apiKey)
	req.Header.Set("X-BAPI-SIGN", sign)
	req.Header.Set("X-BAPI-TIMESTAMP", strconv.FormatInt(tsMs, 10))
	req.Header.Set("X-BAPI-RECV-WINDOW", recvWindow)
	req.Header.Set("X-BAPI-SIGN-TYPE", "2")

	return nil
}

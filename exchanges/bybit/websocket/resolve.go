package websocket

import (
	"fmt"

	"github.com/pulsoats/core/domain/market"
)

const (
	mainnetURL = "wss://stream.bybit.com/v5/"
)

type scope string

const (
	scopePublic  scope = "public"
	scopePrivate scope = "private"
	scopeTrade   scope = "trade"
)

func resolveURL(scope scope, category market.Category) (string, error) {
	host := mainnetURL
	switch scope {
	case scopePublic:
		if category == "" {
			return "", fmt.Errorf("public requires category")
		}
		return host + "public/" + string(category), nil
	case scopePrivate:
		return host + "private", nil
	case scopeTrade:
		return host + "trade", nil
	default:
		return "", fmt.Errorf("unknown scope %q", scope)
	}
}

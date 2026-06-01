package bybit

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/pulsoats/core/exchange"
	"github.com/pulsoats/core/exchanges/bybit/rest"
	"github.com/pulsoats/core/exchanges/bybit/specs"
	"github.com/pulsoats/core/exchanges/bybit/websocket"
	"github.com/pulsoats/core/market"
)

const Code = "bybit"

var Metadata = exchange.Meta{
	Code:       Code,
	Intervals:  specs.ListIntervals(),
	Categories: specs.ListCategories(),
}

type Bybit struct {
	rest *rest.Client
	ws   *websocket.Client
}

func (b *Bybit) Meta() exchange.Meta {
	return Metadata
}

func (b *Bybit) Code() string {
	return Code
}

// NewClient создаёт клиент Bybit. Если creds == nil, создаётся публичный клиент без авторизации.
func NewClient(logger *slog.Logger, creds *exchange.Credentials) (*Bybit, error) {
	if creds != nil {
		if creds.APIKey == "" || creds.APISecret == "" {
			return nil, errors.New("bybit: api_key and api_secret are required")
		}
		return NewBybitClient(creds.APIKey, creds.APISecret, logger), nil
	}
	return NewBybitClient("", "", logger), nil
}

func NewBybitClient(apiKey, secret string, logger *slog.Logger) *Bybit {
	if logger == nil {
		logger = slog.Default()
	}
	return &Bybit{
		rest: rest.NewClient(apiKey, secret, 5*time.Second, rest.WithLogger(logger)),
		ws:   websocket.NewWebSocketClient(apiKey, secret, websocket.WithLogger(logger)),
	}
}

func (b *Bybit) Candles(ctx context.Context, spec market.Spec, interval market.Interval, from time.Time, to time.Time) ([]market.Candle, error) {
	return b.rest.Candles(ctx, spec, interval, from, to)
}

func (b *Bybit) FeeRate(ctx context.Context, category string, symbol, baseCoin string) (market.TakerMakerFees, error) {
	return b.rest.FeeRate(ctx, category, symbol, baseCoin)
}

func (b *Bybit) DefaultFees(category string) (market.TakerMakerFees, error) {
	return specs.DefaultFees(category)
}

func (b *Bybit) InstrumentExists(ctx context.Context, category string, symbol string) (bool, error) {
	return b.rest.InstrumentExists(ctx, category, symbol)
}

func (b *Bybit) StreamCandles(ctx context.Context, spec market.Spec, interval market.Interval, confirmedOnly bool) (chan market.Candle, <-chan error, error) {
	return b.ws.StreamCandles(ctx, spec, interval, confirmedOnly)
}

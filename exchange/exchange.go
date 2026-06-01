package exchange

import (
	"context"
	"log/slog"
	"time"

	"github.com/pulsoats/core/market"
)

// Credentials содержит учётные данные для авторизации на бирже.
type Credentials struct {
	APIKey     string
	APISecret  string
	Passphrase string // опционально, используется на OKX и др.
}

// Factory создаёт клиент биржи. Если creds == nil, создаётся публичный клиент без авторизации.
type Factory func(logger *slog.Logger, creds *Credentials) (Client, error)

// Meta описывает статические возможности конкретной реализации биржи.
type Meta struct {
	Code       string
	Intervals  []string
	Categories []string
}

// PublicClient описывает публичные методы биржи, не требующие авторизации.
type PublicClient interface {
	Meta() Meta
	Code() string
	Candles(ctx context.Context, spec market.Spec, interval market.Interval, from time.Time, to time.Time) ([]market.Candle, error)
	DefaultFees(category string) (market.TakerMakerFees, error)
	StreamCandles(ctx context.Context, spec market.Spec, interval market.Interval, confirmedOnly bool) (chan market.Candle, <-chan error, error)
	InstrumentExists(ctx context.Context, category string, symbol string) (bool, error)
}

// Client описывает полный набор методов биржи, включая приватные.
type Client interface {
	PublicClient
	FeeRate(ctx context.Context, category string, symbol, baseCoin string) (market.TakerMakerFees, error)
}

package exchange

import (
	"context"
	"log/slog"
	"time"

	"github.com/pulsoats/core/market"
)

// Factory создаёт клиент биржи, читая учётные данные из переменных окружения.
type Factory func(logger *slog.Logger, auth bool) (Client, error)

// Meta описывает статические возможности конкретной реализации биржи.
type Meta struct {
	Code       string
	Intervals  []market.Interval
	Categories []market.Category
}

// PublicClient описывает публичные методы биржи, не требующие авторизации.
type PublicClient interface {
	Meta() Meta
	Code() string
	Candles(ctx context.Context, spec market.Spec, interval market.Interval, from time.Time, to time.Time) ([]market.Candle, error)
	DefaultFees(category market.Category) (market.TakerMakerFees, error)
	StreamCandles(ctx context.Context, spec market.Spec, interval market.Interval, confirmedOnly bool) (chan market.Candle, <-chan error, error)
	InstrumentExists(ctx context.Context, category market.Category, symbol string) (bool, error)
}

// Client описывает полный набор методов биржи, включая приватные.
type Client interface {
	PublicClient
	FeeRate(ctx context.Context, category market.Category, symbol, baseCoin string) (market.TakerMakerFees, error)
}

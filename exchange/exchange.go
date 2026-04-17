package exchange

import (
	"context"
	"time"

	market2 "github.com/pulsoats/core/market"
)

// Meta describes static capabilities of a particular exchange implementation.
type Meta struct {
	Code       string
	Intervals  []market2.Interval
	Categories []market2.Category
}

type API interface {
	Meta() Meta
	Code() string
	Candles(ctx context.Context, spec market2.Spec, interval market2.Interval, from time.Time, to time.Time) ([]market2.Candle, error)
	FeeRate(ctx context.Context, category market2.Category, symbol, baseCoin string) (market2.TakerMakerFees, error)
	DefaultFees(category market2.Category) (market2.TakerMakerFees, error)
	StreamCandles(ctx context.Context, spec market2.Spec, interval market2.Interval, confirmedOnly bool) (chan market2.Candle, <-chan error, error)
	InstrumentExists(ctx context.Context, category market2.Category, symbol string) (bool, error)
}

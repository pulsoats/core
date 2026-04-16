package exchange

import (
	"context"
	"time"

	"github.com/pulsoats/core/domain/market"
)

// Meta describes static capabilities of a particular exchange implementation.
type Meta struct {
	Code       string
	Intervals  []market.Interval
	Categories []market.Category
}

type API interface {
	Meta() Meta
	Code() string
	Candles(ctx context.Context, spec market.Spec, interval market.Interval, from time.Time, to time.Time) ([]market.Candle, error)
	FeeRate(ctx context.Context, category market.Category, symbol, baseCoin string) (market.TakerMakerFees, error)
	DefaultFees(category market.Category) (market.TakerMakerFees, error)
	StreamCandles(ctx context.Context, spec market.Spec, interval market.Interval, confirmedOnly bool) (chan market.Candle, <-chan error, error)
	InstrumentExists(ctx context.Context, category market.Category, symbol string) (bool, error)
}

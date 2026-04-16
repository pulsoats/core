package exchange

import (
	"context"
	"errors"
	"time"

	"github.com/pulsoats/core/domain/market"
)

var (
	ErrExchangeNotFound = errors.New("exchange registry: api not registered")
	ErrFactoryNil       = errors.New("exchange registry: factory is nil")
	ErrExchangeEmpty    = errors.New("exchange registry: code is empty")
	ErrFactoryFailed    = errors.New("exchange registry: factory failed")
)

type Factory func(apiKey, apiSecret string) (API, error)

// Meta describes static capabilities of a particular exchange implementation.
type Meta struct {
	Code       string
	Intervals  []market.Interval
	Categories []market.Category
}

type API interface {
	Code() string
	Candles(ctx context.Context, spec market.Spec, interval market.Interval, from time.Time, to time.Time) ([]market.Candle, error)
	FeeRate(ctx context.Context, category market.Category, symbol, baseCoin string) (market.TakerMakerFees, error)
	DefaultFees(category market.Category) (market.TakerMakerFees, error)
	StreamCandles(ctx context.Context, spec market.Spec, interval market.Interval, confirmedOnly bool) (chan market.Candle, <-chan error, error)
	InstrumentExists(ctx context.Context, category market.Category, symbol string) (bool, error)
}

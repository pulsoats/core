package bybit

import (
	"context"
	"log/slog"
	"time"

	"github.com/pulsoats/core/domain/exchange"
	"github.com/pulsoats/core/domain/market"
	"github.com/pulsoats/core/exchanges/bybit/rest"
	"github.com/pulsoats/core/exchanges/bybit/specs"
	"github.com/pulsoats/core/exchanges/bybit/websocket"
)

const Code = "bybit"

var Metadata = exchange.Meta{
	Code:       Code,
	Intervals:  specs.ListIntervals(),
	Categories: specs.ListCategories(),
}

var supportedIntervals = []market.Interval{
	market.Interval1m,
	market.Interval3m,
	market.Interval5m,
	market.Interval15m,
	market.Interval30m,
	market.Interval1h,
	market.Interval2h,
	market.Interval4h,
	market.Interval6h,
	market.Interval12h,
	market.Interval1d,
	market.Interval1w,
	market.Interval1M,
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

func (b *Bybit) Intervals() []market.Interval {
	return append([]market.Interval(nil), supportedIntervals...)
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

func (b *Bybit) FeeRate(ctx context.Context, category market.Category, symbol, baseCoin string) (market.TakerMakerFees, error) {
	return b.rest.FeeRate(ctx, category, symbol, baseCoin)
}

func (b *Bybit) DefaultFees(category market.Category) (market.TakerMakerFees, error) {
	return specs.DefaultFees(category)
}

func (b *Bybit) InstrumentExists(ctx context.Context, category market.Category, symbol string) (bool, error) {
	return b.rest.InstrumentExists(ctx, category, symbol)
}

func (b *Bybit) StreamCandles(ctx context.Context, spec market.Spec, interval market.Interval, confirmedOnly bool) (chan market.Candle, <-chan error, error) {
	return b.ws.StreamCandles(ctx, spec, interval, confirmedOnly)
}

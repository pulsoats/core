package bybit

import (
	"context"
	"log/slog"
	"time"

	"github.com/pulsoats/core/exchange"
	"github.com/pulsoats/core/exchanges/bybit/rest"
	"github.com/pulsoats/core/exchanges/bybit/specs"
	"github.com/pulsoats/core/exchanges/bybit/websocket"
	market2 "github.com/pulsoats/core/market"
)

const Code = "bybit"

var Metadata = exchange.Meta{
	Code:       Code,
	Intervals:  specs.ListIntervals(),
	Categories: specs.ListCategories(),
}

var supportedIntervals = []market2.Interval{
	market2.Interval1m,
	market2.Interval3m,
	market2.Interval5m,
	market2.Interval15m,
	market2.Interval30m,
	market2.Interval1h,
	market2.Interval2h,
	market2.Interval4h,
	market2.Interval6h,
	market2.Interval12h,
	market2.Interval1d,
	market2.Interval1w,
	market2.Interval1M,
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

func (b *Bybit) Intervals() []market2.Interval {
	return append([]market2.Interval(nil), supportedIntervals...)
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

func (b *Bybit) Candles(ctx context.Context, spec market2.Spec, interval market2.Interval, from time.Time, to time.Time) ([]market2.Candle, error) {
	return b.rest.Candles(ctx, spec, interval, from, to)
}

func (b *Bybit) FeeRate(ctx context.Context, category market2.Category, symbol, baseCoin string) (market2.TakerMakerFees, error) {
	return b.rest.FeeRate(ctx, category, symbol, baseCoin)
}

func (b *Bybit) DefaultFees(category market2.Category) (market2.TakerMakerFees, error) {
	return specs.DefaultFees(category)
}

func (b *Bybit) InstrumentExists(ctx context.Context, category market2.Category, symbol string) (bool, error) {
	return b.rest.InstrumentExists(ctx, category, symbol)
}

func (b *Bybit) StreamCandles(ctx context.Context, spec market2.Spec, interval market2.Interval, confirmedOnly bool) (chan market2.Candle, <-chan error, error) {
	return b.ws.StreamCandles(ctx, spec, interval, confirmedOnly)
}

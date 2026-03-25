package bybit

import (
	"context"
	"time"

	"github.com/pulsoats/core/domain/exchange"
	"github.com/pulsoats/core/domain/market"
	"github.com/pulsoats/core/exchanges/bybit/rest"
	"github.com/pulsoats/core/exchanges/bybit/specs"
	"github.com/pulsoats/core/exchanges/bybit/websocket"
	"github.com/pulsoats/core/lib/logx"
)

const Code = "bybit"

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

var Metadata = exchange.Meta{
	Code:       Code,
	Intervals:  specs.ListIntervals(),
	Categories: specs.ListCategories(),
	PriceTypes: specs.ListPriceTypes(),
}

type Bybit struct {
	rest *rest.Client
	ws   *websocket.Client
	log  logx.Logger
}

func (b *Bybit) Code() string {
	return Code
}

func (b *Bybit) Intervals() []market.Interval {
	return append([]market.Interval(nil), supportedIntervals...)
}

type Option func(*Bybit)

func NewBybitClient(apiKey, secret string, opts ...Option) *Bybit {
	b := &Bybit{log: logx.Nop()}
	for _, opt := range opts {
		opt(b)
	}
	restClient := rest.NewClient(apiKey, secret, 5*time.Second, rest.WithLogger(b.log))
	wsClient := websocket.NewWebSocketClient(apiKey, secret, websocket.WithLogger(b.log))

	b.rest = restClient
	b.ws = wsClient
	return b
}

func WithLogger(l logx.Logger) Option {
	return func(b *Bybit) {
		if l == nil {
			l = logx.Nop()
		}
		b.log = l
	}
}

func (b *Bybit) Candles(ctx context.Context, spec market.CandleSpec, from time.Time, to time.Time, priceType market.PriceType) ([]market.Candle, error) {
	return b.rest.Candles(ctx, spec, from, to, priceType)
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

func (b *Bybit) StreamCandles(ctx context.Context, spec market.CandleSpec, confirmedOnly bool) (chan market.Candle, <-chan error, error) {
	return b.ws.StreamCandles(ctx, spec, confirmedOnly)
}

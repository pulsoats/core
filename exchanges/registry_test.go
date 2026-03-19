package exchanges

import (
	"context"
	"testing"
	"time"

	"github.com/pulsoats/core/domain/exchange"
	"github.com/pulsoats/core/domain/market"
	"github.com/pulsoats/core/exchanges/bybit"
	"github.com/stretchr/testify/require"
)

func TestRegistry_Defaults(t *testing.T) {
	r := NewRegistry()

	api, err := r.Create(bybit.Code, "", "")
	require.NoError(t, err)
	require.Equal(t, "bybit", api.Code())
}

func TestRegistry_CreateUnknown(t *testing.T) {
	r := NewRegistry()

	api, err := r.Create("unknown", "", "")
	require.Nil(t, api)
	require.ErrorIs(t, err, exchange.ErrExchangeNotFound)
}

func TestRegistry_RegisterCustom(t *testing.T) {
	r := NewRegistry()
	const custom = "custom"

	meta := exchange.Meta{
		Code:      custom,
		Intervals: []market.Interval{market.Interval1m},
	}

	err := r.add(meta, func(string, string) (exchange.API, error) {
		return &stubAPI{code: "custom"}, nil
	})
	require.NoError(t, err)

	api, err := r.Create(custom, "", "")
	require.NoError(t, err)
	require.Equal(t, "custom", api.Code())
}

func TestRegistry_CreateAllPublic(t *testing.T) {
	r := NewRegistry()
	const custom = "custom"
	meta := exchange.Meta{
		Code:      custom,
		Intervals: []market.Interval{market.Interval1m},
	}
	require.NoError(t, r.add(meta, func(string, string) (exchange.API, error) {
		return &stubAPI{code: custom}, nil
	}))

	all, err := r.CreateAllPublic()
	require.NoError(t, err)
	require.Contains(t, all, bybit.Code)
	require.Contains(t, all, custom)
	require.Equal(t, custom, all[custom].Code())
}

func TestRegistry_Metadata(t *testing.T) {
	r := NewRegistry()

	meta, ok := r.Metadata(bybit.Code)
	require.True(t, ok)
	require.Equal(t, bybit.Code, meta.Code)
	require.NotEmpty(t, meta.Intervals)

	metaList := r.ListMetadata()
	require.NotEmpty(t, metaList)
}

type stubAPI struct {
	code string
}

func (s *stubAPI) Code() string                 { return s.code }
func (s *stubAPI) Intervals() []market.Interval { return nil }
func (s *stubAPI) Candles(context.Context, market.CandleSpec, time.Time, time.Time, market.PriceType) ([]market.Candle, error) {
	return nil, nil
}

func (s *stubAPI) FeeRate(context.Context, market.Category, string, string) (market.TakerMakerFees, error) {
	return market.TakerMakerFees{}, nil
}

func (s *stubAPI) DefaultFees(category market.Category) (market.TakerMakerFees, error) {
	return market.TakerMakerFees{}, nil
}

func (s *stubAPI) StreamCandles(context.Context, market.CandleSpec, bool) (chan market.Candle, <-chan error, error) {
	return nil, nil, nil
}

func (s *stubAPI) InstrumentExists(context.Context, market.Category, string) (bool, error) {
	return true, nil
}

package websocket

import (
	"testing"

	"github.com/pulsoats/core/domain/market"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_decodeCandle(t *testing.T) {
	tests := []struct {
		name       string
		raw        RawCandle
		want       market.Candle
		wantOK     bool
		wantErr    bool
		errMessage string
	}{
		{
			name: "confirmed_candle",
			raw: RawCandle{
				Start:    1700000000000,
				End:      1700000006000,
				Interval: "1",
				Open:     "123.45",
				High:     "130.00",
				Low:      "120.00",
				Close:    "125.55",
				Volume:   "987.654",
				Turnover: "1234.56",
				Confirm:  true,
				Ts:       1700000007000,
			},
			want: market.Candle{
				Time:     1700000000000,
				Open:     12345,
				High:     13000,
				Low:      12000,
				Close:    12555,
				Volume:   987654000,
				Turnover: 1234.56,
			},
			wantOK:  true,
			wantErr: false,
		},
		{
			name: "unconfirmed_candle",
			raw: RawCandle{
				Start:    1700000010000,
				End:      1700000016000,
				Interval: "3",
				Open:     "10.00",
				High:     "11.00",
				Low:      "9.50",
				Close:    "10.50",
				Volume:   "42.0",
				Turnover: "420.0",
				Confirm:  false,
				Ts:       1700000017000,
			},
			want: market.Candle{
				Time:     1700000010000,
				Open:     1000,
				High:     1100,
				Low:      950,
				Close:    1050,
				Volume:   42000000,
				Turnover: 420.0,
			},
			wantOK:  false,
			wantErr: false,
		},
		{
			name: "invalid_volume",
			raw: RawCandle{
				Start:    1700000020000,
				End:      1700000026000,
				Interval: "1",
				Open:     "1",
				High:     "1",
				Low:      "1",
				Close:    "1",
				Volume:   "asdavdsa",
				Turnover: "1",
				Confirm:  true,
				Ts:       1700000027000,
			},
			want:    market.Candle{},
			wantOK:  false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok, err := DecodeCandle(tt.raw)

			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, market.Candle{}, got)
				assert.False(t, ok)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantOK, ok)
		})
	}
}

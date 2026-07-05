package csv

import (
	"strconv"
	"time"

	"github.com/pulsoats/core/detect"
	"github.com/pulsoats/core/lib/format"
	"github.com/pulsoats/core/lib/units"
	"github.com/pulsoats/core/market"
)

// EncodeSignal преобразует detect.Signal в CSV-строку
func EncodeSignal(sig detect.Signal) []string {
	profitability := float64(sig.ExpectedReturnPPM) / 1_000_000

	return []string{
		sig.ID.String(),
		sig.RunID.String(),
		sig.DetectorCode,
		sig.DetectorVersion,
		sig.DetectorOptsLabel,
		sig.CandleTime.Format(time.RFC3339),

		// деньги
		format.CentsToString(sig.CandleValue),
		format.CentsToString(sig.BuyValue),
		format.CentsToString(sig.TakeProfitValue),
		format.CentsToString(sig.StopLossValue),

		// доля (не проценты и не ppm)
		strconv.FormatFloat(profitability, 'f', -1, 64),
		sig.CreatedAt.Format(time.RFC3339),
	}
}

// EncodeCandle преобразует свечу домена в CSV-строку (время в RFC3339, цены в CentsToString).
func EncodeCandle(candle market.Candle) []string {
	candleTime := time.UnixMilli(candle.Time).UTC().Format(time.RFC3339)
	volume := float64(candle.Volume) / float64(units.PPM)

	return []string{
		candleTime,
		format.CentsToString(candle.Open),
		format.CentsToString(candle.High),
		format.CentsToString(candle.Low),
		format.CentsToString(candle.Close),
		strconv.FormatFloat(volume, 'f', -1, 64),
		strconv.FormatFloat(candle.Turnover, 'f', -1, 64),
	}
}

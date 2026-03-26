package csv

import (
	"strconv"
	"time"

	"github.com/pulsoats/core/domain/detect"
	"github.com/pulsoats/core/domain/market"
	"github.com/pulsoats/core/lib/format"
	"github.com/pulsoats/core/lib/units"
)

// EncodeSignal преобразует сигнал домена в CSV-строку.
// Формат соответствует тому, что ожидает DecodeSignal.
func EncodeSignal(sig detect.Signal) []string {
	timeStr := time.UnixMilli(sig.Time).UTC().Format(time.RFC3339)

	profitability := float64(sig.ExpectedReturnPPM) / 1_000_000

	return []string{
		sig.ID.String(),
		sig.Status,
		sig.Detector,
		timeStr,

		// деньги
		format.CentsToString(sig.Value),
		format.CentsToString(sig.BuyValue),
		format.CentsToString(sig.TakeProfitValue),
		format.CentsToString(sig.StopLossValue),

		// доля (не проценты и не ppm)
		strconv.FormatFloat(profitability, 'f', -1, 64),
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

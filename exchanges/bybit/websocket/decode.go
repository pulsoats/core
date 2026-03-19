package websocket

import (
	"strconv"

	"github.com/pulsoats/core/domain/market"
	"github.com/pulsoats/core/lib/parse"
	"github.com/pulsoats/core/lib/units"
)

func DecodeCandle(rawCandle RawCandle) (market.Candle, bool, error) {
	openVal, err := parse.StrToCents(rawCandle.Open)
	if err != nil {
		return market.Candle{}, false, err
	}
	highVal, err := parse.StrToCents(rawCandle.High)
	if err != nil {
		return market.Candle{}, false, err
	}
	lowVal, err := parse.StrToCents(rawCandle.Low)
	if err != nil {
		return market.Candle{}, false, err
	}
	closeVal, err := parse.StrToCents(rawCandle.Close)
	if err != nil {
		return market.Candle{}, false, err
	}

	volume, err := strconv.ParseFloat(rawCandle.Volume, 64)
	if err != nil {
		return market.Candle{}, false, err
	}

	turnover, err := strconv.ParseFloat(rawCandle.Turnover, 64)
	if err != nil {
		return market.Candle{}, false, err
	}

	c := market.Candle{
		Time:     rawCandle.Start,
		Open:     openVal,
		High:     highVal,
		Low:      lowVal,
		Close:    closeVal,
		Volume:   int64(volume * float64(units.PPM)),
		Turnover: turnover,
	}

	return c, rawCandle.Confirm, nil
}

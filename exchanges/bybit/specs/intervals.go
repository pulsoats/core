package specs

import (
	"slices"

	"github.com/pulsoats/core/market"
)

// SupportedIntervals содержит поддерживаемые интервалы Bybit и их строковые представления
var SupportedIntervals = map[market.Interval]string{
	market.Interval1m:  "1",
	market.Interval3m:  "3",
	market.Interval5m:  "5",
	market.Interval15m: "15",
	market.Interval30m: "30",
	market.Interval1h:  "60",
	market.Interval2h:  "120",
	market.Interval4h:  "240",
	market.Interval6h:  "360",
	market.Interval12h: "720",
	market.Interval1d:  "D",
	market.Interval1w:  "W",
	market.Interval1M:  "M",
}

// ListIntervals возвращает слайс строковых представлений интервалов в ASC порядке.
func ListIntervals() []string {
	keys := make([]market.Interval, 0, len(SupportedIntervals))
	for k := range SupportedIntervals {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	res := make([]string, 0, len(keys))
	for _, k := range keys {
		res = append(res, k.String())
	}
	return res
}

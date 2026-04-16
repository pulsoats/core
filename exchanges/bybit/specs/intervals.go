package specs

import (
	"slices"

	"github.com/pulsoats/core/domain/market"
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

// ListIntervals возвращает слайс интервалов в ASC порядке
func ListIntervals() []market.Interval {
	res := make([]market.Interval, 0, len(SupportedIntervals))
	for k, _ := range SupportedIntervals {
		res = append(res, k)
	}
	slices.Sort(res)
	return res
}

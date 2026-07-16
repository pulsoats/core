package detector

import (
	"github.com/pulsoats/core/detect"
	"github.com/pulsoats/core/detect/filter"
	"github.com/pulsoats/core/market"
)

// WithFilters расширяет Detector дополнительными фильтрами, увеличивая WindowSize на максимальный Period среди них.
type WithFilters struct {
	Detector
	filters       []filter.Filter
	extraLookBack int
}

// Wrap возвращает det без обёртки, если filters пуст.
func Wrap(det Detector, filters []filter.Filter) Detector {
	if len(filters) == 0 {
		return det
	}
	extra := 0
	for _, f := range filters {
		if f.Period > extra {
			extra = f.Period
		}
	}
	return &WithFilters{Detector: det, filters: filters, extraLookBack: extra}
}

func (d *WithFilters) WindowSize() int { return d.Detector.WindowSize() + d.extraLookBack }

// Detect прогоняет каждый фильтр через хвост lookback, предшествующий окну детектора.
func (d *WithFilters) Detect(window []market.Candle, fees market.TakerMakerFees) (detect.Signal, bool, error) {
	var zero detect.Signal

	innerSize := d.Detector.WindowSize()
	innerWindow := window[len(window)-innerSize:]
	lookBack := window[:d.extraLookBack]

	sig, found, err := d.Detector.Detect(innerWindow, fees)
	if err != nil || !found {
		return sig, found, err
	}

	for _, f := range d.filters {
		tail := lookBack[d.extraLookBack-f.Period:]
		ok, err := f.Func(innerWindow, tail)
		if err != nil {
			return zero, false, err
		}
		if !ok {
			return zero, false, nil
		}
	}
	return sig, true, nil
}

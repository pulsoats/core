package filter

import (
	"fmt"

	"github.com/pulsoats/core/errorsx"
)

// Config — конфигурация фильтра. Period — количество свечей lookback до начала окна детектора.
type Config struct {
	Code   string
	Period int
}

// Filter — исполняемый фильтр. Period совпадает с Config.Period.
type Filter struct {
	Func   Func
	Period int
}

// FilterFromConfig создает Filter по конфигу. registry не может быть nil.
func FilterFromConfig(registry *Registry, cfg Config) (Filter, error) {
	if registry == nil {
		return Filter{}, fmt.Errorf("filterFunc filter from config: registry is nil: %w", errorsx.ErrInternal)
	}

	ind, err := registry.New(cfg.Code)
	if err != nil {
		return Filter{}, err
	}

	return Filter{
		Func:   ind,
		Period: cfg.Period,
	}, nil
}

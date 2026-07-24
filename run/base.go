package run

import (
	"time"

	"github.com/google/uuid"
	"github.com/pulsoats/core/detect/detector"
	"github.com/pulsoats/core/detect/filter"
	"github.com/pulsoats/core/market"
)

// Base описывает базовый результат работы одного детектора на одном рынке. Сторонние приложения могут дополнять эту структуру.
type Base struct {
	ID              uuid.UUID
	Status          Status
	Market          market.Spec
	Interval        market.Interval
	DetectorConfig  detector.Config
	FiltersConfigs  []filter.Config
	SignalsCount    int64
	FirstCandleTime time.Time
	LastCandleTime  time.Time
	CreatedAt       time.Time
	CreatedBy       string
}

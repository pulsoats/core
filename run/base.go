package run

import (
	"time"

	"github.com/google/uuid"
	"github.com/pulsoats/core/detect"
	"github.com/pulsoats/core/market"
)

type Base struct {
	ID              uuid.UUID
	Status          Status
	Market          market.Spec
	Interval        market.Interval
	Detector        detect.DetectorConfig
	FirstCandleTime time.Time
	LastCandleTime  time.Time
	CreatedAt       time.Time
	CreatedBy       string
}

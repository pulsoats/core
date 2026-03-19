package detect

import (
	"github.com/google/uuid"
	"github.com/pulsoats/core/domain/market"
)

type Signal struct {
	ID                uuid.UUID
	RunID             uuid.UUID
	Status            string
	Detector          string
	OptsLabel         string
	Time              int64
	Value             int64
	BuyValue          int64
	TakeProfitValue   int64
	StopLossValue     int64
	ExpectedReturnPPM int64
	Extremes          []market.Candle
	Fingerprint       uuid.UUID
}

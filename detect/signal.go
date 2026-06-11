package detect

import (
	"github.com/google/uuid"
	"github.com/pulsoats/core/market"
)

type Signal struct {
	ID                uuid.UUID
	RunID             uuid.UUID
	Market            market.Spec
	DetectorCode      string
	DetectorVersion   string
	DetectorOptsLabel string
	Time              int64
	Value             int64
	BuyValue          int64
	TakeProfitValue   int64
	StopLossValue     int64
	ExpectedReturnPPM int64
	Extremes          []market.Candle
	Fingerprint       uuid.UUID
	CreatedAt         int64
}

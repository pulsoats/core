package detect

import (
	"github.com/google/uuid"
	market2 "github.com/pulsoats/core/market"
)

type Signal struct {
	ID                uuid.UUID
	RunID             uuid.UUID
	Market            market2.Spec
	DetectorCode      string
	DetectorOptsLabel string
	Time              int64
	Value             int64
	BuyValue          int64
	TakeProfitValue   int64
	StopLossValue     int64
	ExpectedReturnPPM int64
	Extremes          []market2.Candle
	Fingerprint       uuid.UUID
	CreatedAt         int64
}

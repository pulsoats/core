package detect

import (
	"github.com/google/uuid"
	"github.com/pulsoats/core/market"
)

type Signal struct {
	ID                uuid.UUID
	RunID             uuid.UUID
	DetectorCode      string
	DetectorVersion   string
	DetectorOptsLabel string
	CandleTime        int64
	CandleValue       int64
	BuyValue          int64
	TakeProfitValue   int64
	StopLossValue     int64
	ExpectedReturnPPM int64
	Extremes          []market.Candle `csv:"-"`
	Fingerprint       uuid.UUID       `csv:"-"`
	CreatedAt         int64
}

package detect

import (
	"time"

	"github.com/google/uuid"
)

type Signal struct {
	ID                uuid.UUID
	RunID             uuid.UUID
	DetectorCode      string
	DetectorVersion   string
	DetectorOptsLabel string
	CandleTime        time.Time
	CandleValue       int64
	BuyValue          int64
	TakeProfitValue   int64
	StopLossValue     int64
	ExpectedReturnPPM int64
	Metadata          map[string]string `csv:"-"`
	CreatedAt         time.Time
}

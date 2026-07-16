package detect

import (
	"time"

	"github.com/google/uuid"
)

// Signal — результат работы детектора, содержащий данные для выставления ордера.
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
	Fingerprint       string            `csv:"-"`
	Metadata          map[string]string `csv:"-"`
	CreatedAt         time.Time
}

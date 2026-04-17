package detect

import (
	"context"
	"encoding/json"

	market2 "github.com/pulsoats/core/market"
)

type DetectorKind string

const (
	DetectorKindCandle DetectorKind = "candle"
)

type Detector interface {
	Code() string
	OptsLabel() string
	Kind() DetectorKind
}

type CandleDetector interface {
	Detector
	WindowSize() int
	BarsForBuy() int
	BarsForSell() int
	Detect(ctx context.Context, window []market2.Candle, fees market2.TakerMakerFees) (Signal, bool, error)
}

type DetectorConfig struct {
	Code      string
	OptsLabel string
	Opts      json.RawMessage
}

func (d DetectorConfig) String() string {
	raw, err := json.Marshal(d)
	if err != nil {
		return "<invalid DetectorConfig>"
	}
	return string(raw)
}

// DetectorMeta provides meta information about detector
type DetectorMeta struct {
	Code        string
	Description string
	Kind        DetectorKind
	OptsSchema  json.RawMessage
	Version     string
}

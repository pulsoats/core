package detect

import (
	"context"
	"encoding/json"

	"github.com/pulsoats/core/domain/market"
)

type DetectorKind string

const (
	DetectorKindCandle DetectorKind = "candle"
)

type Detector interface {
	Code() string
	Label() string
	Kind() DetectorKind
}

type CandleDetector interface {
	Detector
	WindowSize() int
	BarsForBuy() int
	BarsForSell() int
	Detect(ctx context.Context, window []market.Candle, fees market.TakerMakerFees) (Signal, bool, error)
}

type DetectorConfig struct {
	Code  string
	Label string
	Opts  json.RawMessage
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
	Code       string
	Desc       string
	Kind       DetectorKind
	OptsSchema json.RawMessage
	Version    string
}

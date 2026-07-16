package detector

import (
	"encoding/json"

	"github.com/pulsoats/core/detect"
	"github.com/pulsoats/core/market"
)

// Detector — торговый алгоритм, генерирующий сигналы по рыночным данным.
type Detector interface {
	Code() string
	OptsLabel() string
	WindowSize() int
	BarsForBuy() int
	BarsForSell() int
	Detect(window []market.Candle, fees market.TakerMakerFees) (detect.Signal, bool, error)
}

// Config описывает конфигурацию детектора. Opts сериализуются в JSON и десериализуются в Registry.
type Config struct {
	Code      string
	Version   string
	OptsLabel string
	Opts      json.RawMessage
}

func (c Config) String() string {
	raw, err := json.Marshal(c)
	if err != nil {
		return "<invalid config>"
	}
	return string(raw)
}

// Meta — статические данные детектора.
type Meta struct {
	Code        string
	Version     string
	Description string
	OptsSchema  json.RawMessage
}

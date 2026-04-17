package w

import (
	"fmt"

	"github.com/pulsoats/core/errorsx"
)

// Opts - опции для W-детектора
type Opts struct {
	LocalMinsDeviation    int64
	MinMaxDeviation       int64
	VolumeSpikeMultiplier int64
	VolumeCVThreshold     int64
	TakeProfitRatio       int64
	StopLossRatio         int64
	BarsForBuy            int
	BarsForSell           int
	WindowSize            int
}

// Validate проверяет корректность конфигурации детектора.
func (o Opts) Validate() error {
	if o.WindowSize < 5 {
		return fmt.Errorf("detector W: window_size must be >= 7: %w", errorsx.ErrInvalidArgument)
	}
	if o.BarsForBuy <= 0 {
		return fmt.Errorf("detector W: bars_for_buy must be > 0: %w", errorsx.ErrInvalidArgument)
	}
	if o.BarsForSell <= 0 {
		return fmt.Errorf("detector W: bars_for_sell must be > 0: %w", errorsx.ErrInvalidArgument)
	}
	return nil
}

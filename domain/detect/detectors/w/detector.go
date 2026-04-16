package w

import (
	"fmt"

	"github.com/pulsoats/core/domain/detect"
)

const code = "W"
const description = `Детектор паттерна "Двойное дно" (W)`
const version = "0.1.0-beta.2"

type Detector struct {
	optsLabel string
	opts      Opts
}

// NewWDetector создает W-детектор.
func NewWDetector(optsLabel string, opts Opts) (detect.CandleDetector, error) {
	if err := opts.Validate(); err != nil {
		return &Detector{}, err
	}

	if optsLabel == "" {
		optsLabel = fmt.Sprintf("%v|%v|%v|%v|%v|%v|%v|%v|%v",
			opts.LocalMinsDeviation,
			opts.MinMaxDeviation,
			opts.VolumeSpikeMultiplier,
			opts.VolumeCVThreshold,
			opts.TakeProfitRatio,
			opts.StopLossRatio,
			opts.BarsForBuy,
			opts.BarsForSell,
			opts.WindowSize,
		)
	}
	return &Detector{optsLabel: optsLabel, opts: opts}, nil
}

func (d *Detector) Code() string { return "W" }

func (d *Detector) OptsLabel() string {
	return d.optsLabel
}

func (d *Detector) Kind() detect.DetectorKind { return detect.DetectorKindCandle }
func (d *Detector) WindowSize() int           { return d.opts.WindowSize }
func (d *Detector) BarsForBuy() int           { return d.opts.BarsForBuy }
func (d *Detector) BarsForSell() int          { return d.opts.BarsForSell }

var Meta = detect.DetectorMeta{
	Code:       code,
	Desc:       description,
	Kind:       detect.DetectorKindCandle,
	OptsSchema: optsSchema,
	Version:    version,
}

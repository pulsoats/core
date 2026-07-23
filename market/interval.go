package market

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pulsoats/core/errorsx"
)

type Interval time.Duration

const (
	Interval1m  = Interval(time.Minute)
	Interval3m  = Interval(3 * time.Minute)
	Interval5m  = Interval(5 * time.Minute)
	Interval15m = Interval(15 * time.Minute)
	Interval30m = Interval(30 * time.Minute)
	Interval1h  = Interval(time.Hour)
	Interval2h  = Interval(2 * time.Hour)
	Interval4h  = Interval(4 * time.Hour)
	Interval6h  = Interval(6 * time.Hour)
	Interval12h = Interval(12 * time.Hour)
	Interval1d  = Interval(24 * time.Hour)
	Interval3d  = Interval(72 * time.Hour)
	Interval1w  = Interval(7 * 24 * time.Hour)
	Interval1M  = Interval(30 * 24 * time.Hour)
)

func (i Interval) String() string {
	switch i {
	case Interval1m:
		return "1m"
	case Interval3m:
		return "3m"
	case Interval5m:
		return "5m"
	case Interval15m:
		return "15m"
	case Interval30m:
		return "30m"
	case Interval1h:
		return "1h"
	case Interval2h:
		return "2h"
	case Interval4h:
		return "4h"
	case Interval6h:
		return "6h"
	case Interval12h:
		return "12h"
	case Interval1d:
		return "1d"
	case Interval3d:
		return "3d"
	case Interval1w:
		return "1w"
	case Interval1M:
		return "1M"
	default:
		return "unknown"
	}
}

func (i Interval) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.String())
}

func ParseInterval(s string) (Interval, error) {
	switch s {
	case "1m":
		return Interval1m, nil
	case "3m":
		return Interval3m, nil
	case "5m":
		return Interval5m, nil
	case "15m":
		return Interval15m, nil
	case "30m":
		return Interval30m, nil
	case "1h":
		return Interval1h, nil
	case "2h":
		return Interval2h, nil
	case "4h":
		return Interval4h, nil
	case "6h":
		return Interval6h, nil
	case "12h":
		return Interval12h, nil
	case "1d":
		return Interval1d, nil
	case "3d":
		return Interval3d, nil
	case "1w":
		return Interval1w, nil
	case "1M":
		return Interval1M, nil
	default:
		return 0, fmt.Errorf("parse interval: %w", errorsx.ErrNotFound)
	}
}

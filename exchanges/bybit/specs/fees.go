package specs

import (
	"fmt"

	"github.com/pulsoats/core/errorsx"
	market2 "github.com/pulsoats/core/market"
)

// DefaultFees возвращает дефолтные комиссии из документации Bybit без запроса к бирже.
func DefaultFees(category market2.Category) (market2.TakerMakerFees, error) {
	switch category {
	case CategorySpot:
		return market2.TakerMakerFees{TakerFeeRate: 1000, MakerFeeRate: 1000}, nil
	case CategoryLinear:
		return market2.TakerMakerFees{TakerFeeRate: 200, MakerFeeRate: 550}, nil
	default:
		return market2.TakerMakerFees{}, fmt.Errorf("bybit specs: fees category=%v: %w", category, errorsx.ErrNotFound)
	}
}

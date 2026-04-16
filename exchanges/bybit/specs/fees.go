package specs

import (
	"fmt"

	"github.com/pulsoats/core/domain/market"
	"github.com/pulsoats/core/errorsx"
)

// DefaultFees возвращает дефолтные комиссии из документации Bybit без запроса к бирже.
func DefaultFees(category market.Category) (market.TakerMakerFees, error) {
	switch category {
	case CategorySpot:
		return market.TakerMakerFees{TakerFeeRate: 1000, MakerFeeRate: 1000}, nil
	case CategoryLinear:
		return market.TakerMakerFees{TakerFeeRate: 200, MakerFeeRate: 550}, nil
	default:
		return market.TakerMakerFees{}, fmt.Errorf("bybit specs: fees category=%v: %w", category, errorsx.ErrNotFound)
	}
}

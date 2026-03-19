package specs

import (
	"fmt"

	"github.com/pulsoats/core/domain/derrors"
	"github.com/pulsoats/core/domain/market"
)

func DefaultFees(category market.Category) (market.TakerMakerFees, error) {
	switch category {
	case CategorySpot:
		return market.TakerMakerFees{TakerFeeRate: 1000, MakerFeeRate: 1000}, nil
	case CategoryLinear:
		return market.TakerMakerFees{TakerFeeRate: 200, MakerFeeRate: 550}, nil
	default:
		return market.TakerMakerFees{}, fmt.Errorf("%w: category %v", derrors.ErrNotFound, category)
	}
}

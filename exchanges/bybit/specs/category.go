package specs

import (
	"github.com/pulsoats/core/domain/market"
)

const (
	CategorySpot    market.Category = "spot"
	CategoryLinear  market.Category = "linear"
	CategoryInverse market.Category = "inverse"
	CategoryOption  market.Category = "option"
)

func IsSupportedCategory(v market.Category) bool {
	switch v {
	case CategorySpot, CategoryLinear, CategoryInverse, CategoryOption:
		return true
	default:
		return false
	}
}

func ListCategories() []market.Category {
	return []market.Category{CategorySpot, CategoryLinear, CategoryInverse, CategoryOption}
}

package specs

import (
	"github.com/pulsoats/core/domain/market"
)

const (
	PriceTypeLast  market.PriceType = "last"
	PriceTypeIndex                  = "index"
	PriceTypeMark                   = "mark"
)

func ListPriceTypes() []market.PriceType {
	return []market.PriceType{PriceTypeLast, PriceTypeIndex, PriceTypeMark}
}

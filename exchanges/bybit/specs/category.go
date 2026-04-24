package specs

const (
	CategorySpot    = "spot"
	CategoryLinear  = "linear"
	CategoryInverse = "inverse"
	CategoryOption  = "option"
)

func IsSupportedCategory(v string) bool {
	switch v {
	case CategorySpot, CategoryLinear, CategoryInverse, CategoryOption:
		return true
	default:
		return false
	}
}

func ListCategories() []string {
	return []string{CategorySpot, CategoryLinear, CategoryInverse, CategoryOption}
}

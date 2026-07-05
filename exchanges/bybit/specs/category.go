package specs

const (
	CategorySpot   = "Spot"
	CategoryLinear = "Linear"
)

func IsSupportedCategory(v string) bool {
	switch v {
	case CategorySpot, CategoryLinear:
		return true
	default:
		return false
	}
}

func ListCategories() []string {
	return []string{CategorySpot, CategoryLinear}
}

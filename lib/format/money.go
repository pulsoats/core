package format

import (
	"strconv"

	"github.com/pulsoats/core/lib/units"
)

// CentsToString - форматирование цены в строку
func CentsToString(cents int64) string {
	return strconv.FormatFloat(float64(cents)/float64(units.Cents), 'f', 2, 64)
}

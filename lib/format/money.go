package format

import (
	"strconv"

	"github.com/pulsoats/core/lib/units"
)

// FormatCents - форматирование цены в строку
func FormatCents(cents int64) string {
	return strconv.FormatFloat(float64(cents)/float64(units.Cents), 'f', 2, 64)
}

package format

import (
	"fmt"

	"github.com/pulsoats/core/lib/units"
)

// FormatPPMPercent - форматирование комиссий/долей в строку относительно ppm (1_000_000)
func FormatPPMPercent(ppm int64) string {
	// например, вывести как 0.00036 или 0.036%
	return fmt.Sprintf("%.6f", float64(ppm)/float64(units.PPM))
}

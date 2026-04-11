package parse

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/pulsoats/core/errorsx"
)

// StrToCents - parsing price string with precision 2 to int64 * 100
func StrToCents(s string) (int64, error) {
	s = strings.TrimSpace(s)

	parts := strings.SplitN(s, ".", 2)

	intPart, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("to cents: %w", errors.Join(errorsx.ErrInvalidArgument, err))
	}

	var frac int64
	if len(parts) == 2 {
		f := parts[1]
		if len(f) >= 2 {
			frac, _ = strconv.ParseInt(f[:2], 10, 64)
		} else if len(f) == 1 {
			frac, _ = strconv.ParseInt(f+"0", 10, 64)
		}
	}

	return intPart*100 + frac, nil
}

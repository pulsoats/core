package websocket

import (
	"context"
	"math/rand"
	"time"
)

func sleepBackoff(ctx context.Context, cur *time.Duration, max time.Duration) bool {
	var jitter time.Duration
	if n := int64(*cur / 5); n > 0 {
		jitter = time.Duration(rand.Int63n(n)) // 0..20%
	}
	delay := *cur + jitter

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
	}

	next := time.Duration(float64(*cur) * 1.7)
	if next > max {
		next = max
	}
	*cur = next

	return true
}

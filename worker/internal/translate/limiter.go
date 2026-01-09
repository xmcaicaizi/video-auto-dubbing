package translate

import (
	"context"
	"time"
)

type rateLimiter struct {
	ch <-chan time.Time
}

func newRateLimiter(rps float64) *rateLimiter {
	if rps <= 0 {
		return nil
	}
	interval := time.Duration(float64(time.Second) / rps)
	if interval <= 0 {
		return nil
	}

	ticker := time.NewTicker(interval)
	ch := make(chan time.Time, 1)
	ch <- time.Now()

	go func() {
		for t := range ticker.C {
			select {
			case ch <- t:
			default:
			}
		}
	}()

	return &rateLimiter{ch: ch}
}

func (r *rateLimiter) Wait(ctx context.Context) error {
	if r == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-r.ch:
		return nil
	}
}

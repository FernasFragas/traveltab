package api

import (
	"context"
	"sync"
	"time"
)

// MinHostInterval is the shortest gap between two requests to the same writer host.
const MinHostInterval = 1500 * time.Millisecond

// HostLimiter spaces requests to each host by at least MinHostInterval. One value is shared by
// every job that fetches from writer sites, and it is safe for concurrent use.
type HostLimiter struct {
	interval time.Duration
	now      func() time.Time
	sleep    func(context.Context, time.Duration) error

	mu   sync.Mutex
	next map[string]time.Time // earliest time the next request to a host may start
}

func NewHostLimiter() *HostLimiter {
	return &HostLimiter{
		interval: MinHostInterval,
		now:      time.Now,
		sleep:    sleepContext,
		next:     make(map[string]time.Time),
	}
}

// Wait blocks until a request to host may start, or until ctx is done.
func (l *HostLimiter) Wait(ctx context.Context, host string) error {
	l.mu.Lock()
	now := l.now()
	slot := l.next[host]
	if slot.Before(now) {
		slot = now
	}
	l.next[host] = slot.Add(l.interval)
	l.mu.Unlock()

	return l.sleep(ctx, slot.Sub(now))
}

func sleepContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

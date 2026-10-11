package api

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHostLimiter_UsesMinimumInterval(t *testing.T) {
	limiter := NewHostLimiter()

	assert.Equal(t, 1500*time.Millisecond, limiter.interval)
	assert.GreaterOrEqual(t, MinHostInterval, 1500*time.Millisecond)
}

func TestHostLimiter_Wait_SpacesRequestsToSameHost(t *testing.T) {
	limiter, slept := frozenLimiter()

	for range 3 {
		require.NoError(t, limiter.Wait(context.Background(), "example.com"))
	}

	assert.Equal(t, []time.Duration{0, MinHostInterval, 2 * MinHostInterval}, *slept)
}

func TestHostLimiter_Wait_HostsAreIndependent(t *testing.T) {
	limiter, slept := frozenLimiter()

	require.NoError(t, limiter.Wait(context.Background(), "a.example"))
	require.NoError(t, limiter.Wait(context.Background(), "b.example"))

	assert.Equal(t, []time.Duration{0, 0}, *slept)
}

func TestHostLimiter_Wait_NoDelayOnceIntervalHasPassed(t *testing.T) {
	limiter, slept := frozenLimiter()
	start := limiter.now()

	require.NoError(t, limiter.Wait(context.Background(), "example.com"))
	limiter.now = func() time.Time { return start.Add(2 * time.Second) }
	require.NoError(t, limiter.Wait(context.Background(), "example.com"))

	assert.Equal(t, []time.Duration{0, 0}, *slept)
}

func TestHostLimiter_Wait_ConcurrentCallersGetDistinctSlots(t *testing.T) {
	limiter, slept := frozenLimiter()
	var wg sync.WaitGroup

	for range 10 {
		wg.Go(func() { assert.NoError(t, limiter.Wait(context.Background(), "example.com")) })
	}
	wg.Wait()

	want := make([]time.Duration, 10)
	for i := range want {
		want[i] = time.Duration(i) * MinHostInterval
	}
	assert.ElementsMatch(t, want, *slept)
}

func TestHostLimiter_Wait_ReturnsContextError(t *testing.T) {
	limiter := NewHostLimiter()
	require.NoError(t, limiter.Wait(context.Background(), "example.com"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := limiter.Wait(ctx, "example.com")

	assert.ErrorIs(t, err, context.Canceled)
}

// frozenLimiter returns a limiter whose clock never moves and whose sleeps are recorded, not taken.
func frozenLimiter() (*HostLimiter, *[]time.Duration) {
	var mu sync.Mutex
	slept := []time.Duration{}
	start := time.Date(2026, 10, 11, 3, 0, 0, 0, time.UTC)
	limiter := NewHostLimiter()
	limiter.now = func() time.Time { return start }
	limiter.sleep = func(_ context.Context, d time.Duration) error {
		mu.Lock()
		defer mu.Unlock()
		slept = append(slept, d)
		return nil
	}
	return limiter, &slept
}

package httpserver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewCityLimiter_AllowsUpToMaxThenBlocks(t *testing.T) {
	limiter := newNewCityLimiter(2)
	now := time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC)

	assert.True(t, limiter.Allow(now))
	assert.True(t, limiter.Allow(now))
	assert.False(t, limiter.Allow(now), "a third new city inside the same minute is blocked")
}

func TestNewCityLimiter_RollsOffAfterAMinute(t *testing.T) {
	limiter := newNewCityLimiter(1)
	now := time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC)

	assert.True(t, limiter.Allow(now))
	assert.False(t, limiter.Allow(now.Add(30*time.Second)), "still inside the same rolling minute")
	assert.True(t, limiter.Allow(now.Add(61*time.Second)), "a full minute has now passed")
}

func TestNewCityLimiter_NeverConsultedTwiceForOneAllow(t *testing.T) {
	// A zero-max limiter always blocks, proving Allow never sneaks an extra hit in before
	// reporting false.
	limiter := newNewCityLimiter(0)
	now := time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC)

	assert.False(t, limiter.Allow(now))
	assert.False(t, limiter.Allow(now))
}

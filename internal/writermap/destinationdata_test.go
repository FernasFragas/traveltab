package writermap

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSourceStatus_NeverCheckedLeavesOutCheckedAt(t *testing.T) {
	status := SourceStatus{Host: "example.com", Status: "active"}

	data, err := json.Marshal(status)

	require.NoError(t, err)
	assert.JSONEq(t, `{"host":"example.com","status":"active"}`, string(data))
}

func TestSourceStatus_CheckedEncodesCheckedAt(t *testing.T) {
	checkedAt := time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)
	status := SourceStatus{Host: "example.com", Status: "active", CheckedAt: &checkedAt}

	data, err := json.Marshal(status)

	require.NoError(t, err)
	assert.JSONEq(t, `{"host":"example.com","status":"active","checked_at":"2026-10-05T00:00:00Z"}`, string(data))
}

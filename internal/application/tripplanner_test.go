package application

import (
	"context"
	"testing"

	"weatherservice/internal/planner"

	"github.com/stretchr/testify/assert"
)

func TestTripPlanner_RejectsInvalidDays(t *testing.T) {
	trip := NewTripPlanner(nil, nil, nil)

	got, err := trip.Plan(context.Background(), planner.Request{City: "Lisbon", Days: 0})

	assert.Nil(t, got)
	assert.ErrorContains(t, err, "a trip is 1 to 5 days, not 0")
}

package caldav_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"calabunga/internal/caldav"
	"calabunga/internal/types"
)

func TestExpandRecurrence_Weekly(t *testing.T) {
	start := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 1, 10, 30, 0, 0, time.UTC)
	rec := &types.Recurrence{
		Frequency: "weekly",
		Interval:  1,
		ByDay:     []string{"MO", "WE", "FR"},
		End:       types.RecurrenceEnd{Type: "count", Value: "3"},
	}

	occurrences, err := caldav.ExpandRecurrence(start, end, rec)
	require.NoError(t, err)
	assert.Len(t, occurrences, 3)
	assert.Equal(t, time.Monday, occurrences[0].Start.Weekday())
}

func TestExpandRecurrence_Never(t *testing.T) {
	start := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 1, 10, 30, 0, 0, time.UTC)
	rec := &types.Recurrence{
		Frequency: "daily",
		Interval:  1,
		End:       types.RecurrenceEnd{Type: "never"},
	}

	rangeEnd := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)
	occurrences, err := caldav.ExpandRecurrenceInRange(start, end, rec, start, rangeEnd)
	require.NoError(t, err)
	assert.Len(t, occurrences, 9)
}

func TestRecurrenceToRRule(t *testing.T) {
	rec := types.Recurrence{
		Frequency: "weekly",
		Interval:  2,
		ByDay:     []string{"MO", "FR"},
		End:       types.RecurrenceEnd{Type: "date", Value: "2026-12-31"},
	}

	rrule, err := caldav.RecurrenceToRRule(rec)
	require.NoError(t, err)
	assert.Contains(t, rrule, "FREQ=WEEKLY")
	assert.Contains(t, rrule, "INTERVAL=2")
	assert.Contains(t, rrule, "BYDAY=MO,FR")
	assert.Contains(t, rrule, "UNTIL=20261231")
}

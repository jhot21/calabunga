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

func TestRecurrenceToRRuleFor_UntilIsInclusiveOfTheLastDay(t *testing.T) {
	rec := types.Recurrence{Frequency: "daily", Interval: 1, End: types.RecurrenceEnd{Type: "date", Value: "2026-06-30"}}

	timed, err := caldav.RecurrenceToRRuleFor(rec, false)
	require.NoError(t, err)
	// Midnight would drop an occurrence at 10:00 on the final day.
	assert.Contains(t, timed, "UNTIL=20260630T235959Z")

	allDay, err := caldav.RecurrenceToRRuleFor(rec, true)
	require.NoError(t, err)
	// RFC 5545: UNTIL must be a DATE when DTSTART is a DATE.
	assert.Contains(t, allDay, "UNTIL=20260630")
	assert.NotContains(t, allDay, "T235959Z")
}

func TestParseRRule_Representable(t *testing.T) {
	cases := []struct {
		name string
		rule string
		want types.Recurrence
	}{
		{"daily", "FREQ=DAILY", types.Recurrence{Frequency: "daily", Interval: 1, End: types.RecurrenceEnd{Type: "never"}}},
		{"weekly days", "FREQ=WEEKLY;BYDAY=TU,TH", types.Recurrence{Frequency: "weekly", Interval: 1, ByDay: []string{"TU", "TH"}, End: types.RecurrenceEnd{Type: "never"}}},
		{"interval and count", "FREQ=MONTHLY;INTERVAL=3;COUNT=5", types.Recurrence{Frequency: "monthly", Interval: 3, End: types.RecurrenceEnd{Type: "count", Value: "5"}}},
		{"month day", "FREQ=MONTHLY;BYMONTHDAY=1,15", types.Recurrence{Frequency: "monthly", Interval: 1, ByMonthDay: []int{1, 15}, End: types.RecurrenceEnd{Type: "never"}}},
		{"until datetime", "FREQ=YEARLY;UNTIL=20301231T235959Z", types.Recurrence{Frequency: "yearly", Interval: 1, End: types.RecurrenceEnd{Type: "date", Value: "2030-12-31"}}},
		{"until date", "FREQ=DAILY;UNTIL=20260630", types.Recurrence{Frequency: "daily", Interval: 1, End: types.RecurrenceEnd{Type: "date", Value: "2026-06-30"}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := caldav.ParseRRule(c.rule)
			require.True(t, ok)
			assert.Equal(t, c.want, got)
		})
	}
}

func TestParseRRule_NotRepresentableInTheEditor(t *testing.T) {
	for _, rule := range []string{
		"FREQ=MONTHLY;BYDAY=1MO",           // ordinal weekday
		"FREQ=MONTHLY;BYDAY=MO;BYSETPOS=-1", // set position
		"FREQ=HOURLY",
		"FREQ=DAILY;COUNT=3;UNTIL=20260630",
		"FREQ=WEEKLY;WKST=SU",
		"FREQ=YEARLY;BYMONTH=2",
		"",
		"garbage",
	} {
		t.Run(rule, func(t *testing.T) {
			_, ok := caldav.ParseRRule(rule)
			assert.False(t, ok)
		})
	}
}

func TestParseRRule_RoundTripsWhatTheEditorWrites(t *testing.T) {
	rec := types.Recurrence{Frequency: "weekly", Interval: 2, ByDay: []string{"MO", "FR"}, End: types.RecurrenceEnd{Type: "count", Value: "10"}}
	rule, err := caldav.RecurrenceToRRule(rec)
	require.NoError(t, err)

	got, ok := caldav.ParseRRule(rule)
	require.True(t, ok)
	assert.Equal(t, rec, got)
}

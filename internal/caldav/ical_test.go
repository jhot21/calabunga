package caldav_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"calabunga/internal/caldav"
	"calabunga/internal/types"
)

const sampleVEvent = `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:test-uid-123@example.com
SUMMARY:Team standup
DTSTART:20260603T100000Z
DTEND:20260603T103000Z
DESCRIPTION:Daily sync
LOCATION:Conference Room
END:VEVENT
END:VCALENDAR`

func TestParseVCalendar(t *testing.T) {
	events, err := caldav.ParseVCalendar([]byte(sampleVEvent), "cal-1", `"etag-abc"`, time.Time{}, time.Time{})
	require.NoError(t, err)
	require.Len(t, events, 1)

	e := events[0]
	assert.Equal(t, "test-uid-123@example.com", e.UID)
	assert.Equal(t, "cal-1", e.CalendarID)
	assert.Equal(t, `"etag-abc"`, e.ETag)
	assert.Equal(t, "Team standup", e.Title)
	assert.Equal(t, "Daily sync", e.Description)
	assert.Equal(t, "Conference Room", e.Location)
	assert.False(t, e.AllDay)

	expectedStart := time.Date(2026, 6, 3, 10, 0, 0, 0, time.UTC)
	assert.Equal(t, expectedStart, e.Start)
}

func TestParseVCalendar_AllDay(t *testing.T) {
	ical := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:allday@example.com
SUMMARY:Holiday
DTSTART;VALUE=DATE:20260604
DTEND;VALUE=DATE:20260605
END:VEVENT
END:VCALENDAR`

	events, err := caldav.ParseVCalendar([]byte(ical), "cal-1", `"etag"`, time.Time{}, time.Time{})
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.True(t, events[0].AllDay)
}

func TestParseVCalendar_Recurring(t *testing.T) {
	icalData := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:pay-bills@example.com
SUMMARY:Pay Credit Card
DTSTART:20210310T100000Z
DTEND:20210310T101000Z
RRULE:FREQ=MONTHLY;BYMONTHDAY=10
END:VEVENT
END:VCALENDAR`

	rangeStart := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

	events, err := caldav.ParseVCalendar([]byte(icalData), "cal-1", `"etag"`, rangeStart, rangeEnd)
	require.NoError(t, err)
	require.Len(t, events, 1, "should return the June 2026 occurrence, not the 2021 master")
	assert.Equal(t, "pay-bills@example.com", events[0].UID)
	assert.Equal(t, 2026, events[0].Start.Year())
	assert.Equal(t, time.June, events[0].Start.Month())
	assert.Equal(t, 10, events[0].Start.Day())
}

func TestEventToVCalendar(t *testing.T) {
	e := types.Event{
		UID:         "test-uid@example.com",
		CalendarID:  "cal-1",
		Title:       "Team standup",
		Start:       time.Date(2026, 6, 3, 10, 0, 0, 0, time.UTC),
		End:         time.Date(2026, 6, 3, 10, 30, 0, 0, time.UTC),
		AllDay:      false,
		Description: "Daily sync",
	}

	ical, err := caldav.EventToVCalendar(e)
	require.NoError(t, err)
	assert.Contains(t, string(ical), "BEGIN:VCALENDAR")
	assert.Contains(t, string(ical), "UID:test-uid@example.com")
	assert.Contains(t, string(ical), "SUMMARY:Team standup")
	assert.Contains(t, string(ical), "DTSTART:20260603T100000Z")
}

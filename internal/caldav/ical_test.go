package caldav_test

import (
	"strings"
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

const recurringVEvent = `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:series@example.com
SUMMARY:Standup
DTSTART:20260602T100000Z
DTEND:20260602T103000Z
RRULE:FREQ=WEEKLY;BYDAY=TU
END:VEVENT
END:VCALENDAR`

func TestParseVCalendar_RecurringEventsCarryTheirRule(t *testing.T) {
	events, err := caldav.ParseVCalendar([]byte(recurringVEvent), "cal-1", `"e"`,
		time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.Greater(t, len(events), 1)
	for _, e := range events {
		require.NotNil(t, e.Recurrence)
		assert.Equal(t, "weekly", e.Recurrence.Frequency)
		assert.Equal(t, []string{"TU"}, e.Recurrence.ByDay)
	}
}

func TestEventToVCalendar_WritesRecurrenceFromUI(t *testing.T) {
	start := time.Date(2026, 6, 3, 10, 0, 0, 0, time.UTC)
	out, err := caldav.EventToVCalendar(types.Event{
		UID: "new@example.com", Title: "Standup", Start: start, End: start.Add(time.Hour),
		Recurrence: &types.Recurrence{Frequency: "weekly", Interval: 1, ByDay: []string{"WE"}, End: types.RecurrenceEnd{Type: "never"}},
	})
	require.NoError(t, err)

	assert.Contains(t, string(out), "RRULE:FREQ=WEEKLY;BYDAY=WE")
}

func TestEventToVCalendar_RejectsAnInvalidRecurrence(t *testing.T) {
	start := time.Date(2026, 6, 3, 10, 0, 0, 0, time.UTC)
	_, err := caldav.EventToVCalendar(types.Event{
		UID: "new@example.com", Title: "x", Start: start, End: start.Add(time.Hour),
		Recurrence: &types.Recurrence{Frequency: "hourly", Interval: 1},
	})
	assert.ErrorIs(t, err, caldav.ErrInvalidRecurrence)
}

func june(fixture string) ([]types.Event, error) {
	return caldav.ParseVCalendar([]byte(fixture), "cal-1", `"e"`,
		time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC))
}

func starts(events []types.Event) []string {
	var out []string
	for _, e := range events {
		out = append(out, e.Start.UTC().Format("0102T1504"))
	}
	return out
}

const exdateFixture = `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:ex@example.com
SUMMARY:Standup
DTSTART:20260602T100000Z
DTEND:20260602T103000Z
RRULE:FREQ=WEEKLY;BYDAY=TU
EXDATE:20260616T100000Z
END:VEVENT
END:VCALENDAR`

func TestParseVCalendar_ExdateRemovesThatOccurrence(t *testing.T) {
	events, err := june(exdateFixture)
	require.NoError(t, err)
	assert.Equal(t, []string{"0602T1000", "0609T1000", "0623T1000"}, starts(events))
}

func TestParseVCalendar_ExdateWithSeveralValues(t *testing.T) {
	fixture := strings.Replace(exdateFixture, "EXDATE:20260616T100000Z", "EXDATE:20260616T100000Z,20260623T100000Z", 1)
	events, err := june(fixture)
	require.NoError(t, err)
	assert.Equal(t, []string{"0602T1000", "0609T1000"}, starts(events))
}

func TestParseVCalendar_RdateAddsOccurrences(t *testing.T) {
	fixture := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:rd@example.com
SUMMARY:Irregular
DTSTART:20260602T100000Z
DTEND:20260602T110000Z
RDATE:20260610T100000Z,20260620T100000Z
END:VEVENT
END:VCALENDAR`
	events, err := june(fixture)
	require.NoError(t, err)
	assert.Equal(t, []string{"0602T1000", "0610T1000", "0620T1000"}, starts(events))
}

const overrideFixture = `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:ov@example.com
SUMMARY:Standup
DTSTART:20260602T100000Z
DTEND:20260602T103000Z
RRULE:FREQ=WEEKLY;BYDAY=TU
END:VEVENT
BEGIN:VEVENT
UID:ov@example.com
RECURRENCE-ID:20260616T100000Z
SUMMARY:Standup (moved)
DTSTART:20260617T140000Z
DTEND:20260617T143000Z
END:VEVENT
END:VCALENDAR`

func TestParseVCalendar_OverrideReplacesItsOriginalSlot(t *testing.T) {
	events, err := june(overrideFixture)
	require.NoError(t, err)

	// June 16 10:00 is gone; the moved occurrence appears once, on June 17.
	assert.ElementsMatch(t, []string{"0602T1000", "0609T1000", "0623T1000", "0617T1400"}, starts(events))

	var moved types.Event
	for _, e := range events {
		if e.Title == "Standup (moved)" {
			moved = e
		}
	}
	require.NotEmpty(t, moved.UID)
	assert.NotEmpty(t, moved.RecurrenceID, "overrides must be marked so they are not edited as the series")
}

func TestParseVCalendar_OverrideOutsideTheRangeIsLeftOut(t *testing.T) {
	fixture := strings.Replace(overrideFixture, "DTSTART:20260617T140000Z", "DTSTART:20260817T140000Z", 1)
	fixture = strings.Replace(fixture, "DTEND:20260617T143000Z", "DTEND:20260817T143000Z", 1)

	events, err := june(fixture)
	require.NoError(t, err)

	assert.Equal(t, []string{"0602T1000", "0609T1000", "0623T1000"}, starts(events))
}

func TestParseVCalendar_SeriesKeepsWallClockAcrossDST(t *testing.T) {
	fixture := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:dst@example.com
SUMMARY:Morning
DTSTART;TZID=America/Denver:20260303T100000
DTEND;TZID=America/Denver:20260303T110000
RRULE:FREQ=WEEKLY;BYDAY=TU
END:VEVENT
END:VCALENDAR`
	events, err := caldav.ParseVCalendar([]byte(fixture), "c", `"e"`,
		time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)

	// DST began March 8: 10:00 local is 17:00Z before it and 16:00Z after.
	assert.Equal(t, []string{"0303T1700", "0310T1600", "0317T1600"}, starts(events))
}

// TEXT values are escaped on the wire (RFC 5545 3.3.11): \, \; \\ and \n.
const escapedTextFixture = `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:text@example.com
SUMMARY:Lunch\, then walk
DESCRIPTION:Line one\nLine two\; with a \\ backslash
LOCATION:Cafe Bluebird\, 524 W Laurel St\, Fort Collins\, CO 80521
DTSTART:20260610T100000Z
DTEND:20260610T110000Z
END:VEVENT
END:VCALENDAR`

func TestParseVCalendar_UnescapesTextValues(t *testing.T) {
	e := parseOne(t, escapedTextFixture)

	assert.Equal(t, "Lunch, then walk", e.Title)
	assert.Equal(t, "Line one\nLine two; with a \\ backslash", e.Description)
	assert.Equal(t, "Cafe Bluebird, 524 W Laurel St, Fort Collins, CO 80521", e.Location)
}

func TestEventToVCalendar_EscapesTextValuesOnce(t *testing.T) {
	start := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	out, err := caldav.EventToVCalendar(types.Event{
		UID: "t@example.com", Title: "Lunch, then walk", Start: start, End: start.Add(time.Hour),
		Description: "Line one\nLine two; ok", Location: "Cafe Bluebird, 524 W Laurel St",
	})
	require.NoError(t, err)

	assert.Contains(t, string(out), `LOCATION:Cafe Bluebird\, 524 W Laurel St`)
	assert.Contains(t, string(out), `SUMMARY:Lunch\, then walk`)
	assert.Contains(t, string(out), `DESCRIPTION:Line one\nLine two\; ok`)
}

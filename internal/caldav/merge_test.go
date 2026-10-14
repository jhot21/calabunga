package caldav_test

import (
	"testing"
	"time"

	"calabunga/internal/caldav"
	"calabunga/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const mergeTZFixture = `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Other Client//EN
BEGIN:VTIMEZONE
TZID:America/Denver
BEGIN:STANDARD
DTSTART:19701101T020000
TZOFFSETFROM:-0600
TZOFFSETTO:-0700
END:STANDARD
BEGIN:DAYLIGHT
DTSTART:19700308T020000
TZOFFSETFROM:-0700
TZOFFSETTO:-0600
END:DAYLIGHT
END:VTIMEZONE
BEGIN:VEVENT
UID:merge-uid@example.com
SUMMARY:Dentist
DESCRIPTION:Bring forms
DTSTART;TZID=America/Denver:20260603T100000
DTEND;TZID=America/Denver:20260603T110000
ATTENDEE:mailto:bob@example.com
X-CUSTOM-FLAG:keepme
SEQUENCE:2
BEGIN:VALARM
ACTION:DISPLAY
DESCRIPTION:Reminder
TRIGGER:-PT15M
END:VALARM
END:VEVENT
END:VCALENDAR`

const mergeAllDayFixture = `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:allday-uid@example.com
SUMMARY:Holiday
DTSTART;VALUE=DATE:20260603
DTEND;VALUE=DATE:20260604
END:VEVENT
END:VCALENDAR`

const mergeDurationFixture = `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:duration-uid@example.com
SUMMARY:Call
DTSTART:20260603T100000Z
DURATION:PT1H
END:VEVENT
END:VCALENDAR`

const mergeSeriesFixture = `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:series-uid@example.com
SUMMARY:Trash Bins
DTSTART:20260602T230000Z
DTEND:20260602T233000Z
RRULE:FREQ=WEEKLY;BYDAY=TU
BEGIN:VALARM
ACTION:DISPLAY
DESCRIPTION:Reminder
TRIGGER:-PT1H
END:VALARM
END:VEVENT
END:VCALENDAR`

func parseOne(t *testing.T, data string) types.Event {
	t.Helper()
	events, err := caldav.ParseVCalendar([]byte(data), "cal-1", `"e"`, time.Time{}, time.Time{})
	require.NoError(t, err)
	require.Len(t, events, 1)
	return events[0]
}

func TestMergeEvent_KeepsDataCalabungaDoesNotManage(t *testing.T) {
	e := parseOne(t, mergeTZFixture)
	e.Title = "Dentist (moved)"

	out, err := caldav.MergeEvent([]byte(mergeTZFixture), e)
	require.NoError(t, err)
	got := string(out)

	assert.Contains(t, got, "SUMMARY:Dentist (moved)")
	assert.Contains(t, got, "BEGIN:VTIMEZONE")
	assert.Contains(t, got, "BEGIN:VALARM")
	assert.Contains(t, got, "TRIGGER:-PT15M")
	assert.Contains(t, got, "ATTENDEE:mailto:bob@example.com")
	assert.Contains(t, got, "X-CUSTOM-FLAG:keepme")
	assert.Contains(t, got, "DESCRIPTION:Bring forms")
	assert.Contains(t, got, "SEQUENCE:3")
}

func TestMergeEvent_UnchangedTimesKeepOriginalZone(t *testing.T) {
	e := parseOne(t, mergeTZFixture)
	e.Title = "Renamed"

	out, err := caldav.MergeEvent([]byte(mergeTZFixture), e)
	require.NoError(t, err)

	assert.Contains(t, string(out), "DTSTART;TZID=America/Denver:20260603T100000")
	assert.Contains(t, string(out), "DTEND;TZID=America/Denver:20260603T110000")
}

func TestMergeEvent_MovedTimeIsWrittenInOriginalZone(t *testing.T) {
	e := parseOne(t, mergeTZFixture)
	// The editor hands back UTC instants: one hour later is 11:00 in Denver.
	e.Start = e.Start.Add(time.Hour).UTC()
	e.End = e.End.Add(time.Hour).UTC()

	out, err := caldav.MergeEvent([]byte(mergeTZFixture), e)
	require.NoError(t, err)

	assert.Contains(t, string(out), "DTSTART;TZID=America/Denver:20260603T110000")
	assert.Contains(t, string(out), "DTEND;TZID=America/Denver:20260603T120000")
}

func TestMergeEvent_ClearedDescriptionIsRemoved(t *testing.T) {
	e := parseOne(t, mergeTZFixture)
	e.Description = ""

	out, err := caldav.MergeEvent([]byte(mergeTZFixture), e)
	require.NoError(t, err)

	assert.NotContains(t, string(out), "DESCRIPTION:Bring forms")
	assert.Contains(t, string(out), "TRIGGER:-PT15M", "the alarm's own DESCRIPTION must not be confused with the event's")
}

func TestMergeEvent_AllDayUnchangedKeepsDateValues(t *testing.T) {
	e := parseOne(t, mergeAllDayFixture)
	e.Title = "Holiday!"

	out, err := caldav.MergeEvent([]byte(mergeAllDayFixture), e)
	require.NoError(t, err)

	assert.Contains(t, string(out), "DTSTART;VALUE=DATE:20260603")
	assert.Contains(t, string(out), "DTEND;VALUE=DATE:20260604")
}

func TestMergeEvent_DurationKeptWhenTimesUnchanged(t *testing.T) {
	e := parseOne(t, mergeDurationFixture)
	assert.Equal(t, e.Start.Add(time.Hour), e.End, "a DURATION-only event must read with its end time")
	e.Title = "Call (renamed)"

	out, err := caldav.MergeEvent([]byte(mergeDurationFixture), e)
	require.NoError(t, err)

	assert.Contains(t, string(out), "DURATION:PT1H")
	assert.NotContains(t, string(out), "DTEND")
}

func TestMergeEvent_ChangedEndReplacesDuration(t *testing.T) {
	e := parseOne(t, mergeDurationFixture)
	e.End = e.End.Add(30 * time.Minute)

	out, err := caldav.MergeEvent([]byte(mergeDurationFixture), e)
	require.NoError(t, err)

	assert.NotContains(t, string(out), "DURATION")
	assert.Contains(t, string(out), "DTEND:20260603T113000Z")
}

func seriesOccurrence(t *testing.T) types.Event {
	t.Helper()
	rs := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	re := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	events, err := caldav.ParseVCalendar([]byte(mergeSeriesFixture), "cal-1", `"e"`, rs, re)
	require.NoError(t, err)
	require.Greater(t, len(events), 2)
	return events[2] // 2026-06-16, not the master's own date
}

func TestMergeEvent_SeriesEditKeepsRuleAlarmAndMasterDate(t *testing.T) {
	e := seriesOccurrence(t)
	require.Equal(t, 16, e.Start.Day())
	e.Title = "Trash Bins (renamed)"

	out, err := caldav.MergeEvent([]byte(mergeSeriesFixture), e)
	require.NoError(t, err)
	got := string(out)

	assert.Contains(t, got, "SUMMARY:Trash Bins (renamed)")
	assert.Contains(t, got, "RRULE:FREQ=WEEKLY;BYDAY=TU")
	assert.Contains(t, got, "TRIGGER:-PT1H")
	assert.Contains(t, got, "DTSTART:20260602T230000Z", "editing one occurrence must not move the series start")
	assert.Contains(t, got, "DTEND:20260602T233000Z")
}

func TestMergeEvent_SeriesClockChangeAppliesToMasterDate(t *testing.T) {
	e := seriesOccurrence(t)
	e.Start = e.Start.Add(-time.Hour)
	e.End = e.End.Add(-time.Hour)

	out, err := caldav.MergeEvent([]byte(mergeSeriesFixture), e)
	require.NoError(t, err)

	assert.Contains(t, string(out), "DTSTART:20260602T220000Z")
	assert.Contains(t, string(out), "DTEND:20260602T223000Z")
	assert.Contains(t, string(out), "RRULE:FREQ=WEEKLY;BYDAY=TU")
}

func TestMergeEvent_UnknownUIDIsAnError(t *testing.T) {
	e := parseOne(t, mergeTZFixture)
	e.UID = "someone-else@example.com"

	_, err := caldav.MergeEvent([]byte(mergeTZFixture), e)
	assert.Error(t, err)
}

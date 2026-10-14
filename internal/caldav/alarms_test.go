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

// Alarms calabunga manages (DISPLAY, N whole minutes before the start) next to
// ones it must leave exactly as stored.
const alarmsFixture = `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Other Client//EN
BEGIN:VEVENT
UID:alarms@example.com
SUMMARY:Dentist
DTSTART:20260610T100000Z
DTEND:20260610T110000Z
BEGIN:VALARM
ACTION:DISPLAY
DESCRIPTION:Reminder
TRIGGER:-PT15M
END:VALARM
BEGIN:VALARM
ACTION:DISPLAY
DESCRIPTION:Reminder
TRIGGER:-P1D
END:VALARM
BEGIN:VALARM
ACTION:EMAIL
SUMMARY:Mail me
DESCRIPTION:Reminder
ATTENDEE:mailto:me@example.com
TRIGGER:-PT30M
END:VALARM
BEGIN:VALARM
ACTION:DISPLAY
DESCRIPTION:After the end
TRIGGER;RELATED=END:PT5M
END:VALARM
BEGIN:VALARM
ACTION:DISPLAY
DESCRIPTION:Absolute
TRIGGER;VALUE=DATE-TIME:20260610T080000Z
END:VALARM
BEGIN:VALARM
ACTION:DISPLAY
DESCRIPTION:Nag
TRIGGER:-PT10M
REPEAT:3
DURATION:PT5M
END:VALARM
END:VEVENT
END:VCALENDAR`

func TestParseVCalendar_ReadsSimpleAlarmsOnly(t *testing.T) {
	e := parseOne(t, alarmsFixture)
	assert.ElementsMatch(t, []int{15, 1440}, e.Alarms)
}

func TestParseVCalendar_NoAlarmsIsAnEmptyListNotNull(t *testing.T) {
	e := parseOne(t, mergeDurationFixture)
	require.NotNil(t, e.Alarms, "the field is always present so clients can tell 'none' from 'unknown'")
	assert.Empty(t, e.Alarms)
}

func TestEventToVCalendar_WritesAlarms(t *testing.T) {
	start := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	out, err := caldav.EventToVCalendar(types.Event{
		UID: "new@example.com", Title: "Standup", Start: start, End: start.Add(time.Hour),
		Alarms: []int{10, 60, 1440, 10080, 0, 10}, // 10 is listed twice
	})
	require.NoError(t, err)
	got := string(out)

	assert.Equal(t, 5, strings.Count(got, "BEGIN:VALARM"), "duplicates collapse")
	for _, trigger := range []string{"TRIGGER:-PT10M", "TRIGGER:-PT1H", "TRIGGER:-P1D", "TRIGGER:-P1W", "TRIGGER:PT0S"} {
		assert.Contains(t, got, trigger)
	}
	assert.Contains(t, got, "ACTION:DISPLAY")
	assert.Contains(t, got, "DESCRIPTION:Standup", "a DISPLAY alarm needs a description")
}

func TestEventToVCalendar_RejectsInvalidAlarms(t *testing.T) {
	start := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	for _, alarms := range [][]int{{-5}, {40321}, make([]int, 11)} {
		_, err := caldav.EventToVCalendar(types.Event{
			UID: "x@example.com", Title: "x", Start: start, End: start.Add(time.Hour), Alarms: alarms,
		})
		assert.ErrorIs(t, err, caldav.ErrInvalidAlarm, "%v", alarms)
	}
}

func TestMergeEvent_AlarmsAbsentLeavesStoredAlarmsAlone(t *testing.T) {
	e := parseOne(t, alarmsFixture)
	e.Alarms = nil // a client that doesn't know about alarms
	e.Title = "Renamed"

	out, err := caldav.MergeEvent([]byte(alarmsFixture), e)
	require.NoError(t, err)

	assert.Equal(t, 6, strings.Count(string(out), "BEGIN:VALARM"))
}

func TestMergeEvent_UnchangedAlarmsAreKeptVerbatim(t *testing.T) {
	e := parseOne(t, alarmsFixture)
	e.Title = "Renamed"

	out, err := caldav.MergeEvent([]byte(alarmsFixture), e)
	require.NoError(t, err)

	got := string(out)
	assert.Equal(t, 6, strings.Count(got, "BEGIN:VALARM"))
	assert.Contains(t, got, "TRIGGER:-P1D", "the original form is untouched, not rewritten as hours")
}

func TestMergeEvent_ChangedAlarmsReplaceOnlyTheManagedOnes(t *testing.T) {
	e := parseOne(t, alarmsFixture)
	e.Alarms = []int{5, 120}

	out, err := caldav.MergeEvent([]byte(alarmsFixture), e)
	require.NoError(t, err)
	got := string(out)

	assert.Contains(t, got, "TRIGGER:-PT5M")
	assert.Contains(t, got, "TRIGGER:-PT2H")
	assert.NotContains(t, got, "TRIGGER:-PT15M")
	assert.NotContains(t, got, "TRIGGER:-P1D")
	// everything calabunga doesn't manage survives
	assert.Contains(t, got, "ACTION:EMAIL")
	assert.Contains(t, got, "ATTENDEE:mailto:me@example.com")
	assert.Contains(t, got, "TRIGGER;RELATED=END:PT5M")
	assert.Contains(t, got, "TRIGGER;VALUE=DATE-TIME:20260610T080000Z")
	assert.Contains(t, got, "REPEAT:3")
}

func TestMergeEvent_EmptyAlarmListRemovesOnlyTheManagedOnes(t *testing.T) {
	e := parseOne(t, alarmsFixture)
	e.Alarms = []int{}

	out, err := caldav.MergeEvent([]byte(alarmsFixture), e)
	require.NoError(t, err)
	got := string(out)

	assert.NotContains(t, got, "TRIGGER:-PT15M")
	assert.NotContains(t, got, "TRIGGER:-P1D")
	assert.Equal(t, 4, strings.Count(got, "BEGIN:VALARM"))
}

func TestMergeEvent_InvalidAlarmsAreRejected(t *testing.T) {
	e := parseOne(t, alarmsFixture)
	e.Alarms = []int{-1}

	_, err := caldav.MergeEvent([]byte(alarmsFixture), e)
	assert.ErrorIs(t, err, caldav.ErrInvalidAlarm)
}

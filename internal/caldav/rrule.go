package caldav

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	rrule "github.com/teambition/rrule-go"
	"calabunga/internal/types"
)

type Occurrence struct {
	Start time.Time
	End   time.Time
}

func ExpandRecurrence(start, end time.Time, rec *types.Recurrence) ([]Occurrence, error) {
	far := start.Add(50 * 365 * 24 * time.Hour)
	return ExpandRecurrenceInRange(start, end, rec, start, far)
}

func ExpandRecurrenceInRange(start, end time.Time, rec *types.Recurrence, rangeStart, rangeEnd time.Time) ([]Occurrence, error) {
	rSet, err := buildRRuleSet(start, rec)
	if err != nil {
		return nil, err
	}

	duration := end.Sub(start)
	var result []Occurrence
	for _, t := range rSet.Between(rangeStart, rangeEnd, true) {
		if len(result) >= 500 {
			break
		}
		result = append(result, Occurrence{Start: t, End: t.Add(duration)})
	}
	return result, nil
}

func RecurrenceToRRule(rec types.Recurrence) (string, error) {
	var parts []string

	freq := strings.ToUpper(rec.Frequency)
	switch freq {
	case "DAILY", "WEEKLY", "MONTHLY", "YEARLY":
		parts = append(parts, "FREQ="+freq)
	default:
		return "", fmt.Errorf("unsupported frequency: %s", rec.Frequency)
	}

	if rec.Interval > 1 {
		parts = append(parts, fmt.Sprintf("INTERVAL=%d", rec.Interval))
	}

	if len(rec.ByDay) > 0 {
		parts = append(parts, "BYDAY="+strings.Join(rec.ByDay, ","))
	}

	if len(rec.ByMonthDay) > 0 {
		days := make([]string, len(rec.ByMonthDay))
		for i, d := range rec.ByMonthDay {
			days[i] = strconv.Itoa(d)
		}
		parts = append(parts, "BYMONTHDAY="+strings.Join(days, ","))
	}

	switch rec.End.Type {
	case "count":
		parts = append(parts, "COUNT="+rec.End.Value)
	case "date":
		t, err := time.Parse("2006-01-02", rec.End.Value)
		if err != nil {
			return "", fmt.Errorf("invalid end date: %w", err)
		}
		parts = append(parts, "UNTIL="+t.UTC().Format("20060102T150405Z"))
	case "never", "":
	default:
		return "", fmt.Errorf("unsupported end type: %s", rec.End.Type)
	}

	return strings.Join(parts, ";"), nil
}

func buildRRuleSet(dtstart time.Time, rec *types.Recurrence) (*rrule.Set, error) {
	rruleStr, err := RecurrenceToRRule(*rec)
	if err != nil {
		return nil, err
	}

	r, err := rrule.StrToRRule("DTSTART:" + dtstart.UTC().Format("20060102T150405Z") + "\nRRULE:" + rruleStr)
	if err != nil {
		return nil, fmt.Errorf("build rrule: %w", err)
	}

	set := &rrule.Set{}
	set.RRule(r)
	return set, nil
}

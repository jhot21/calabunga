package caldav

import (
	"errors"
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

// ErrInvalidRecurrence marks a recurrence the editor sent that can't be written as an RRULE.
var ErrInvalidRecurrence = errors.New("invalid recurrence")

func RecurrenceToRRule(rec types.Recurrence) (string, error) {
	return RecurrenceToRRuleFor(rec, false)
}

// RecurrenceToRRuleFor builds the RRULE value. allDay selects the UNTIL form:
// RFC 5545 requires a DATE when DTSTART is a DATE.
func RecurrenceToRRuleFor(rec types.Recurrence, allDay bool) (string, error) {
	var parts []string

	freq := strings.ToUpper(rec.Frequency)
	switch freq {
	case "DAILY", "WEEKLY", "MONTHLY", "YEARLY":
		parts = append(parts, "FREQ="+freq)
	default:
		return "", fmt.Errorf("%w: unsupported frequency %q", ErrInvalidRecurrence, rec.Frequency)
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
		if n, err := strconv.Atoi(rec.End.Value); err != nil || n < 1 {
			return "", fmt.Errorf("%w: invalid count %q", ErrInvalidRecurrence, rec.End.Value)
		}
		parts = append(parts, "COUNT="+rec.End.Value)
	case "date":
		t, err := time.Parse("2006-01-02", rec.End.Value)
		if err != nil {
			return "", fmt.Errorf("%w: invalid end date %q", ErrInvalidRecurrence, rec.End.Value)
		}
		if allDay {
			parts = append(parts, "UNTIL="+t.Format("20060102"))
		} else {
			// End of the day, so an occurrence on the last day isn't cut off.
			parts = append(parts, "UNTIL="+t.Format("20060102")+"T235959Z")
		}
	case "never", "":
	default:
		return "", fmt.Errorf("%w: unsupported end type %q", ErrInvalidRecurrence, rec.End.Type)
	}

	return strings.Join(parts, ";"), nil
}

var weekdays = map[string]bool{"SU": true, "MO": true, "TU": true, "WE": true, "TH": true, "FR": true, "SA": true}

// ParseRRule converts an RRULE value into the editor's model. ok is false when
// the rule uses anything the editor can't express (ordinal weekdays, BYSETPOS,
// BYMONTH, WKST, sub-daily frequencies, ...); callers must then keep the stored
// rule untouched rather than round-tripping it through the model.
func ParseRRule(value string) (rec types.Recurrence, ok bool) {
	rec = types.Recurrence{Interval: 1, End: types.RecurrenceEnd{Type: "never"}}
	if strings.TrimSpace(value) == "" {
		return rec, false
	}

	var sawCount, sawUntil bool
	for _, part := range strings.Split(value, ";") {
		key, val, found := strings.Cut(part, "=")
		if !found || val == "" {
			return rec, false
		}
		switch strings.ToUpper(key) {
		case "FREQ":
			f := strings.ToUpper(val)
			if f != "DAILY" && f != "WEEKLY" && f != "MONTHLY" && f != "YEARLY" {
				return rec, false
			}
			rec.Frequency = strings.ToLower(f)
		case "INTERVAL":
			n, err := strconv.Atoi(val)
			if err != nil || n < 1 {
				return rec, false
			}
			rec.Interval = n
		case "BYDAY":
			for _, d := range strings.Split(val, ",") {
				d = strings.ToUpper(d)
				if !weekdays[d] {
					return rec, false
				}
				rec.ByDay = append(rec.ByDay, d)
			}
		case "BYMONTHDAY":
			for _, d := range strings.Split(val, ",") {
				n, err := strconv.Atoi(d)
				if err != nil || n < 1 || n > 31 {
					return rec, false
				}
				rec.ByMonthDay = append(rec.ByMonthDay, n)
			}
		case "COUNT":
			n, err := strconv.Atoi(val)
			if err != nil || n < 1 {
				return rec, false
			}
			sawCount = true
			rec.End = types.RecurrenceEnd{Type: "count", Value: val}
		case "UNTIL":
			day := val
			if i := strings.IndexByte(val, 'T'); i >= 0 {
				day = val[:i]
			}
			t, err := time.Parse("20060102", day)
			if err != nil {
				return rec, false
			}
			sawUntil = true
			rec.End = types.RecurrenceEnd{Type: "date", Value: t.Format("2006-01-02")}
		default:
			return rec, false
		}
	}

	if rec.Frequency == "" || (sawCount && sawUntil) {
		return rec, false
	}
	return rec, true
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

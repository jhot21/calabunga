package caldav

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"calabunga/internal/types"
	ical "github.com/emersion/go-ical"
)

// ErrUnsupportedChange is returned when an edit can't be applied without
// risking the stored event (for example, toggling all-day on a recurring series).
var ErrUnsupportedChange = errors.New("this change to the event is not supported yet")

// MergeEvent applies the fields calabunga lets users edit onto the existing
// calendar object and leaves everything else alone: alarms, attendees,
// recurrence rules and overrides, VTIMEZONE definitions, and X- properties
// written by other clients. Rebuilding the object from scratch would delete them.
func MergeEvent(existing []byte, e types.Event) ([]byte, error) {
	cal, err := ical.NewDecoder(bytes.NewReader(existing)).Decode()
	if err != nil {
		return nil, fmt.Errorf("parse existing event: %w", err)
	}

	// The encoder insists on these; some clients and servers omit them.
	if cal.Props.Get(ical.PropVersion) == nil {
		cal.Props.SetText(ical.PropVersion, "2.0")
	}
	if cal.Props.Get(ical.PropProductID) == nil {
		cal.Props.SetText(ical.PropProductID, "-//Calabunga//EN")
	}

	if e.RecurrenceID != "" {
		// A modified occurrence is its own VEVENT; applying its fields to the
		// series would rewrite every other occurrence.
		return nil, ErrUnsupportedChange
	}

	master := findMaster(cal, e.UID)
	if master == nil {
		return nil, fmt.Errorf("no VEVENT with UID %q in calendar object", e.UID)
	}

	cur, err := vEventToEvent(master, e.CalendarID, e.ETag)
	if err != nil {
		return nil, err
	}

	// Decided before the rule is touched: an edit from an occurrence of a series
	// must not move the series start, even if the user is turning repeat off.
	wasSeries := master.Props.Get(ical.PropRecurrenceRule) != nil

	master.Props.SetText(ical.PropSummary, e.Title)
	setOptionalText(master, ical.PropDescription, e.Description)
	setOptionalText(master, ical.PropLocation, e.Location)
	if err := mergeTimes(master, cur, e, wasSeries); err != nil {
		return nil, err
	}
	if err := mergeRecurrence(master, e); err != nil {
		return nil, err
	}
	if err := mergeAlarms(master, e.Alarms, e.Title); err != nil {
		return nil, err
	}
	master.Props.SetDateTime(ical.PropDateTimeStamp, time.Now().UTC())
	bumpSequence(master)

	var buf bytes.Buffer
	if err := ical.NewEncoder(&buf).Encode(cal); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// findMaster returns the VEVENT for uid that is not a RECURRENCE-ID override.
func findMaster(cal *ical.Calendar, uid string) *ical.Component {
	for _, child := range cal.Children {
		if child.Name != ical.CompEvent || child.Props.Get(ical.PropRecurrenceID) != nil {
			continue
		}
		if p := child.Props.Get(ical.PropUID); p != nil && p.Value == uid {
			return child
		}
	}
	return nil
}

func setOptionalText(comp *ical.Component, name, value string) {
	if value == "" {
		comp.Props.Del(name)
		return
	}
	comp.Props.SetText(name, value)
}

func mergeTimes(master *ical.Component, cur, e types.Event, wasSeries bool) error {
	if cur.AllDay == e.AllDay && cur.Start.Equal(e.Start) && cur.End.Equal(e.End) {
		return nil // untouched: keeps TZID, DURATION, and VALUE=DATE exactly as stored
	}

	start, end := e.Start, e.End

	if wasSeries {
		// The editor hands back one expanded occurrence, not the series start.
		// Taking its date would collapse the series onto that day, so only the
		// time of day and the length are applied, on the series' own start date.
		if cur.AllDay != e.AllDay {
			return ErrUnsupportedChange
		}
		dur := e.End.Sub(e.Start)
		loc := cur.Start.Location()

		var shift time.Duration // wall-clock change within the day
		if e.OriginalStart != nil {
			o, n := e.OriginalStart.In(loc), e.Start.In(loc)
			if o.Year() != n.Year() || o.YearDay() != n.YearDay() {
				// Moving a series to another day would leave DTSTART out of
				// step with BYDAY/BYMONTHDAY.
				return ErrUnsupportedChange
			}
			shift = wall(n).Sub(wall(o))
		} else {
			es := e.Start.In(loc)
			shift = wall(es).Sub(wall(time.Date(es.Year(), es.Month(), es.Day(), cur.Start.Hour(), cur.Start.Minute(), cur.Start.Second(), 0, loc)))
		}
		if e.AllDay {
			shift = 0
		}

		if shift == 0 && dur == cur.End.Sub(cur.Start) {
			return nil
		}
		w := wall(cur.Start).Add(shift)
		start = time.Date(w.Year(), w.Month(), w.Day(), w.Hour(), w.Minute(), w.Second(), 0, loc)
		end = start.Add(dur)
	}

	if e.AllDay {
		master.Props.SetDate(ical.PropDateTimeStart, start)
		master.Props.SetDate(ical.PropDateTimeEnd, end)
	} else {
		// Write back in the zone the event was stored in; the editor sends UTC.
		loc := cur.Start.Location()
		master.Props.SetDateTime(ical.PropDateTimeStart, start.In(loc))
		master.Props.SetDateTime(ical.PropDateTimeEnd, end.In(loc))
	}
	master.Props.Del(ical.PropDuration)
	return nil
}

// wall returns t's wall-clock fields as a UTC time, so differences between two
// times ignore their zone offsets (and DST changes between them).
func wall(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, time.UTC)
}

// mergeRecurrence applies the editor's recurrence. A stored rule the editor
// can't express is left alone unless the user sets a new one.
func mergeRecurrence(master *ical.Component, e types.Event) error {
	existing := master.Props.Get(ical.PropRecurrenceRule)

	if e.Recurrence == nil {
		if existing == nil {
			return nil
		}
		if _, ok := ParseRRule(existing.Value); !ok {
			return nil // the editor couldn't show it, so it wasn't removed on purpose
		}
		master.Props.Del(ical.PropRecurrenceRule)
		master.Props.Del(ical.PropExceptionDates)
		master.Props.Del(ical.PropRecurrenceDates)
		return nil
	}

	rule, err := RecurrenceToRRuleFor(*e.Recurrence, e.AllDay)
	if err != nil {
		return err
	}
	if existing != nil {
		if stored, ok := ParseRRule(existing.Value); ok && sameRecurrence(stored, *e.Recurrence) {
			return nil // unchanged: keep the stored text exactly
		}
	}
	master.Props.Set(rruleProp(rule))
	return nil
}

func sameRecurrence(a, b types.Recurrence) bool {
	norm := func(r types.Recurrence) types.Recurrence {
		if r.Interval < 1 {
			r.Interval = 1
		}
		if r.End.Type == "" {
			r.End.Type = "never"
		}
		r.Frequency = strings.ToLower(r.Frequency)
		r.ByDay = append([]string(nil), r.ByDay...)
		sort.Strings(r.ByDay)
		r.ByMonthDay = append([]int(nil), r.ByMonthDay...)
		sort.Ints(r.ByMonthDay)
		return r
	}
	return reflect.DeepEqual(norm(a), norm(b))
}

func bumpSequence(comp *ical.Component) {
	n := 0
	if p := comp.Props.Get(ical.PropSequence); p != nil {
		if v, err := p.Int(); err == nil {
			n = v
		}
	}
	p := ical.NewProp(ical.PropSequence)
	p.Value = strconv.Itoa(n + 1)
	comp.Props.Set(p)
}

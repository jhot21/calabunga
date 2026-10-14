package caldav

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
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

	master := findMaster(cal, e.UID)
	if master == nil {
		return nil, fmt.Errorf("no VEVENT with UID %q in calendar object", e.UID)
	}

	cur, err := vEventToEvent(master, e.CalendarID, e.ETag)
	if err != nil {
		return nil, err
	}

	master.Props.SetText(ical.PropSummary, e.Title)
	setOptionalText(master, ical.PropDescription, e.Description)
	setOptionalText(master, ical.PropLocation, e.Location)
	if err := mergeTimes(master, cur, e); err != nil {
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

func mergeTimes(master *ical.Component, cur, e types.Event) error {
	if cur.AllDay == e.AllDay && cur.Start.Equal(e.Start) && cur.End.Equal(e.End) {
		return nil // untouched: keeps TZID, DURATION, and VALUE=DATE exactly as stored
	}

	start, end := e.Start, e.End

	if master.Props.Get("RRULE") != nil {
		// The editor hands back one expanded occurrence, not the series start.
		// Taking its date would collapse the series onto that day, so only the
		// time of day and the length are applied, on the series' own start date.
		if cur.AllDay != e.AllDay {
			return ErrUnsupportedChange
		}
		dur := e.End.Sub(e.Start)
		if e.AllDay {
			start, end = cur.Start, cur.Start.Add(dur)
		} else {
			loc := cur.Start.Location()
			es := e.Start.In(loc)
			if dur == cur.End.Sub(cur.Start) && es.Hour() == cur.Start.Hour() &&
				es.Minute() == cur.Start.Minute() && es.Second() == cur.Start.Second() {
				return nil
			}
			start = time.Date(cur.Start.Year(), cur.Start.Month(), cur.Start.Day(), es.Hour(), es.Minute(), es.Second(), 0, loc)
			end = start.Add(dur)
		}
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

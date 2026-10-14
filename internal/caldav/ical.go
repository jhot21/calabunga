package caldav

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	ical "github.com/emersion/go-ical"
	rrulelib "github.com/teambition/rrule-go"
	"calabunga/internal/types"
)

// ParseVCalendar parses a VCALENDAR and returns Event objects.
// rangeStart/rangeEnd are used to expand recurring events to the occurrences
// that fall within the window; pass zero values when no range is needed.
func ParseVCalendar(data []byte, calendarID, etag string, rangeStart, rangeEnd time.Time) ([]types.Event, error) {
	dec := ical.NewDecoder(bytes.NewReader(data))
	cal, err := dec.Decode()
	if err != nil {
		return nil, fmt.Errorf("parse ical: %w", err)
	}

	// A resource holds one master VEVENT plus an override VEVENT (sharing its
	// UID, carrying RECURRENCE-ID) for each occurrence modified on its own.
	var masters, overrides []*ical.Component
	replaced := map[string][]time.Time{} // UID -> original slots taken over by overrides
	for _, child := range cal.Children {
		if child.Name != ical.CompEvent {
			continue
		}
		rid := child.Props.Get(ical.PropRecurrenceID)
		if rid == nil {
			masters = append(masters, child)
			continue
		}
		overrides = append(overrides, child)
		if uid := child.Props.Get(ical.PropUID); uid != nil {
			if ts := parseDateValues(rid); len(ts) > 0 {
				replaced[uid.Value] = append(replaced[uid.Value], ts[0])
			}
		}
	}

	var events []types.Event
	for _, child := range masters {
		uid := ""
		if p := child.Props.Get(ical.PropUID); p != nil {
			uid = p.Value
		}
		expanded, err := expandVEvent(child, calendarID, etag, rangeStart, rangeEnd, replaced[uid])
		if err != nil {
			return nil, err
		}
		events = append(events, expanded...)
	}
	for _, child := range overrides {
		ev, err := vEventToEvent(child, calendarID, etag)
		if err != nil {
			return nil, err
		}
		ev.RecurrenceID = child.Props.Get(ical.PropRecurrenceID).Value
		if !rangeStart.IsZero() && !rangeEnd.IsZero() && (!ev.Start.Before(rangeEnd) || ev.End.Before(rangeStart)) {
			continue
		}
		events = append(events, ev)
	}
	return events, nil
}

func expandVEvent(comp *ical.Component, calendarID, etag string, rangeStart, rangeEnd time.Time, replaced []time.Time) ([]types.Event, error) {
	base, err := vEventToEvent(comp, calendarID, etag)
	if err != nil {
		return nil, err
	}

	rruleProp := comp.Props.Get("RRULE")
	rdates := collectDates(comp, ical.PropRecurrenceDates)
	if (rruleProp == nil && len(rdates) == 0) || rangeStart.IsZero() || rangeEnd.IsZero() {
		return []types.Event{base}, nil
	}

	duration := base.End.Sub(base.Start)

	var set *rrulelib.Set
	if rruleProp != nil {
		// Build the DTSTART line, preserving TZID so the rrule library handles DST
		// correctly (e.g. America/Denver wall-clock 10:00 stays 10:00 across MST/MDT).
		tzid := base.Start.Location().String()
		var dtstart string
		if tzid == "" || tzid == "UTC" {
			dtstart = "DTSTART:" + base.Start.UTC().Format("20060102T150405Z")
		} else {
			dtstart = "DTSTART;TZID=" + tzid + ":" + base.Start.Format("20060102T150405")
		}

		set, err = rrulelib.StrToRRuleSet(dtstart + "\nRRULE:" + rruleProp.Value)
		if err != nil {
			// Unparseable RRULE — fall back to master event
			return []types.Event{base}, nil
		}
	} else {
		// RDATE-only: DTSTART is the first instance, the RDATEs are the rest.
		set = &rrulelib.Set{}
		set.DTStart(base.Start)
		set.RDate(base.Start)
	}
	for _, t := range rdates {
		set.RDate(t)
	}
	for _, t := range collectDates(comp, ical.PropExceptionDates) {
		set.ExDate(t)
	}
	for _, t := range replaced {
		set.ExDate(t)
	}

	var occurrences []types.Event
	for _, t := range set.Between(rangeStart.UTC(), rangeEnd.UTC(), true) {
		occ := base
		occ.Start = t
		occ.End = t.Add(duration)
		occurrences = append(occurrences, occ)
	}
	return occurrences, nil
}

// collectDates returns every date-time listed in the component's EXDATE or
// RDATE properties. Each property may list several comma-separated values.
func collectDates(comp *ical.Component, name string) []time.Time {
	var out []time.Time
	for i := range comp.Props[name] {
		out = append(out, parseDateValues(&comp.Props[name][i])...)
	}
	return out
}

// parseDateValues parses a property whose value is a comma-separated list of
// dates or date-times, honoring its TZID and VALUE parameters. Values that
// don't parse (such as RDATE periods) are skipped.
func parseDateValues(p *ical.Prop) []time.Time {
	var out []time.Time
	for _, v := range strings.Split(p.Value, ",") {
		one := ical.NewProp(p.Name)
		one.Params = p.Params
		one.Value = strings.TrimSpace(v)
		if t, err := one.DateTime(nil); err == nil {
			out = append(out, t)
		}
	}
	return out
}

func vEventToEvent(comp *ical.Component, calendarID, etag string) (types.Event, error) {
	e := types.Event{
		CalendarID: calendarID,
		ETag:       etag,
	}

	if uid := comp.Props.Get(ical.PropUID); uid != nil {
		e.UID = uid.Value
	}
	if sum := comp.Props.Get(ical.PropSummary); sum != nil {
		e.Title = sum.Value
	}
	if desc := comp.Props.Get(ical.PropDescription); desc != nil {
		e.Description = desc.Value
	}
	if loc := comp.Props.Get(ical.PropLocation); loc != nil {
		e.Location = loc.Value
	}

	if rr := comp.Props.Get(ical.PropRecurrenceRule); rr != nil {
		if rec, ok := ParseRRule(rr.Value); ok {
			e.Recurrence = &rec
		}
	}

	dtstart := comp.Props.Get(ical.PropDateTimeStart)
	if dtstart == nil {
		return e, fmt.Errorf("event %s missing DTSTART", e.UID)
	}

	if dtstart.ValueType() == ical.ValueDate {
		e.AllDay = true
		t, err := dtstart.DateTime(nil)
		if err != nil {
			return e, fmt.Errorf("parse DTSTART date: %w", err)
		}
		e.Start = t
		if dtend := comp.Props.Get(ical.PropDateTimeEnd); dtend != nil {
			t2, _ := dtend.DateTime(nil)
			e.End = t2
		}
	} else {
		t, err := dtstart.DateTime(nil)
		if err != nil {
			return e, fmt.Errorf("parse DTSTART: %w", err)
		}
		e.Start = t
		if dtend := comp.Props.Get(ical.PropDateTimeEnd); dtend != nil {
			t2, err := dtend.DateTime(nil)
			if err == nil {
				e.End = t2
			}
		}
	}

	// An event may carry DURATION instead of DTEND (RFC 5545 allows either).
	if e.End.IsZero() {
		if dur := comp.Props.Get(ical.PropDuration); dur != nil {
			if d, err := dur.Duration(); err == nil {
				e.End = e.Start.Add(d)
			}
		}
	}

	return e, nil
}

func EventToVCalendar(e types.Event) ([]byte, error) {
	cal := ical.NewCalendar()
	cal.Props.SetText(ical.PropVersion, "2.0")
	cal.Props.SetText(ical.PropProductID, "-//Calabunga//EN")

	ev := ical.NewEvent()
	ev.Props.SetText(ical.PropUID, e.UID)
	ev.Props.SetText(ical.PropSummary, e.Title)
	ev.Props.SetText(ical.PropDescription, e.Description)
	ev.Props.SetText(ical.PropLocation, e.Location)
	ev.Props.SetDateTime(ical.PropDateTimeStamp, time.Now().UTC())

	if e.AllDay {
		ev.Props.SetDate(ical.PropDateTimeStart, e.Start)
		ev.Props.SetDate(ical.PropDateTimeEnd, e.End)
	} else {
		ev.Props.SetDateTime(ical.PropDateTimeStart, e.Start)
		ev.Props.SetDateTime(ical.PropDateTimeEnd, e.End)
	}

	if e.Recurrence != nil {
		rule, err := RecurrenceToRRuleFor(*e.Recurrence, e.AllDay)
		if err != nil {
			return nil, err
		}
		ev.Props.Set(rruleProp(rule))
	}

	cal.Children = append(cal.Children, ev.Component)

	var buf bytes.Buffer
	enc := ical.NewEncoder(&buf)
	if err := enc.Encode(cal); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// rruleProp builds an RRULE property. The value is a rule, not text, so it must
// not go through SetText's escaping.
func rruleProp(rule string) *ical.Prop {
	p := ical.NewProp(ical.PropRecurrenceRule)
	p.Value = rule
	return p
}

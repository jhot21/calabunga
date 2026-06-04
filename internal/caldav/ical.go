package caldav

import (
	"bytes"
	"fmt"
	"time"

	ical "github.com/emersion/go-ical"
	"calabunga/internal/types"
)

func ParseVCalendar(data []byte, calendarID, etag string) ([]types.Event, error) {
	dec := ical.NewDecoder(bytes.NewReader(data))
	cal, err := dec.Decode()
	if err != nil {
		return nil, fmt.Errorf("parse ical: %w", err)
	}

	var events []types.Event
	for _, child := range cal.Children {
		if child.Name != ical.CompEvent {
			continue
		}
		e, err := vEventToEvent(child, calendarID, etag)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
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

	cal.Children = append(cal.Children, ev.Component)

	var buf bytes.Buffer
	enc := ical.NewEncoder(&buf)
	if err := enc.Encode(cal); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

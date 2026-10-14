package caldav

import (
	"errors"
	"fmt"
	"strings"
	"time"

	ical "github.com/emersion/go-ical"
)

// ErrInvalidAlarm marks an alarm list the editor sent that can't be written.
var ErrInvalidAlarm = errors.New("invalid alarm")

const (
	maxAlarmMinutes = 4 * 7 * 24 * 60 // four weeks, Google Calendar's limit
	maxAlarms       = 10
)

// managedAlarmMinutes reports how many minutes before the start a VALARM fires,
// if it is one calabunga manages: a DISPLAY alarm triggered a whole number of
// minutes before the start. Everything else (email, RELATED=END, absolute
// times, repeats, positive offsets) is left exactly as stored.
func managedAlarmMinutes(a *ical.Component) (int, bool) {
	action := a.Props.Get(ical.PropAction)
	if action == nil || !strings.EqualFold(action.Value, "DISPLAY") {
		return 0, false
	}
	if a.Props.Get(ical.PropRepeat) != nil || a.Props.Get(ical.PropDuration) != nil {
		return 0, false
	}
	trigger := a.Props.Get(ical.PropTrigger)
	if trigger == nil || trigger.ValueType() != ical.ValueDuration {
		return 0, false
	}
	if rel := trigger.Params.Get(ical.ParamRelated); rel != "" && !strings.EqualFold(rel, "START") {
		return 0, false
	}
	d, err := trigger.Duration()
	if err != nil || d > 0 || d%time.Minute != 0 {
		return 0, false
	}
	minutes := int(-d / time.Minute)
	if minutes > maxAlarmMinutes {
		return 0, false
	}
	return minutes, true
}

// alarmsOf returns the managed alarms of an event. It is never nil, so clients
// can tell "no alarms" apart from "unknown".
func alarmsOf(ev *ical.Component) []int {
	out := []int{}
	for _, child := range ev.Children {
		if child.Name != ical.CompAlarm {
			continue
		}
		if m, ok := managedAlarmMinutes(child); ok {
			out = append(out, m)
		}
	}
	return out
}

// normalizeAlarms validates the editor's list and drops duplicates.
func normalizeAlarms(minutes []int) ([]int, error) {
	if len(minutes) > maxAlarms {
		return nil, fmt.Errorf("%w: at most %d notifications", ErrInvalidAlarm, maxAlarms)
	}
	seen := map[int]bool{}
	out := make([]int, 0, len(minutes))
	for _, m := range minutes {
		if m < 0 || m > maxAlarmMinutes {
			return nil, fmt.Errorf("%w: %d minutes is out of range", ErrInvalidAlarm, m)
		}
		if !seen[m] {
			seen[m] = true
			out = append(out, m)
		}
	}
	return out, nil
}

func newAlarm(minutes int, description string) *ical.Component {
	if description == "" {
		description = "Reminder"
	}
	a := ical.NewComponent(ical.CompAlarm)
	a.Props.SetText(ical.PropAction, "DISPLAY")
	a.Props.SetText(ical.PropDescription, description)
	trigger := ical.NewProp(ical.PropTrigger)
	trigger.Value = triggerValue(minutes)
	a.Props.Set(trigger)
	return a
}

// triggerValue formats the offset the way other clients write it (-PT15M,
// -PT1H, -P1D) rather than as a count of seconds.
func triggerValue(minutes int) string {
	switch {
	case minutes == 0:
		return "PT0S"
	case minutes%(7*24*60) == 0:
		return fmt.Sprintf("-P%dW", minutes/(7*24*60))
	case minutes%(24*60) == 0:
		return fmt.Sprintf("-P%dD", minutes/(24*60))
	case minutes%60 == 0:
		return fmt.Sprintf("-PT%dH", minutes/60)
	default:
		return fmt.Sprintf("-PT%dM", minutes)
	}
}

// mergeAlarms applies the editor's alarm list to an existing event. A nil list
// means the client doesn't know about alarms, so the stored ones stay. Only
// managed alarms are replaced, and only when the set actually changed.
func mergeAlarms(master *ical.Component, e []int, title string) error {
	if e == nil {
		return nil
	}
	want, err := normalizeAlarms(e)
	if err != nil {
		return err
	}

	have := alarmsOf(master)
	if sameSet(have, want) {
		return nil
	}

	kept := master.Children[:0:0]
	for _, child := range master.Children {
		if child.Name == ical.CompAlarm {
			if _, managed := managedAlarmMinutes(child); managed {
				continue
			}
		}
		kept = append(kept, child)
	}
	for _, m := range want {
		kept = append(kept, newAlarm(m, title))
	}
	master.Children = kept
	return nil
}

func sameSet(a, b []int) bool {
	set := func(xs []int) map[int]bool {
		m := map[int]bool{}
		for _, x := range xs {
			m[x] = true
		}
		return m
	}
	sa, sb := set(a), set(b)
	if len(sa) != len(sb) {
		return false
	}
	for k := range sa {
		if !sb[k] {
			return false
		}
	}
	return true
}

package types

import "time"

type Calendar struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
	CTag  string `json:"ctag"`
}

type Event struct {
	UID         string      `json:"uid"`
	CalendarID  string      `json:"calendarId"`
	Href        string      `json:"href,omitempty"` // resource path on the CalDAV server
	ETag        string      `json:"etag"`
	Title       string      `json:"title"`
	Start       time.Time   `json:"start"`
	End         time.Time   `json:"end"`
	AllDay      bool        `json:"allDay"`
	Description string      `json:"description"`
	Location    string      `json:"location"`
	Recurrence  *Recurrence `json:"recurrence,omitempty"`
	// Alarms are notifications, as minutes before the start. Always present on
	// reads. On writes, nil means "leave the stored alarms alone" and an empty
	// list removes them.
	Alarms []int `json:"alarms"`
	// RecurrenceID is set on an occurrence that was modified on its own (an
	// override VEVENT). It identifies the original slot it replaced.
	RecurrenceID string `json:"recurrenceId,omitempty"`
	// OriginalStart is, on update requests only, the start of the occurrence the
	// user opened. It lets the server tell a time change from moving to another day.
	OriginalStart *time.Time `json:"originalStart,omitempty"`
}

type Recurrence struct {
	Frequency  string        `json:"frequency"`
	Interval   int           `json:"interval"`
	ByDay      []string      `json:"byDay,omitempty"`
	ByMonthDay []int         `json:"byMonthDay,omitempty"`
	End        RecurrenceEnd `json:"end"`
}

type RecurrenceEnd struct {
	Type  string `json:"type"`
	Value string `json:"value,omitempty"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type EditScope string

const (
	EditThis             EditScope = "this"
	EditThisAndFollowing EditScope = "thisAndFollowing"
	EditAll              EditScope = "all"
)

type ConnectRequest struct {
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
	TTL      string `json:"ttl"`
}

type ConnectResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expiresAt"`
}

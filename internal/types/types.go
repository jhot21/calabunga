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
	ETag        string      `json:"etag"`
	Title       string      `json:"title"`
	Start       time.Time   `json:"start"`
	End         time.Time   `json:"end"`
	AllDay      bool        `json:"allDay"`
	Description string      `json:"description"`
	Location    string      `json:"location"`
	Recurrence  *Recurrence `json:"recurrence,omitempty"`
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

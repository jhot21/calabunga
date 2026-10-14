package api_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"calabunga/internal/api"
	"calabunga/internal/auth"
	"calabunga/internal/config"
	"calabunga/internal/types"
)

const sampleCalendarResponse = `<?xml version="1.0"?>
<multistatus xmlns="DAV:" xmlns:cal="urn:ietf:params:xml:ns:caldav">
  <response>
    <href>/dav/alice/personal/event1.ics</href>
    <propstat>
      <prop>
        <getetag>"etag-001"</getetag>
        <cal:calendar-data>BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:event1@example.com
SUMMARY:Team standup
DTSTART:20260603T100000Z
DTEND:20260603T103000Z
END:VEVENT
END:VCALENDAR</cal:calendar-data>
      </prop>
      <status>HTTP/1.1 200 OK</status>
    </propstat>
  </response>
</multistatus>`

func TestGetEvents(t *testing.T) {
	caldavSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			w.WriteHeader(200)
			return
		}
		// The REPORT body must use <d:prop> in DAV: namespace so servers like Baikal/SabreDAV
		// return calendar-data. A <prop> in the caldav namespace is silently ignored.
		if r.Method == "REPORT" {
			body, _ := io.ReadAll(r.Body)
			assert.Contains(t, string(body), `<d:prop>`, "REPORT query must use DAV:-namespaced <d:prop>")
		}
		w.WriteHeader(207)
		w.Write([]byte(sampleCalendarResponse))
	}))
	defer caldavSrv.Close()

	cfg := config.Config{JWTSecret: "testsecret12345678901234567890ab", JWTMaxTTL: 720 * time.Hour}
	creds := auth.Credentials{ServerURL: caldavSrv.URL, Username: "alice", Password: "pass", Quirks: "noop"}
	token, _ := auth.Issue(cfg.JWTSecret, creds, 24*time.Hour, cfg.JWTMaxTTL)

	handler := api.NewEventsHandler(cfg)
	req := httptest.NewRequest("GET", "/api/events?start=2026-06-01T00:00:00Z&end=2026-06-30T00:00:00Z&calendarId=/dav/alice/personal/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	api.JWTMiddleware(cfg)(handler).ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var events []types.Event
	require.NoError(t, json.NewDecoder(w.Body).Decode(&events))
	require.Len(t, events, 1)
	assert.Equal(t, "event1@example.com", events[0].UID)
	assert.Equal(t, "Team standup", events[0].Title)
	assert.Equal(t, `"etag-001"`, events[0].ETag)
}

func TestCreateEvent(t *testing.T) {
	caldavSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "OPTIONS":
			w.WriteHeader(200)
		case "PUT":
			w.Header().Set("ETag", `"new-etag-001"`)
			w.WriteHeader(201)
		}
	}))
	defer caldavSrv.Close()

	cfg := config.Config{JWTSecret: "testsecret12345678901234567890ab", JWTMaxTTL: 720 * time.Hour}
	creds := auth.Credentials{ServerURL: caldavSrv.URL, Username: "alice", Password: "pass", Quirks: "noop"}
	token, _ := auth.Issue(cfg.JWTSecret, creds, 24*time.Hour, cfg.JWTMaxTTL)

	event := types.Event{
		CalendarID: caldavSrv.URL + "/dav/alice/personal/",
		Title:      "New meeting",
		Start:      time.Date(2026, 6, 10, 14, 0, 0, 0, time.UTC),
		End:        time.Date(2026, 6, 10, 15, 0, 0, 0, time.UTC),
	}
	body, _ := json.Marshal(event)
	req := httptest.NewRequest("POST", "/api/events", strings.NewReader(string(body)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	api.JWTMiddleware(cfg)(api.NewEventsHandler(cfg)).ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	var created types.Event
	require.NoError(t, json.NewDecoder(w.Body).Decode(&created))
	assert.Equal(t, `"new-etag-001"`, created.ETag)
}

func TestDeleteEvent(t *testing.T) {
	caldavSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "OPTIONS":
			w.WriteHeader(200)
		case "DELETE":
			assert.Equal(t, `"old-etag"`, r.Header.Get("If-Match"))
			w.WriteHeader(204)
		}
	}))
	defer caldavSrv.Close()

	cfg := config.Config{JWTSecret: "testsecret12345678901234567890ab", JWTMaxTTL: 720 * time.Hour}
	creds := auth.Credentials{ServerURL: caldavSrv.URL, Username: "alice", Password: "pass", Quirks: "noop"}
	token, _ := auth.Issue(cfg.JWTSecret, creds, 24*time.Hour, cfg.JWTMaxTTL)

	req := httptest.NewRequest("DELETE", "/api/events/event1@example.com?calendarId="+caldavSrv.URL+"/dav/alice/personal/&editScope=all&etag=%22old-etag%22", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	api.JWTMiddleware(cfg)(api.NewEventsHandler(cfg)).ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

// Deleting one occurrence is not supported yet. Honoring it by deleting the
// resource would silently remove the whole series, so it must be refused.
func TestDeleteEvent_RefusesPartialScopes(t *testing.T) {
	for _, scope := range []string{"this", "thisAndFollowing"} {
		t.Run(scope, func(t *testing.T) {
			deleted := false
			caldavSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.Method {
				case "OPTIONS":
					w.WriteHeader(200)
				case "DELETE":
					deleted = true
					w.WriteHeader(204)
				}
			}))
			defer caldavSrv.Close()

			cfg := config.Config{JWTSecret: "testsecret12345678901234567890ab", JWTMaxTTL: 720 * time.Hour}
			creds := auth.Credentials{ServerURL: caldavSrv.URL, Username: "alice", Password: "pass", Quirks: "noop"}
			token, _ := auth.Issue(cfg.JWTSecret, creds, 24*time.Hour, cfg.JWTMaxTTL)

			req := httptest.NewRequest("DELETE", "/api/events/event1@example.com?calendarId="+caldavSrv.URL+"/dav/alice/personal/&editScope="+scope+"&etag=%22old-etag%22", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()

			api.JWTMiddleware(cfg)(api.NewEventsHandler(cfg)).ServeHTTP(w, req)

			assert.Equal(t, http.StatusNotImplemented, w.Code)
			assert.False(t, deleted, "no DELETE may reach the CalDAV server")
		})
	}
}


const updateFixture = `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:event1@example.com
SUMMARY:Team standup
DTSTART:20260603T100000Z
DTEND:20260603T103000Z
BEGIN:VALARM
ACTION:DISPLAY
DESCRIPTION:Reminder
TRIGGER:-PT15M
END:VALARM
END:VEVENT
END:VCALENDAR`

type updateServer struct {
	fixture  string // calendar object served on GET; defaults to updateFixture
	srv      *httptest.Server
	gotPath  string
	gotMatch string
	gotBody  string
	putCount int
}

// newUpdateServer serves updateFixture at resourcePath with the given ETag and records the PUT.
func newUpdateServer(t *testing.T, resourcePath, etag string) *updateServer {
	t.Helper()
	us := &updateServer{fixture: updateFixture}
	us.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "OPTIONS":
			w.WriteHeader(200)
		case "GET":
			if r.URL.Path != resourcePath {
				w.WriteHeader(404)
				return
			}
			w.Header().Set("ETag", etag)
			w.Header().Set("Content-Type", "text/calendar")
			w.Write([]byte(us.fixture))
		case "PUT":
			b, _ := io.ReadAll(r.Body)
			us.putCount++
			us.gotPath, us.gotMatch, us.gotBody = r.URL.Path, r.Header.Get("If-Match"), string(b)
			w.Header().Set("ETag", `"etag-new"`)
			w.WriteHeader(204)
		}
	}))
	t.Cleanup(us.srv.Close)
	return us
}

func doUpdate(t *testing.T, us *updateServer, ev types.Event) *httptest.ResponseRecorder {
	t.Helper()
	cfg := config.Config{JWTSecret: "testsecret12345678901234567890ab", JWTMaxTTL: 720 * time.Hour}
	creds := auth.Credentials{ServerURL: us.srv.URL, Username: "alice", Password: "pass", Quirks: "noop"}
	token, _ := auth.Issue(cfg.JWTSecret, creds, 24*time.Hour, cfg.JWTMaxTTL)

	ev.CalendarID = us.srv.URL + "/dav/alice/personal/"
	body, _ := json.Marshal(ev)
	req := httptest.NewRequest("PUT", "/api/events/"+ev.UID+"?editScope=all&calendarId="+ev.CalendarID, strings.NewReader(string(body)))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	api.JWTMiddleware(cfg)(api.NewEventsHandler(cfg)).ServeHTTP(w, req)
	return w
}

func updateEvent() types.Event {
	return types.Event{
		UID: "event1@example.com", ETag: `"etag-1"`, Title: "Team standup (renamed)",
		Start: time.Date(2026, 6, 3, 10, 0, 0, 0, time.UTC), End: time.Date(2026, 6, 3, 10, 30, 0, 0, time.UTC),
	}
}

func TestUpdateEvent_MergesIntoExistingResourceAtItsHref(t *testing.T) {
	// The resource name is not UID.ics, so the path must come from the REPORT href.
	us := newUpdateServer(t, "/dav/alice/personal/custom-name.ics", `"etag-1"`)
	ev := updateEvent()
	ev.Href = "/dav/alice/personal/custom-name.ics"

	w := doUpdate(t, us, ev)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "/dav/alice/personal/custom-name.ics", us.gotPath)
	assert.Equal(t, `"etag-1"`, us.gotMatch)
	assert.Contains(t, us.gotBody, "SUMMARY:Team standup (renamed)")
	assert.Contains(t, us.gotBody, "TRIGGER:-PT15M", "existing alarm must survive the edit")
}

func TestUpdateEvent_FallsBackToUIDPathWithoutHref(t *testing.T) {
	us := newUpdateServer(t, "/dav/alice/personal/event1@example.com.ics", `"etag-1"`)

	w := doUpdate(t, us, updateEvent())

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "/dav/alice/personal/event1@example.com.ics", us.gotPath)
}

func TestUpdateEvent_StaleETagIsAConflictAndWritesNothing(t *testing.T) {
	us := newUpdateServer(t, "/dav/alice/personal/event1@example.com.ics", `"etag-2"`)

	w := doUpdate(t, us, updateEvent()) // client still holds "etag-1"

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Equal(t, 0, us.putCount)
}

func TestUpdateEvent_MissingResourceIsAConflict(t *testing.T) {
	us := newUpdateServer(t, "/dav/alice/personal/somewhere-else.ics", `"etag-1"`)

	w := doUpdate(t, us, updateEvent())

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Equal(t, 0, us.putCount)
}

func TestUpdateEvent_RejectsHrefOutsideTheCalendar(t *testing.T) {
	us := newUpdateServer(t, "/dav/alice/other/event1.ics", `"etag-1"`)
	ev := updateEvent()
	ev.Href = "/dav/alice/other/event1.ics"

	w := doUpdate(t, us, ev)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, 0, us.putCount)
}

func TestGetEvents_ReturnsResourceHref(t *testing.T) {
	caldavSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			w.WriteHeader(200)
			return
		}
		w.WriteHeader(207)
		w.Write([]byte(sampleCalendarResponse))
	}))
	defer caldavSrv.Close()

	cfg := config.Config{JWTSecret: "testsecret12345678901234567890ab", JWTMaxTTL: 720 * time.Hour}
	creds := auth.Credentials{ServerURL: caldavSrv.URL, Username: "alice", Password: "pass", Quirks: "noop"}
	token, _ := auth.Issue(cfg.JWTSecret, creds, 24*time.Hour, cfg.JWTMaxTTL)

	req := httptest.NewRequest("GET", "/api/events?calendarId=/dav/alice/personal/&start=2026-06-01T00:00:00Z&end=2026-07-01T00:00:00Z", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	api.JWTMiddleware(cfg)(api.NewEventsHandler(cfg)).ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var events []types.Event
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &events))
	require.Len(t, events, 1)
	assert.Equal(t, "/dav/alice/personal/event1.ics", events[0].Href)
}

const seriesUpdateFixture = `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:event1@example.com
SUMMARY:Team standup
DTSTART:20260602T100000Z
DTEND:20260602T103000Z
RRULE:FREQ=WEEKLY;BYDAY=TU
END:VEVENT
END:VCALENDAR`

func seriesEdit(us *updateServer, start time.Time, original time.Time) types.Event {
	us.fixture = seriesUpdateFixture
	ev := updateEvent()
	ev.Start, ev.End = start, start.Add(30*time.Minute)
	ev.OriginalStart = &original
	ev.Recurrence = &types.Recurrence{Frequency: "weekly", Interval: 1, ByDay: []string{"TU"}, End: types.RecurrenceEnd{Type: "never"}}
	return ev
}

func TestUpdateEvent_SeriesTimeChangeKeepsTheRule(t *testing.T) {
	us := newUpdateServer(t, "/dav/alice/personal/event1@example.com.ics", `"etag-1"`)
	opened := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)

	w := doUpdate(t, us, seriesEdit(us, opened.Add(time.Hour), opened))

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, us.gotBody, "RRULE:FREQ=WEEKLY;BYDAY=TU")
	assert.Contains(t, us.gotBody, "DTSTART:20260602T110000Z")
	assert.NotContains(t, w.Body.String(), "originalStart", "the request-only field must not be echoed back")
}

func TestUpdateEvent_SeriesMovedToAnotherDayIsNotImplemented(t *testing.T) {
	us := newUpdateServer(t, "/dav/alice/personal/event1@example.com.ics", `"etag-1"`)
	opened := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)

	w := doUpdate(t, us, seriesEdit(us, opened.Add(24*time.Hour), opened))

	assert.Equal(t, http.StatusNotImplemented, w.Code)
	assert.Equal(t, 0, us.putCount)
}

func TestUpdateEvent_InvalidRecurrenceIsABadRequest(t *testing.T) {
	us := newUpdateServer(t, "/dav/alice/personal/event1@example.com.ics", `"etag-1"`)
	ev := updateEvent()
	ev.Recurrence = &types.Recurrence{Frequency: "hourly", Interval: 1}

	w := doUpdate(t, us, ev)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, 0, us.putCount)
}

func TestCreateEvent_WritesTheRecurrenceRule(t *testing.T) {
	var putBody string
	caldavSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "OPTIONS":
			w.WriteHeader(200)
		case "PUT":
			b, _ := io.ReadAll(r.Body)
			putBody = string(b)
			w.WriteHeader(201)
		}
	}))
	defer caldavSrv.Close()

	cfg := config.Config{JWTSecret: "testsecret12345678901234567890ab", JWTMaxTTL: 720 * time.Hour}
	creds := auth.Credentials{ServerURL: caldavSrv.URL, Username: "alice", Password: "pass", Quirks: "noop"}
	token, _ := auth.Issue(cfg.JWTSecret, creds, 24*time.Hour, cfg.JWTMaxTTL)

	event := types.Event{
		CalendarID: caldavSrv.URL + "/dav/alice/personal/",
		Title:      "Standup",
		Start:      time.Date(2026, 6, 10, 14, 0, 0, 0, time.UTC),
		End:        time.Date(2026, 6, 10, 15, 0, 0, 0, time.UTC),
		Recurrence: &types.Recurrence{Frequency: "weekly", Interval: 1, ByDay: []string{"WE"}, End: types.RecurrenceEnd{Type: "never"}},
	}
	body, _ := json.Marshal(event)
	req := httptest.NewRequest("POST", "/api/events", strings.NewReader(string(body)))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	api.JWTMiddleware(cfg)(api.NewEventsHandler(cfg)).ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	assert.Contains(t, putBody, "RRULE:FREQ=WEEKLY;BYDAY=WE")
}

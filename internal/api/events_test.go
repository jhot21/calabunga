package api_test

import (
	"encoding/json"
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

	req := httptest.NewRequest("DELETE", "/api/events/event1@example.com?calendarId="+caldavSrv.URL+"/dav/alice/personal/&editScope=this&etag=%22old-etag%22", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	api.JWTMiddleware(cfg)(api.NewEventsHandler(cfg)).ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

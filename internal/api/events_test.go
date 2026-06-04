package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

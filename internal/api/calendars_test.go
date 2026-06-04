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

func TestGetCalendars(t *testing.T) {
	caldavSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/.well-known/caldav":
			w.WriteHeader(200)
		case r.Method == "PROPFIND":
			body, _ := io.ReadAll(r.Body)
			bodyStr := string(body)
			switch {
			case strings.Contains(bodyStr, "current-user-principal"):
				w.WriteHeader(207)
				w.Write([]byte(`<?xml version="1.0"?>
<multistatus xmlns="DAV:">
  <response>
    <href>/dav/alice/</href>
    <propstat>
      <prop>
        <current-user-principal><href>/dav/alice/</href></current-user-principal>
      </prop>
      <status>HTTP/1.1 200 OK</status>
    </propstat>
  </response>
</multistatus>`))
			case strings.Contains(bodyStr, "calendar-home-set"):
				w.WriteHeader(207)
				w.Write([]byte(`<?xml version="1.0"?>
<multistatus xmlns="DAV:" xmlns:cal="urn:ietf:params:xml:ns:caldav">
  <response>
    <href>/dav/alice/</href>
    <propstat>
      <prop>
        <cal:calendar-home-set><href>/dav/alice/</href></cal:calendar-home-set>
      </prop>
      <status>HTTP/1.1 200 OK</status>
    </propstat>
  </response>
</multistatus>`))
			default:
				w.WriteHeader(207)
				w.Write([]byte(`<?xml version="1.0"?>
<multistatus xmlns="DAV:" xmlns:cal="urn:ietf:params:xml:ns:caldav" xmlns:cs="http://calendarserver.org/ns/">
  <response>
    <href>/dav/alice/personal/</href>
    <propstat>
      <prop>
        <displayname>Personal</displayname>
        <cs:getctag>abc123</cs:getctag>
        <cal:calendar-color>#4285F4</cal:calendar-color>
      </prop>
      <status>HTTP/1.1 200 OK</status>
    </propstat>
  </response>
</multistatus>`))
			}
		default:
			w.WriteHeader(200)
		}
	}))
	defer caldavSrv.Close()

	cfg := config.Config{
		JWTSecret: "testsecret12345678901234567890ab",
		JWTMaxTTL: 720 * time.Hour,
	}
	creds := auth.Credentials{ServerURL: caldavSrv.URL, Username: "alice", Password: "pass", Quirks: "noop"}
	token, _ := auth.Issue(cfg.JWTSecret, creds, 24*time.Hour, cfg.JWTMaxTTL)

	handler := api.NewCalendarsHandler(cfg)
	req := httptest.NewRequest("GET", "/api/calendars", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	api.JWTMiddleware(cfg)(handler).ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var cals []types.Calendar
	require.NoError(t, json.NewDecoder(w.Body).Decode(&cals))
	require.Len(t, cals, 1)
	assert.Equal(t, "Personal", cals[0].Name)
	assert.Equal(t, "abc123", cals[0].CTag)
}

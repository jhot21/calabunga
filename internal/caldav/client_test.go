package caldav_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"calabunga/internal/caldav"
	"calabunga/internal/quirks"
)

func TestPropfind(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PROPFIND", r.Method)
		assert.Equal(t, "1", r.Header.Get("Depth"))
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.WriteHeader(207)
		w.Write([]byte(`<?xml version="1.0"?>
<multistatus xmlns="DAV:">
  <response>
    <href>/dav/alice/calendar/</href>
    <propstat>
      <prop><displayname>Personal</displayname></prop>
      <status>HTTP/1.1 200 OK</status>
    </propstat>
  </response>
</multistatus>`))
	}))
	defer srv.Close()

	client := caldav.NewClient(srv.URL, "alice", "pass", quirks.Noop{})
	body, err := client.Propfind("/dav/alice/calendar/", 1, `<prop><displayname/></prop>`)
	require.NoError(t, err)
	assert.Contains(t, string(body), "Personal")
}

func TestReport(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "REPORT", r.Method)
		w.WriteHeader(207)
		w.Write([]byte(`<multistatus xmlns="DAV:"></multistatus>`))
	}))
	defer srv.Close()

	client := caldav.NewClient(srv.URL, "alice", "pass", quirks.Noop{})
	body, err := client.Report("/dav/alice/calendar/", `<calendar-query xmlns="urn:ietf:params:xml:ns:caldav"/>`)
	require.NoError(t, err)
	assert.NotEmpty(t, body)
}

func TestPut(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		assert.Equal(t, `"old-etag"`, r.Header.Get("If-Match"))
		w.Header().Set("ETag", `"new-etag"`)
		w.WriteHeader(204)
	}))
	defer srv.Close()

	client := caldav.NewClient(srv.URL, "alice", "pass", quirks.Noop{})
	etag, err := client.Put("/dav/alice/calendar/event.ics", []byte("BEGIN:VCALENDAR"), `"old-etag"`)
	require.NoError(t, err)
	assert.Equal(t, `"new-etag"`, etag)
}

func TestDelete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, `"old-etag"`, r.Header.Get("If-Match"))
		w.WriteHeader(204)
	}))
	defer srv.Close()

	client := caldav.NewClient(srv.URL, "alice", "pass", quirks.Noop{})
	err := client.Delete("/dav/alice/calendar/event.ics", `"old-etag"`)
	require.NoError(t, err)
}

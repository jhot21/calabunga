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

func TestDiscover_WellKnown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/caldav":
			http.Redirect(w, r, "/dav/", http.StatusMovedPermanently)
		case "/dav/":
			w.WriteHeader(207)
			w.Write([]byte(`<?xml version="1.0"?>
<multistatus xmlns="DAV:">
  <response>
    <href>/dav/</href>
    <propstat>
      <prop><current-user-principal><href>/dav/alice/</href></current-user-principal></prop>
      <status>HTTP/1.1 200 OK</status>
    </propstat>
  </response>
</multistatus>`))
		case "/dav/alice/":
			w.WriteHeader(207)
			w.Write([]byte(`<?xml version="1.0"?>
<multistatus xmlns="DAV:">
  <response>
    <href>/dav/alice/</href>
    <propstat>
      <prop><calendar-home-set xmlns="urn:ietf:params:xml:ns:caldav"><href>/dav/alice/calendars/</href></calendar-home-set></prop>
      <status>HTTP/1.1 200 OK</status>
    </propstat>
  </response>
</multistatus>`))
		}
	}))
	defer srv.Close()

	client := caldav.NewClient(srv.URL, "alice", "pass", quirks.Noop{})
	home, err := caldav.Discover(client, srv.URL)
	require.NoError(t, err)
	assert.Equal(t, "/dav/alice/calendars/", home)
}

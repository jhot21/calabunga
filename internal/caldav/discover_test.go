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

// TestDiscover_NamespacedHref verifies discovery works when servers return
// namespace-prefixed hrefs like <d:href> (as SabreDAV/Baikal does).
func TestDiscover_NamespacedHref(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		switch r.URL.Path {
		case "/.well-known/caldav":
			http.Redirect(w, r, "/dav.php/", http.StatusMovedPermanently)
		case "/dav.php/":
			w.WriteHeader(207)
			w.Write([]byte(`<?xml version="1.0"?>` +
				`<d:multistatus xmlns:d="DAV:" xmlns:cal="urn:ietf:params:xml:ns:caldav">` +
				`<d:response><d:href>/dav.php/</d:href>` +
				`<d:propstat><d:prop><d:current-user-principal><d:href>/dav.php/principals/alice/</d:href></d:current-user-principal></d:prop>` +
				`<d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response></d:multistatus>`))
		case "/dav.php/principals/alice/":
			w.WriteHeader(207)
			w.Write([]byte(`<?xml version="1.0"?>` +
				`<d:multistatus xmlns:d="DAV:" xmlns:cal="urn:ietf:params:xml:ns:caldav">` +
				`<d:response><d:href>/dav.php/principals/alice/</d:href>` +
				`<d:propstat><d:prop><cal:calendar-home-set><d:href>/dav.php/calendars/alice/</d:href></cal:calendar-home-set></d:prop>` +
				`<d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response></d:multistatus>`))
		}
	}))
	defer srv.Close()

	client := caldav.NewClient(srv.URL, "alice", "pass", quirks.Noop{})
	home, err := caldav.Discover(client, srv.URL)
	require.NoError(t, err)
	assert.Equal(t, "/dav.php/calendars/alice/", home)
}

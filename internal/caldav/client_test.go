package caldav_test

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestDigestAuth(t *testing.T) {
	const realm = "TestRealm"
	const nonce = "testnonce123"
	const opaque = "testopaque456"

	digestResponse := func(username, password, method, uri string) string {
		ha1 := fmt.Sprintf("%x", md5.Sum([]byte(username+":"+realm+":"+password)))
		ha2 := fmt.Sprintf("%x", md5.Sum([]byte(method+":"+uri)))
		// The client sends nc=00000001 and a random cnonce; we extract them from the header.
		return ha1 + ":" + nonce + ":" + ha2 // used only for structure verification
	}
	_ = digestResponse

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")

		if !strings.HasPrefix(auth, "Digest ") {
			w.Header().Set("WWW-Authenticate", fmt.Sprintf(
				`Digest realm="%s",qop="auth",nonce="%s",opaque="%s"`, realm, nonce, opaque))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Parse the Digest response and verify it's correct.
		params := make(map[string]string)
		for _, part := range strings.Split(strings.TrimPrefix(auth, "Digest "), ",") {
			kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
			if len(kv) == 2 {
				params[strings.TrimSpace(kv[0])] = strings.Trim(strings.TrimSpace(kv[1]), `"`)
			}
		}

		ha1 := fmt.Sprintf("%x", md5.Sum([]byte(params["username"]+":"+realm+":alice-password")))
		ha2 := fmt.Sprintf("%x", md5.Sum([]byte(r.Method+":"+params["uri"])))
		expected := fmt.Sprintf("%x", md5.Sum([]byte(
			ha1+":"+nonce+":"+params["nc"]+":"+params["cnonce"]+":auth:"+ha2)))

		if params["response"] != expected {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(207)
		w.Write([]byte(`<multistatus xmlns="DAV:"></multistatus>`))
	}))
	defer srv.Close()

	client := caldav.NewClient(srv.URL, "alice", "alice-password", quirks.Noop{})
	body, err := client.Propfind("/dav/", 0, `<prop><displayname/></prop>`)
	require.NoError(t, err)
	assert.NotEmpty(t, body)
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

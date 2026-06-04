package quirks_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"calabunga/internal/quirks"
)

func TestDetect_Baikal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", "Apache/2.4 (baikal)")
		w.Header().Set("DAV", "1, 3, extended-mkcol, access-control, calendarserver-principal-property-search, calendar-access")
		w.WriteHeader(200)
	}))
	defer srv.Close()

	q := quirks.Detect(srv.URL, "alice", "pass")
	assert.Equal(t, "baikal", q.Name())
}

func TestDetect_Unknown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", "SomeOtherServer/1.0")
		w.WriteHeader(200)
	}))
	defer srv.Close()

	q := quirks.Detect(srv.URL, "alice", "pass")
	assert.Equal(t, "noop", q.Name())
}

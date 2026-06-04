package quirks_test

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"calabunga/internal/quirks"
)

func TestBaikal_ModifyRequest(t *testing.T) {
	b := quirks.Baikal{}
	req := httptest.NewRequest("PUT", "/dav/calendar/event.ics", nil)
	b.ModifyRequest(req)
	assert.Equal(t, "text/calendar", req.Header.Get("Content-Type"))
}

func TestBaikal_SupportsPush(t *testing.T) {
	assert.False(t, quirks.Baikal{}.SupportsPush())
}

package quirks

import "net/http"

type Baikal struct{}

func (Baikal) Name() string { return "baikal" }

func (Baikal) ModifyRequest(r *http.Request) {
	if r.Method == "PUT" {
		r.Header.Set("Content-Type", "text/calendar")
	}
}

func (Baikal) ModifyResponse(b []byte) []byte { return b }
func (Baikal) SupportsPush() bool             { return false }

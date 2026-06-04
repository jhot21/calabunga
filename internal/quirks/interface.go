package quirks

import "net/http"

type Quirks interface {
	Name() string
	ModifyRequest(r *http.Request)
	ModifyResponse(body []byte) []byte
	SupportsPush() bool
}

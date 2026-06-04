package quirks

import "net/http"

type Noop struct{}

func (Noop) Name() string                   { return "noop" }
func (Noop) ModifyRequest(_ *http.Request)  {}
func (Noop) ModifyResponse(b []byte) []byte { return b }
func (Noop) SupportsPush() bool             { return false }

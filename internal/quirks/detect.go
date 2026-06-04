package quirks

import (
	"net/http"
	"strings"
)

func Detect(serverURL, username, password string) Quirks {
	req, err := http.NewRequest("OPTIONS", serverURL, nil)
	if err != nil {
		return Noop{}
	}
	req.SetBasicAuth(username, password)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Noop{}
	}
	defer resp.Body.Close()

	server := strings.ToLower(resp.Header.Get("Server"))
	dav := strings.ToLower(resp.Header.Get("DAV"))

	if isBaikal(server, dav) {
		return Baikal{}
	}
	return Noop{}
}

func isBaikal(server, dav string) bool {
	return strings.Contains(server, "baikal") ||
		(strings.Contains(dav, "calendar-access") &&
			strings.Contains(dav, "extended-mkcol"))
}

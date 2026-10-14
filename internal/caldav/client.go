package caldav

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"calabunga/internal/quirks"
)

type Client struct {
	baseURL  string
	username string
	password string
	quirks   quirks.Quirks
	http     *http.Client
}

func NewClient(baseURL, username, password string, q quirks.Quirks) *Client {
	return &Client{
		baseURL:  strings.TrimRight(baseURL, "/"),
		username: username,
		password: password,
		quirks:   q,
		http:     &http.Client{Transport: &digestTransport{username: username, password: password}},
	}
}

func (c *Client) Propfind(path string, depth int, xmlBody string) ([]byte, error) {
	body := `<?xml version="1.0" encoding="UTF-8"?><propfind xmlns="DAV:">` + xmlBody + `</propfind>`
	req, err := c.newRequest("PROPFIND", path, []byte(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Depth", strconv.Itoa(depth))
	req.Header.Set("Content-Type", "application/xml; charset=utf-8")
	return c.do(req, 207)
}

func (c *Client) Report(path, xmlBody string) ([]byte, error) {
	req, err := c.newRequest("REPORT", path, []byte(xmlBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Depth", "1")
	req.Header.Set("Content-Type", "application/xml; charset=utf-8")
	return c.do(req, 207)
}

// Get fetches a single calendar object and returns its body and ETag.
func (c *Client) Get(path string) ([]byte, string, error) {
	req, err := c.newRequest("GET", path, nil)
	if err != nil {
		return nil, "", err
	}
	c.quirks.ModifyRequest(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, "", ErrUnauthorized
	case http.StatusNotFound:
		return nil, "", ErrNotFound
	default:
		return nil, "", fmt.Errorf("GET %s: unexpected status %d", path, resp.StatusCode)
	}
	return c.quirks.ModifyResponse(body), resp.Header.Get("ETag"), nil
}

func (c *Client) Put(path string, ical []byte, ifMatch string) (string, error) {
	req, err := c.newRequest("PUT", path, ical)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "text/calendar; charset=utf-8")
	if ifMatch != "" {
		req.Header.Set("If-Match", ifMatch)
	}
	c.quirks.ModifyRequest(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	c.quirks.ModifyResponse(body)
	if resp.StatusCode == http.StatusPreconditionFailed || resp.StatusCode == http.StatusPreconditionRequired {
		return "", ErrConflict
	}
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		return "", fmt.Errorf("PUT %s: unexpected status %d", path, resp.StatusCode)
	}
	return resp.Header.Get("ETag"), nil
}

func (c *Client) Delete(path, ifMatch string) error {
	req, err := c.newRequest("DELETE", path, nil)
	if err != nil {
		return err
	}
	if ifMatch != "" {
		req.Header.Set("If-Match", ifMatch)
	}
	c.quirks.ModifyRequest(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	c.quirks.ModifyResponse(body)
	if resp.StatusCode == http.StatusPreconditionFailed {
		return ErrConflict
	}
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("DELETE %s: unexpected status %d", path, resp.StatusCode)
	}
	return nil
}

var ErrConflict = fmt.Errorf("etag mismatch: event was modified externally")
var ErrUnauthorized = fmt.Errorf("caldav: server rejected credentials")
var ErrNotFound = fmt.Errorf("caldav: server URL not found")

func (c *Client) newRequest(method, path string, body []byte) (*http.Request, error) {
	url := c.baseURL + path
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.username, c.password)
	return req, nil
}

func (c *Client) do(req *http.Request, expectStatus int) ([]byte, error) {
	c.quirks.ModifyRequest(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	body = c.quirks.ModifyResponse(body)
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, ErrUnauthorized
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode != expectStatus {
		return nil, fmt.Errorf("%s %s: expected %d, got %d", req.Method, req.URL.Path, expectStatus, resp.StatusCode)
	}
	return body, nil
}

// digestTransport transparently handles Digest auth negotiation.
// It sends the initial request with Basic auth; if the server returns 401
// with a Digest challenge, it retries with the correct Digest response.
type digestTransport struct {
	username string
	password string
}

func (t *digestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Buffer the body so we can replay it if a Digest retry is needed.
	var bodyBytes []byte
	if req.Body != nil && req.Body != http.NoBody {
		var err error
		if bodyBytes, err = io.ReadAll(req.Body); err != nil {
			return nil, err
		}
		req.Body.Close()
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil || resp.StatusCode != http.StatusUnauthorized {
		return resp, err
	}

	wwwAuth := resp.Header.Get("WWW-Authenticate")
	resp.Body.Close()

	if !strings.HasPrefix(strings.ToLower(wwwAuth), "digest") {
		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Header:     resp.Header,
			Body:       http.NoBody,
		}, nil
	}

	req2 := req.Clone(req.Context())
	req2.Header.Del("Authorization")
	if len(bodyBytes) > 0 {
		req2.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		req2.ContentLength = int64(len(bodyBytes))
	}
	req2.Header.Set("Authorization", buildDigestAuth(t.username, t.password, req.Method, req.URL.RequestURI(), wwwAuth))
	return http.DefaultTransport.RoundTrip(req2)
}

func buildDigestAuth(username, password, method, uri, challenge string) string {
	idx := strings.IndexByte(challenge, ' ')
	if idx < 0 {
		return ""
	}
	params := parseDigestParams(challenge[idx+1:])

	realm := params["realm"]
	nonce := params["nonce"]
	opaque := params["opaque"]

	qop := ""
	for _, q := range strings.Split(params["qop"], ",") {
		if strings.TrimSpace(q) == "auth" {
			qop = "auth"
			break
		}
	}

	ha1 := md5hex(username + ":" + realm + ":" + password)
	ha2 := md5hex(method + ":" + uri)

	nc := "00000001"
	cnonce := randHex(8)

	var response string
	if qop == "auth" {
		response = md5hex(ha1 + ":" + nonce + ":" + nc + ":" + cnonce + ":auth:" + ha2)
	} else {
		response = md5hex(ha1 + ":" + nonce + ":" + ha2)
	}

	auth := fmt.Sprintf(`Digest username="%s", realm="%s", nonce="%s", uri="%s", response="%s"`,
		username, realm, nonce, uri, response)
	if qop == "auth" {
		auth += fmt.Sprintf(`, qop=auth, nc=%s, cnonce="%s"`, nc, cnonce)
	}
	if opaque != "" {
		auth += fmt.Sprintf(`, opaque="%s"`, opaque)
	}
	return auth
}

func parseDigestParams(s string) map[string]string {
	params := make(map[string]string)
	for _, part := range strings.Split(s, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) == 2 {
			params[strings.TrimSpace(kv[0])] = strings.Trim(strings.TrimSpace(kv[1]), `"`)
		}
	}
	return params
}

func md5hex(s string) string {
	h := md5.Sum([]byte(s))
	return fmt.Sprintf("%x", h)
}

func randHex(n int) string {
	b := make([]byte, n)
	rand.Read(b) //nolint:errcheck
	return fmt.Sprintf("%x", b)
}

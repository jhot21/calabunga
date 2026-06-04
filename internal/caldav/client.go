package caldav

import (
	"bytes"
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
		http:     &http.Client{},
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
	if resp.StatusCode == http.StatusPreconditionFailed {
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
	if resp.StatusCode != expectStatus {
		return nil, fmt.Errorf("%s %s: expected %d, got %d", req.Method, req.URL.Path, expectStatus, resp.StatusCode)
	}
	return body, nil
}

package caldav

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

func Discover(client *Client, serverURL string) (string, error) {
	wellKnown := strings.TrimRight(serverURL, "/") + "/.well-known/caldav"
	base, err := followRedirects(wellKnown)
	if err != nil {
		base = serverURL
	}

	principal, err := client.findPrincipal(base)
	if err != nil {
		return "", fmt.Errorf("find principal: %w", err)
	}

	home, err := client.findCalendarHome(principal)
	if err != nil {
		return "", fmt.Errorf("find calendar home: %w", err)
	}
	return home, nil
}

func followRedirects(rawURL string) (string, error) {
	httpClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := httpClient.Get(rawURL)
	if err != nil {
		return "", err
	}
	resp.Body.Close()
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		loc := resp.Header.Get("Location")
		if loc != "" {
			base, err := url.Parse(rawURL)
			if err != nil {
				return loc, nil
			}
			ref, err := url.Parse(loc)
			if err != nil {
				return loc, nil
			}
			return base.ResolveReference(ref).String(), nil
		}
	}
	return rawURL, nil
}

func (c *Client) findPrincipal(base string) (string, error) {
	path := strings.TrimPrefix(base, c.baseURL)
	if path == "" {
		path = "/"
	}
	body, err := c.Propfind(path, 0, `<prop><current-user-principal/></prop>`)
	if err != nil {
		return "", err
	}
	return extractHref(body, "current-user-principal")
}

func (c *Client) findCalendarHome(principalPath string) (string, error) {
	body, err := c.Propfind(principalPath, 0,
		`<prop><calendar-home-set xmlns="urn:ietf:params:xml:ns:caldav"/></prop>`)
	if err != nil {
		return "", err
	}
	return extractHref(body, "calendar-home-set")
}

func extractHref(xmlBody []byte, element string) (string, error) {
	s := string(xmlBody)
	idx := findElement(s, element)
	if idx < 0 {
		return "", fmt.Errorf("element %q not found in response", element)
	}
	sub := s[idx:]
	tagEnd := strings.Index(sub, ">")
	if tagEnd < 0 {
		return "", fmt.Errorf("malformed element %q", element)
	}
	sub = sub[tagEnd+1:]
	hStart := strings.Index(sub, "<href>")
	if hStart < 0 {
		return "", fmt.Errorf("no href in element %q", element)
	}
	sub = sub[hStart+6:]
	hEnd := strings.Index(sub, "</href>")
	if hEnd < 0 {
		return "", fmt.Errorf("no closing href in element %q", element)
	}
	return strings.TrimSpace(sub[:hEnd]), nil
}

func findElement(s, element string) int {
	patterns := []string{
		"<" + element + ">",
		"<" + element + " ",
		":" + element + ">",
		":" + element + " ",
	}
	best := -1
	for _, p := range patterns {
		if idx := strings.Index(s, p); idx >= 0 {
			if best < 0 || idx < best {
				best = idx
			}
		}
	}
	return best
}

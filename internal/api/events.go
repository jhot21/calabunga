package api

import (
	"crypto/rand"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"calabunga/internal/auth"
	"calabunga/internal/caldav"
	"calabunga/internal/config"
	"calabunga/internal/types"
)

func NewEventsHandler(cfg config.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGetEvents(w, r, cfg)
		case http.MethodPost:
			handleCreateEvent(w, r, cfg)
		case http.MethodPut:
			handleUpdateEvent(w, r, cfg)
		case http.MethodDelete:
			handleDeleteEvent(w, r, cfg)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		}
	})
}

func handleGetEvents(w http.ResponseWriter, r *http.Request, cfg config.Config) {
	claims := claimsFromCtx(r)
	creds, err := auth.Decrypt(cfg.JWTSecret, claims)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "token_expired", "could not decrypt credentials")
		return
	}

	q := resolveQuirks(creds.Quirks)
	client := caldav.NewClient(creds.ServerURL, creds.Username, creds.Password, q)

	calendarID := r.URL.Query().Get("calendarId")
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	if calendarID == "" || startStr == "" || endStr == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "calendarId, start, and end are required")
		return
	}

	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid start time")
		return
	}
	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid end time")
		return
	}

	query := fmt.Sprintf(`<calendar-query xmlns="urn:ietf:params:xml:ns:caldav" xmlns:d="DAV:">
  <d:prop>
    <d:getetag/>
    <calendar-data/>
  </d:prop>
  <filter>
    <comp-filter name="VCALENDAR">
      <comp-filter name="VEVENT">
        <time-range start="%s" end="%s"/>
      </comp-filter>
    </comp-filter>
  </filter>
</calendar-query>`,
		start.UTC().Format("20060102T150405Z"),
		end.UTC().Format("20060102T150405Z"))

	body, err := client.Report(calendarID, query)
	if err != nil {
		if errors.Is(err, caldav.ErrUnauthorized) {
			writeError(w, http.StatusUnauthorized, "auth_failed", "CalDAV server rejected credentials")
			return
		}
		writeError(w, http.StatusBadGateway, "caldav_error", "failed to fetch events")
		return
	}

	events, err := parseEventReport(body, calendarID, start, end)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "failed to parse events")
		return
	}

	writeJSON(w, http.StatusOK, events)
}

func parseEventReport(xmlBody []byte, calendarID string, rangeStart, rangeEnd time.Time) ([]types.Event, error) {
	type propstat struct {
		ETag         string `xml:"prop>getetag"`
		CalendarData string `xml:"prop>calendar-data"`
	}
	type response struct {
		Href     string   `xml:"href"`
		Propstat propstat `xml:"propstat"`
	}
	type multistatus struct {
		Responses []response `xml:"response"`
	}

	var ms multistatus
	if err := xml.Unmarshal(xmlBody, &ms); err != nil {
		return nil, err
	}

	var all []types.Event
	for _, r := range ms.Responses {
		if r.Propstat.CalendarData == "" {
			continue
		}
		events, err := caldav.ParseVCalendar([]byte(strings.TrimSpace(r.Propstat.CalendarData)), calendarID, r.Propstat.ETag, rangeStart, rangeEnd)
		if err != nil {
			continue
		}
		all = append(all, events...)
	}
	return all, nil
}

func handleCreateEvent(w http.ResponseWriter, r *http.Request, cfg config.Config) {
	claims := claimsFromCtx(r)
	creds, err := auth.Decrypt(cfg.JWTSecret, claims)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "token_expired", "could not decrypt credentials")
		return
	}

	var event types.Event
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}

	if event.UID == "" {
		event.UID = generateUID()
	}

	ical, err := caldav.EventToVCalendar(event)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "failed to serialize event")
		return
	}

	q := resolveQuirks(creds.Quirks)
	client := caldav.NewClient(creds.ServerURL, creds.Username, creds.Password, q)

	path := extractPath(event.CalendarID) + event.UID + ".ics"

	etag, err := client.Put(path, ical, "")
	if err != nil {
		writeError(w, http.StatusBadGateway, "caldav_error", "failed to create event")
		return
	}
	event.ETag = etag
	writeJSON(w, http.StatusCreated, event)
}

func handleUpdateEvent(w http.ResponseWriter, r *http.Request, cfg config.Config) {
	claims := claimsFromCtx(r)
	creds, err := auth.Decrypt(cfg.JWTSecret, claims)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "token_expired", "could not decrypt credentials")
		return
	}

	var event types.Event
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}

	scope := types.EditScope(r.URL.Query().Get("editScope"))
	if scope == "" {
		scope = types.EditAll
	}

	ical, err := caldav.EventToVCalendar(event)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "failed to serialize event")
		return
	}

	q := resolveQuirks(creds.Quirks)
	client := caldav.NewClient(creds.ServerURL, creds.Username, creds.Password, q)

	calendarID := r.URL.Query().Get("calendarId")
	path := extractPath(calendarID) + event.UID + ".ics"

	switch scope {
	case types.EditAll:
		etag, err := client.Put(path, ical, event.ETag)
		if err == caldav.ErrConflict {
			writeError(w, http.StatusConflict, "conflict", "event was modified externally")
			return
		}
		if err != nil {
			writeError(w, http.StatusBadGateway, "caldav_error", "failed to update event")
			return
		}
		event.ETag = etag
	case types.EditThis, types.EditThisAndFollowing:
		writeError(w, http.StatusNotImplemented, "not_implemented",
			"editing individual occurrences of recurring events is not yet supported")
		return
	}

	writeJSON(w, http.StatusOK, event)
}

func handleDeleteEvent(w http.ResponseWriter, r *http.Request, cfg config.Config) {
	claims := claimsFromCtx(r)
	creds, err := auth.Decrypt(cfg.JWTSecret, claims)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "token_expired", "could not decrypt credentials")
		return
	}

	q := resolveQuirks(creds.Quirks)
	client := caldav.NewClient(creds.ServerURL, creds.Username, creds.Password, q)

	calendarID := r.URL.Query().Get("calendarId")
	uid := chi.URLParam(r, "uid")
	if uid == "" {
		uid = r.URL.Query().Get("uid")
	}
	etag := r.URL.Query().Get("etag")
	path := extractPath(calendarID) + uid + ".ics"

	if err := client.Delete(path, etag); err != nil {
		if err == caldav.ErrConflict {
			writeError(w, http.StatusConflict, "conflict", "event was modified externally")
			return
		}
		writeError(w, http.StatusBadGateway, "caldav_error", "failed to delete event")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func generateUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x@calabunga", b)
}

func extractPath(rawURL string) string {
	if strings.HasPrefix(rawURL, "http") {
		u, err := url.Parse(rawURL)
		if err == nil {
			return u.Path
		}
	}
	return rawURL
}

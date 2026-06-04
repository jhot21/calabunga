package api

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
	"time"

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

	query := fmt.Sprintf(`<calendar-query xmlns="urn:ietf:params:xml:ns:caldav">
  <prop>
    <getetag xmlns="DAV:"/>
    <calendar-data/>
  </prop>
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
		writeError(w, http.StatusBadGateway, "caldav_error", "failed to fetch events")
		return
	}

	events, err := parseEventReport(body, calendarID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "failed to parse events")
		return
	}

	writeJSON(w, http.StatusOK, events)
}

func parseEventReport(xmlBody []byte, calendarID string) ([]types.Event, error) {
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
		events, err := caldav.ParseVCalendar([]byte(strings.TrimSpace(r.Propstat.CalendarData)), calendarID, r.Propstat.ETag)
		if err != nil {
			continue
		}
		all = append(all, events...)
	}
	return all, nil
}

func handleCreateEvent(w http.ResponseWriter, r *http.Request, cfg config.Config) {
	writeError(w, http.StatusNotImplemented, "not_implemented", "not yet implemented")
}

func handleUpdateEvent(w http.ResponseWriter, r *http.Request, cfg config.Config) {
	writeError(w, http.StatusNotImplemented, "not_implemented", "not yet implemented")
}

func handleDeleteEvent(w http.ResponseWriter, r *http.Request, cfg config.Config) {
	writeError(w, http.StatusNotImplemented, "not_implemented", "not yet implemented")
}

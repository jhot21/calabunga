package api

import (
	"encoding/xml"
	"errors"
	"net/http"
	"strings"

	"calabunga/internal/auth"
	"calabunga/internal/caldav"
	"calabunga/internal/config"
	"calabunga/internal/quirks"
	"calabunga/internal/types"
)

func NewCalendarsHandler(cfg config.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := claimsFromCtx(r)
		creds, err := auth.Decrypt(cfg.JWTSecret, claims)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "token_expired", "could not decrypt credentials")
			return
		}

		q := resolveQuirks(creds.Quirks)
		client := caldav.NewClient(creds.ServerURL, creds.Username, creds.Password, q)

		home, err := caldav.Discover(client, creds.ServerURL)
		if err != nil {
			if errors.Is(err, caldav.ErrUnauthorized) {
				writeError(w, http.StatusUnauthorized, "auth_failed", "CalDAV server rejected credentials")
				return
			}
			writeError(w, http.StatusBadGateway, "caldav_error", "could not discover calendar home")
			return
		}

		body, err := client.Propfind(home, 1, `
			<prop>
				<displayname/>
				<cs:getctag xmlns:cs="http://calendarserver.org/ns/"/>
				<cal:calendar-color xmlns:cal="urn:ietf:params:xml:ns:caldav"/>
			</prop>`)
		if err != nil {
			if errors.Is(err, caldav.ErrUnauthorized) {
				writeError(w, http.StatusUnauthorized, "auth_failed", "CalDAV server rejected credentials")
				return
			}
			writeError(w, http.StatusBadGateway, "caldav_error", "could not list calendars")
			return
		}

		cals := parseCalendars(body)
		writeJSON(w, http.StatusOK, cals)
	})
}

func parseCalendars(xmlBody []byte) []types.Calendar {
	type propstat struct {
		DisplayName string `xml:"prop>displayname"`
		CTag        string `xml:"prop>getctag"`
		Color       string `xml:"prop>calendar-color"`
	}
	type response struct {
		Href     string   `xml:"href"`
		Propstat propstat `xml:"propstat"`
	}
	type multistatus struct {
		Responses []response `xml:"response"`
	}

	var ms multistatus
	xml.Unmarshal(xmlBody, &ms)

	var cals []types.Calendar
	for _, r := range ms.Responses {
		if r.Propstat.DisplayName == "" {
			continue
		}
		if !strings.HasSuffix(r.Href, "/") {
			continue
		}
		cals = append(cals, types.Calendar{
			ID:    r.Href,
			Name:  r.Propstat.DisplayName,
			Color: r.Propstat.Color,
			CTag:  r.Propstat.CTag,
		})
	}
	return cals
}

func resolveQuirks(name string) quirks.Quirks {
	switch name {
	case "baikal":
		return quirks.Baikal{}
	default:
		return quirks.Noop{}
	}
}

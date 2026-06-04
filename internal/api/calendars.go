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
				<apple:calendar-color xmlns:apple="http://apple.com/ns/ical/"/>
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

type calendarColor string

func (c *calendarColor) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var value string
	if err := d.DecodeElement(&value, &start); err != nil {
		return err
	}
	*c = calendarColor(value)
	return nil
}

func parseCalendars(xmlBody []byte) []types.Calendar {
	type prop struct {
		DisplayName string          `xml:"displayname"`
		CTag        string          `xml:"getctag"`
		Color       calendarColor   `xml:"calendar-color"`
	}
	type propstat struct {
		Prop   prop   `xml:"prop"`
		Status string `xml:"status"`
	}
	type response struct {
		Href     string     `xml:"href"`
		Propstat []propstat `xml:"propstat"`
	}
	type multistatus struct {
		Responses []response `xml:"response"`
	}

	var ms multistatus
	xml.Unmarshal(xmlBody, &ms)

	var cals []types.Calendar
	for _, r := range ms.Responses {
		if !strings.HasSuffix(r.Href, "/") {
			continue
		}
		var props *prop
		for _, ps := range r.Propstat {
			if strings.Contains(ps.Status, "200") && ps.Prop.DisplayName != "" {
				props = &ps.Prop
				break
			}
		}
		if props == nil {
			continue
		}
		cals = append(cals, types.Calendar{
			ID:    r.Href,
			Name:  props.DisplayName,
			Color: string(props.Color),
			CTag:  props.CTag,
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

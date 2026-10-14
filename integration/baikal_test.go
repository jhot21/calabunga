//go:build integration

package integration_test

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcexec "github.com/testcontainers/testcontainers-go/exec"
	"github.com/testcontainers/testcontainers-go/wait"
	"calabunga/internal/api"
	"calabunga/internal/auth"
	"calabunga/internal/caldav"
	"calabunga/internal/config"
	"calabunga/internal/quirks"
	"calabunga/internal/types"
)

var (
	baikalURL string
	testUser  = "test@example.com"
	testPass  = "testpassword123"
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "ckulka/baikal:latest",
		ExposedPorts: []string{"80/tcp"},
		WaitingFor:   wait.ForHTTP("/").WithStartupTimeout(60 * time.Second),
		Env: map[string]string{
			"BAIKAL_ADMIN_PASSWORD": "adminpass",
		},
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		panic(fmt.Sprintf("start baikal container: %v", err))
	}
	defer container.Terminate(ctx)

	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "80")
	baikalURL = fmt.Sprintf("http://%s:%s", host, port.Port())

	if err := provisionTestUser(ctx, container, testUser, testPass); err != nil {
		panic(fmt.Sprintf("provision test user: %v", err))
	}

	m.Run()
}

// baikalRealm is Baikal's default auth realm; Digest hashes are md5(user:realm:password).
const baikalRealm = "BaikalDAV"

// provisionTestUser configures a fresh Baikal container the way its install
// wizard would, then creates a user with a default calendar. The image ships
// with an empty database and no baikal.yaml, and has no sqlite3 binary, so the
// schema is loaded through PHP's PDO from Baikal's own db.sql.
func provisionTestUser(ctx context.Context, container testcontainers.Container, username, password string) error {
	md5hex := func(s string) string { return fmt.Sprintf("%x", md5.Sum([]byte(s))) }

	config := fmt.Sprintf(`system:
    configured_version: 0.10.1
    timezone: UTC
    card_enabled: true
    cal_enabled: true
    dav_auth_type: Digest
    admin_passwordhash: %s
    failed_access_message: 'user %%u authentication failure for Baikal'
    auth_realm: %s
    base_uri: ''
    invite_from: noreply@localhost
database:
    sqlite_file: /var/www/baikal/Specific/db/db.sqlite
    backend: sqlite
    mysql_host: ''
    mysql_dbname: ''
    mysql_username: ''
    mysql_password: ''
    encryption_key: 00000000000000000000000000000000
    pgsql_host: ''
    pgsql_dbname: ''
    pgsql_username: ''
    pgsql_password: ''
`, md5hex("admin:"+baikalRealm+":adminpass"), baikalRealm)
	if err := container.CopyToContainer(ctx, []byte(config), "/var/www/baikal/config/baikal.yaml", 0o644); err != nil {
		return fmt.Errorf("write baikal.yaml: %w", err)
	}

	// Baikal refuses to serve unless the web user can write its config file.
	exitCode, _, err := container.Exec(ctx, []string{"chown", "www-data:www-data", "/var/www/baikal/config/baikal.yaml"})
	if err != nil || exitCode != 0 {
		return fmt.Errorf("chown baikal.yaml: exitCode=%d err=%v", exitCode, err)
	}

	// Run as www-data so the SQLite file is writable by Apache afterwards.
	const script = `
$db = new PDO("sqlite:/var/www/baikal/Specific/db/db.sqlite");
$db->setAttribute(PDO::ATTR_ERRMODE, PDO::ERRMODE_EXCEPTION);
$db->exec(file_get_contents("/var/www/baikal/Core/Resources/Db/SQLite/db.sql"));
[$user, $digest] = [$argv[1], $argv[2]];
$db->prepare("INSERT INTO users (username, digesta1) VALUES (?, ?)")->execute([$user, $digest]);
$db->prepare("INSERT INTO principals (uri, email, displayname) VALUES (?, ?, ?)")->execute(["principals/$user", $user, "Test User"]);
$db->exec("INSERT INTO calendars (synctoken, components) VALUES (1, 'VEVENT')");
$db->prepare("INSERT INTO calendarinstances (calendarid, principaluri, access, displayname, uri, description, calendarorder, calendarcolor, timezone, transparent) VALUES (?, ?, 1, 'Default Calendar', 'default', '', 0, '#0082C9', NULL, 0)")
   ->execute([$db->lastInsertId(), "principals/$user"]);
`
	var reader io.Reader
	exitCode, reader, err = container.Exec(ctx,
		[]string{"php", "-r", script, username, md5hex(username + ":" + baikalRealm + ":" + password)},
		tcexec.WithUser("www-data"))
	if err != nil || exitCode != 0 {
		out, _ := io.ReadAll(reader)
		return fmt.Errorf("provision user: exitCode=%d err=%v output=%s", exitCode, err, out)
	}
	return nil
}

func TestBaikal_Connect(t *testing.T) {
	cfg := config.Config{JWTSecret: "integration-test-secret-32bytes!", JWTMaxTTL: 24 * time.Hour}

	q := quirks.Detect(baikalURL, testUser, testPass)
	assert.Equal(t, "baikal", q.Name(), "should detect Baikal quirks")

	creds := auth.Credentials{
		ServerURL: baikalURL,
		Username:  testUser,
		Password:  testPass,
		Quirks:    q.Name(),
	}
	token, err := auth.Issue(cfg.JWTSecret, creds, 1*time.Hour, cfg.JWTMaxTTL)
	require.NoError(t, err)

	claims, err := auth.Validate(cfg.JWTSecret, token)
	require.NoError(t, err)

	decrypted, err := auth.Decrypt(cfg.JWTSecret, claims)
	require.NoError(t, err)
	assert.Equal(t, testUser, decrypted.Username)
}

func TestBaikal_ListCalendars(t *testing.T) {
	client := caldav.NewClient(baikalURL, testUser, testPass, quirks.Baikal{})

	home, err := caldav.Discover(client, baikalURL)
	require.NoError(t, err)
	assert.NotEmpty(t, home)

	body, err := client.Propfind(home, 1, `<prop><displayname/></prop>`)
	require.NoError(t, err)
	assert.NotEmpty(t, body)
}

func TestBaikal_CreateAndDeleteEvent(t *testing.T) {
	client := caldav.NewClient(baikalURL, testUser, testPass, quirks.Baikal{})

	home, err := caldav.Discover(client, baikalURL)
	require.NoError(t, err)

	ical := []byte(`BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:integration-test-event@calabunga
SUMMARY:Integration Test
DTSTART:20260610T100000Z
DTEND:20260610T110000Z
END:VEVENT
END:VCALENDAR`)

	path := home + "default/integration-test-event.ics"
	etag, err := client.Put(path, ical, "")
	require.NoError(t, err)
	assert.NotEmpty(t, etag)

	err = client.Delete(path, etag)
	require.NoError(t, err)
}

// Editing through calabunga must keep what other clients stored on the event.
func TestBaikal_EditKeepsAlarmZoneAndCustomProperties(t *testing.T) {
	client := caldav.NewClient(baikalURL, testUser, testPass, quirks.Baikal{})
	home, err := caldav.Discover(client, baikalURL)
	require.NoError(t, err)
	calendarPath := home + "default/"

	// A resource name that is not UID.ics, as other clients produce.
	path := calendarPath + "phone-made-event.ics"
	_, err = client.Put(path, []byte(`BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Other Client//EN
BEGIN:VTIMEZONE
TZID:America/Denver
BEGIN:STANDARD
DTSTART:19701101T020000
TZOFFSETFROM:-0600
TZOFFSETTO:-0700
END:STANDARD
BEGIN:DAYLIGHT
DTSTART:19700308T020000
TZOFFSETFROM:-0700
TZOFFSETTO:-0600
END:DAYLIGHT
END:VTIMEZONE
BEGIN:VEVENT
UID:edit-keeps-data@calabunga
DTSTAMP:20260601T000000Z
SUMMARY:Dentist
DTSTART;TZID=America/Denver:20260610T100000
DTEND;TZID=America/Denver:20260610T110000
X-CUSTOM-FLAG:keepme
BEGIN:VALARM
ACTION:DISPLAY
DESCRIPTION:Reminder
TRIGGER:-PT15M
END:VALARM
END:VEVENT
END:VCALENDAR`), "")
	require.NoError(t, err)
	defer client.Delete(path, "")

	cfg := config.Config{JWTSecret: "integration-test-secret-32bytes!", JWTMaxTTL: 24 * time.Hour}
	token, err := auth.Issue(cfg.JWTSecret, auth.Credentials{
		ServerURL: baikalURL, Username: testUser, Password: testPass, Quirks: "baikal",
	}, time.Hour, cfg.JWTMaxTTL)
	require.NoError(t, err)
	serve := func(req *http.Request) *httptest.ResponseRecorder {
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		api.JWTMiddleware(cfg)(api.NewEventsHandler(cfg)).ServeHTTP(w, req)
		return w
	}

	// List through the API, exactly as the browser does.
	w := serve(httptest.NewRequest("GET", "/api/events?calendarId="+calendarPath+"&start=2026-06-01T00:00:00Z&end=2026-07-01T00:00:00Z", nil))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var events []types.Event
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &events))
	var ev types.Event
	for _, e := range events {
		if e.UID == "edit-keeps-data@calabunga" {
			ev = e
		}
	}
	require.NotEmpty(t, ev.UID, "event should be listed")
	assert.Equal(t, path, ev.Href)

	// Edit the title and move it an hour later (the editor sends UTC).
	ev.Title = "Dentist (moved)"
	ev.Start = ev.Start.Add(time.Hour).UTC()
	ev.End = ev.End.Add(time.Hour).UTC()
	body, _ := json.Marshal(ev)
	w = serve(httptest.NewRequest("PUT", "/api/events/"+ev.UID+"?editScope=all&calendarId="+calendarPath, strings.NewReader(string(body))))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	stored, _, err := client.Get(path)
	require.NoError(t, err)
	got := string(stored)
	assert.Contains(t, got, "SUMMARY:Dentist (moved)")
	assert.Contains(t, got, "BEGIN:VTIMEZONE")
	assert.Contains(t, got, "BEGIN:VALARM")
	assert.Contains(t, got, "TRIGGER:-PT15M")
	assert.Contains(t, got, "X-CUSTOM-FLAG:keepme")
	assert.Contains(t, got, "DTSTART;TZID=America/Denver:20260610T110000")
}

// A series created and edited through calabunga must stay a series on the server.
func TestBaikal_RecurringSeriesRoundTrip(t *testing.T) {
	client := caldav.NewClient(baikalURL, testUser, testPass, quirks.Baikal{})
	home, err := caldav.Discover(client, baikalURL)
	require.NoError(t, err)
	calendarPath := home + "default/"

	cfg := config.Config{JWTSecret: "integration-test-secret-32bytes!", JWTMaxTTL: 24 * time.Hour}
	token, err := auth.Issue(cfg.JWTSecret, auth.Credentials{
		ServerURL: baikalURL, Username: testUser, Password: testPass, Quirks: "baikal",
	}, time.Hour, cfg.JWTMaxTTL)
	require.NoError(t, err)
	serve := func(req *http.Request) *httptest.ResponseRecorder {
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		api.JWTMiddleware(cfg)(api.NewEventsHandler(cfg)).ServeHTTP(w, req)
		return w
	}

	// Create a weekly Wednesday series the way the editor does.
	start := time.Date(2026, 6, 3, 15, 0, 0, 0, time.UTC) // a Wednesday
	created := types.Event{
		UID: "series-roundtrip@calabunga", CalendarID: calendarPath, Title: "Standup",
		Start: start, End: start.Add(30 * time.Minute),
		Recurrence: &types.Recurrence{Frequency: "weekly", Interval: 1, ByDay: []string{"WE"}, End: types.RecurrenceEnd{Type: "never"}},
	}
	body, _ := json.Marshal(created)
	w := serve(httptest.NewRequest("POST", "/api/events", strings.NewReader(string(body))))
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var made types.Event
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &made))
	defer client.Delete(made.Href, "")

	list := func() []types.Event {
		w := serve(httptest.NewRequest("GET", "/api/events?calendarId="+calendarPath+"&start=2026-06-01T00:00:00Z&end=2026-07-01T00:00:00Z", nil))
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var all []types.Event
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &all))
		var mine []types.Event
		for _, e := range all {
			if e.UID == created.UID {
				mine = append(mine, e)
			}
		}
		return mine
	}

	occurrences := list()
	require.Len(t, occurrences, 4, "Wednesdays June 3, 10, 17, 24")
	require.NotNil(t, occurrences[0].Recurrence)

	// Open the June 17 occurrence, rename it and move it 30 minutes later.
	opened := occurrences[2]
	edited := opened
	edited.Title = "Standup (renamed)"
	edited.Start = opened.Start.Add(30 * time.Minute)
	edited.End = opened.End.Add(30 * time.Minute)
	edited.OriginalStart = &opened.Start
	body, _ = json.Marshal(edited)
	w = serve(httptest.NewRequest("PUT", "/api/events/"+edited.UID+"?editScope=all&calendarId="+calendarPath, strings.NewReader(string(body))))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	stored, _, err := client.Get(made.Href)
	require.NoError(t, err)
	assert.Contains(t, string(stored), "RRULE:FREQ=WEEKLY;BYDAY=WE")
	assert.Contains(t, string(stored), "SUMMARY:Standup (renamed)")
	assert.Contains(t, string(stored), "DTSTART:20260603T153000Z", "the series start keeps its own date")

	after := list()
	require.Len(t, after, 4, "still a series of four Wednesdays")
	assert.Equal(t, time.Date(2026, 6, 17, 15, 30, 0, 0, time.UTC), after[2].Start.UTC())

	// Moving the series to another day is refused rather than corrupting the rule.
	moved := after[2]
	origStart := moved.Start
	moved.Start = moved.Start.Add(24 * time.Hour)
	moved.End = moved.End.Add(24 * time.Hour)
	moved.OriginalStart = &origStart
	body, _ = json.Marshal(moved)
	w = serve(httptest.NewRequest("PUT", "/api/events/"+moved.UID+"?editScope=all&calendarId="+calendarPath, strings.NewReader(string(body))))
	assert.Equal(t, http.StatusNotImplemented, w.Code)
}

// A series edited elsewhere (a deleted occurrence, a moved one) must list correctly.
func TestBaikal_ListHonorsExdateAndMovedOccurrences(t *testing.T) {
	client := caldav.NewClient(baikalURL, testUser, testPass, quirks.Baikal{})
	home, err := caldav.Discover(client, baikalURL)
	require.NoError(t, err)
	calendarPath := home + "default/"

	path := calendarPath + "edited-elsewhere.ics"
	_, err = client.Put(path, []byte(`BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Other Client//EN
BEGIN:VEVENT
UID:edited-elsewhere@calabunga
DTSTAMP:20260601T000000Z
SUMMARY:Standup
DTSTART:20260602T100000Z
DTEND:20260602T103000Z
RRULE:FREQ=WEEKLY;BYDAY=TU
EXDATE:20260609T100000Z
END:VEVENT
BEGIN:VEVENT
UID:edited-elsewhere@calabunga
DTSTAMP:20260601T000000Z
RECURRENCE-ID:20260616T100000Z
SUMMARY:Standup (moved)
DTSTART:20260617T140000Z
DTEND:20260617T143000Z
END:VEVENT
END:VCALENDAR`), "")
	require.NoError(t, err)
	defer client.Delete(path, "")

	cfg := config.Config{JWTSecret: "integration-test-secret-32bytes!", JWTMaxTTL: 24 * time.Hour}
	token, err := auth.Issue(cfg.JWTSecret, auth.Credentials{
		ServerURL: baikalURL, Username: testUser, Password: testPass, Quirks: "baikal",
	}, time.Hour, cfg.JWTMaxTTL)
	require.NoError(t, err)
	req := httptest.NewRequest("GET", "/api/events?calendarId="+calendarPath+"&start=2026-06-01T00:00:00Z&end=2026-06-30T00:00:00Z", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	api.JWTMiddleware(cfg)(api.NewEventsHandler(cfg)).ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var events []types.Event
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &events))
	var got []string
	for _, e := range events {
		if e.UID == "edited-elsewhere@calabunga" {
			got = append(got, e.Start.UTC().Format("0102T1504"))
		}
	}
	// June 9 was deleted, June 16 moved to June 17 14:00.
	assert.ElementsMatch(t, []string{"0602T1000", "0623T1000", "0617T1400"}, got)
}

// Notifications set through calabunga are stored as VALARMs and can be changed.
func TestBaikal_AlarmsRoundTrip(t *testing.T) {
	client := caldav.NewClient(baikalURL, testUser, testPass, quirks.Baikal{})
	home, err := caldav.Discover(client, baikalURL)
	require.NoError(t, err)
	calendarPath := home + "default/"

	cfg := config.Config{JWTSecret: "integration-test-secret-32bytes!", JWTMaxTTL: 24 * time.Hour}
	token, err := auth.Issue(cfg.JWTSecret, auth.Credentials{
		ServerURL: baikalURL, Username: testUser, Password: testPass, Quirks: "baikal",
	}, time.Hour, cfg.JWTMaxTTL)
	require.NoError(t, err)
	serve := func(req *http.Request) *httptest.ResponseRecorder {
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		api.JWTMiddleware(cfg)(api.NewEventsHandler(cfg)).ServeHTTP(w, req)
		return w
	}

	start := time.Date(2026, 6, 12, 9, 0, 0, 0, time.UTC)
	body, _ := json.Marshal(types.Event{
		UID: "alarms-roundtrip@calabunga", CalendarID: calendarPath, Title: "Flight",
		Start: start, End: start.Add(time.Hour), Alarms: []int{10, 1440},
	})
	w := serve(httptest.NewRequest("POST", "/api/events", strings.NewReader(string(body))))
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var made types.Event
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &made))
	defer client.Delete(made.Href, "")

	list := func() types.Event {
		w := serve(httptest.NewRequest("GET", "/api/events?calendarId="+calendarPath+"&start=2026-06-01T00:00:00Z&end=2026-07-01T00:00:00Z", nil))
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var all []types.Event
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &all))
		for _, e := range all {
			if e.UID == "alarms-roundtrip@calabunga" {
				return e
			}
		}
		t.Fatal("event not listed")
		return types.Event{}
	}

	ev := list()
	assert.ElementsMatch(t, []int{10, 1440}, ev.Alarms)

	ev.Alarms = []int{60}
	body, _ = json.Marshal(ev)
	w = serve(httptest.NewRequest("PUT", "/api/events/"+ev.UID+"?editScope=all&calendarId="+calendarPath, strings.NewReader(string(body))))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	stored, _, err := client.Get(made.Href)
	require.NoError(t, err)
	assert.Contains(t, string(stored), "TRIGGER:-PT1H")
	assert.NotContains(t, string(stored), "TRIGGER:-PT10M")
	assert.NotContains(t, string(stored), "TRIGGER:-P1D")
	assert.Equal(t, []int{60}, list().Alarms)
}

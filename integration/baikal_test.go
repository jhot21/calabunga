//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"calabunga/internal/auth"
	"calabunga/internal/caldav"
	"calabunga/internal/config"
	"calabunga/internal/quirks"
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

func provisionTestUser(ctx context.Context, container testcontainers.Container, username, password string) error {
	phpCmd := []string{"php", "-r",
		fmt.Sprintf("echo '{SHA256}' . base64_encode(hash('sha256', %q, true));", password),
	}
	exitCode, reader, err := container.Exec(ctx, phpCmd)
	if err != nil || exitCode != 0 {
		return fmt.Errorf("php hash: exitCode=%d err=%v", exitCode, err)
	}
	out, _ := io.ReadAll(reader)
	hash := strings.TrimSpace(string(out))

	dbPath := "/var/www/baikal/Specific/db/db.sqlite"

	principalSQL := fmt.Sprintf(
		"INSERT OR IGNORE INTO principals (uri,email,displayname,vcardurl,password) VALUES ('principals/%s','%s','Test User',NULL,'%s');",
		username, username, hash,
	)
	exitCode, _, err = container.Exec(ctx, []string{"sqlite3", dbPath, principalSQL})
	if err != nil || exitCode != 0 {
		return fmt.Errorf("insert principal: exitCode=%d err=%v", exitCode, err)
	}

	calSQL := fmt.Sprintf(
		"INSERT OR IGNORE INTO calendars (synctoken,components) VALUES (1,'VEVENT');"+
			"INSERT OR IGNORE INTO calendarinstances"+
			" (calendarid,principaluri,access,displayname,description,calendarorder,calendarcolor,timezone,transparent)"+
			" VALUES (last_insert_rowid(),'principals/%s',1,'Default Calendar','',0,'#0082C9',NULL,0);",
		username,
	)
	exitCode, _, err = container.Exec(ctx, []string{"sqlite3", dbPath, calSQL})
	if err != nil || exitCode != 0 {
		return fmt.Errorf("insert calendar: exitCode=%d err=%v", exitCode, err)
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

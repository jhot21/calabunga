# calabunga

A small, self-hosted web calendar for any CalDAV server. Log in with your CalDAV URL and credentials, and calabunga shows your calendars and events in a month or week view and lets you create, edit, and delete events.

It is a single Go binary with the React frontend embedded, and it keeps no database. The server holds no state of its own: your CalDAV account is the source of truth.

## Features

- Month and week views, with a mini-calendar and per-calendar show/hide
- Create, edit, and delete events, including all-day and recurring events
- Optimistic updates, with conflict detection when an event changed on the server (ETags)
- Calendars and events appear instantly on reload from a local cache, then refresh in the background
- Light, dark, and system themes
- Works with standard CalDAV servers; [Baikal](https://sabre.io/baikal/) is auto-detected and handled

Edits are written in place, so alarms, attendees, and other data stored by other clients survive a save in calabunga.

Not yet supported on recurring events:

- Editing or deleting a single occurrence, or "this and following". Editing the whole series works, and deleting one occurrence is refused rather than deleting the series.
- Moving a whole series to a different day. Changing its time or length works.
- Recurrence rules the editor can't express (such as "first Tuesday of the month") are shown as one-off events but are kept as stored when you edit the event.

## How it works

The browser talks only to calabunga's `/api`, and calabunga talks to your CalDAV server.

On login, calabunga checks your credentials against the CalDAV server, then issues a signed JWT. Your username and password are encrypted (AES-GCM, key derived from `JWT_SECRET`) and carried inside that token. Nothing is stored server-side, and each request decrypts the credentials to talk to CalDAV on your behalf. Sessions are refreshed as they near expiry, up to a maximum lifetime.

Consequences worth knowing:

- Anyone who has `JWT_SECRET` and a token can recover that token's credentials. Keep the secret private.
- Changing `JWT_SECRET`, or restarting without setting it, invalidates every session.
- Serve calabunga over HTTPS (for example behind a reverse proxy). Credentials cross the wire at login.
- The browser keeps a copy of your calendars and events in `localStorage` for up to 24 hours so reloads are instant. It is cleared on logout, session expiry, and a new login.

## Quick start

With Docker Compose:

```sh
export JWT_SECRET="$(openssl rand -hex 32)"
docker compose up -d --build
```

Then open <http://localhost:8080> (or the `PORT` you set) and connect with your CalDAV URL, for example `https://dav.example.com/dav.php`.

A prebuilt multi-arch image is published to `ghcr.io/jhot21/calabunga` on every push to `main`.

## Configuration

All configuration is through environment variables.

| Variable          | Default            | Description                                                         |
| ----------------- | ------------------ | ------------------------------------------------------------------- |
| `PORT`            | `8080`             | Port to listen on                                                   |
| `JWT_SECRET`      | random per start   | Signs session tokens and encrypts credentials. **Set this.**         |
| `JWT_DEFAULT_TTL` | `24h`              | Session length when the login form doesn't ask for one              |
| `JWT_MAX_TTL`     | `720h`             | Longest session a user can request                                  |

If `JWT_SECRET` is unset, a random one is generated on each start, so sessions don't survive a restart.

## Development

Requirements: Go and Node (versions pinned in `mise.toml`) and [Task](https://taskfile.dev).

```sh
task build        # build the frontend and the Go binary into bin/calabunga
task run          # build and run
task test         # Go unit tests
task dev          # Go server plus the Vite dev server
```

The Vite dev server proxies `/api` to `localhost:8080`, so run the Go server with `PORT=8080` when using `task dev`.

Frontend tests: `npm --prefix web test`.

The integration tests run against a real Baikal in Docker:

```sh
task test:integration
```

### Layout

```
cmd/server        entry point
internal/api      HTTP handlers (connect, calendars, events) and JWT middleware
internal/auth     token issuing, validation, credential encryption
internal/caldav   CalDAV client, discovery, iCalendar and RRULE parsing
internal/quirks   per-server workarounds (Baikal)
web               React + Vite frontend, embedded into the binary
```

## License

[MIT](LICENSE)

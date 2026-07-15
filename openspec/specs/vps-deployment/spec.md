# vps-deployment Specification

## Purpose

Diese Spezifikation beschreibt die Capability `vps-deployment`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)
## Requirements
### Requirement: Single Go binary serves frontend and API
The system SHALL be packaged as a single Go binary that embeds the React production build via `embed.FS` and serves both the API routes and the static frontend from the same process.

#### Scenario: API request routing
- **WHEN** a request arrives at `/api/*`
- **THEN** the Go Chi router handles it and returns JSON

#### Scenario: Frontend asset serving
- **WHEN** a request arrives at any non-API path
- **THEN** the Go binary serves the embedded React build (index.html for SPA routes, static assets for `/assets/*`)

### Requirement: Systemd service management
The system SHALL run as a systemd service on the VPS, automatically starting on boot and restarting on crash.

#### Scenario: Service starts on boot
- **WHEN** the VPS reboots
- **THEN** the `vereinswerk.service` systemd unit starts automatically and the app is reachable within 30 seconds

#### Scenario: Service restarts on crash
- **WHEN** the Go process exits unexpectedly
- **THEN** systemd restarts it within 5 seconds (`Restart=on-failure`, `RestartSec=5`)

### Requirement: HTTPS via Nginx reverse proxy and Let's Encrypt

The system SHALL be reachable exclusively over HTTPS. Nginx terminates TLS and proxies to the Go binary on port 8080. Certificates are managed by Certbot (Let's Encrypt).

The primary user-facing hostname SHALL be `teamwerk.team-stuttgart.org`. During the domain-rename transition the legacy hostname `internal.team-stuttgart.org` SHALL be served by the same Nginx instance as an alias (same `server` block, single Let's Encrypt certificate with both hostnames as SANs). Both hostnames SHALL return the full application (not a redirect) during the transition; the follow-up flip to a `301` on `internal.*` is out of scope for this capability and handled by a separate change.

#### Scenario: HTTP redirected to HTTPS on primary hostname
- **WHEN** a client accesses `http://teamwerk.team-stuttgart.org`
- **THEN** Nginx returns HTTP 301 to the HTTPS URL

#### Scenario: HTTP redirected to HTTPS on transitional alias
- **WHEN** a client accesses `http://internal.team-stuttgart.org`
- **THEN** Nginx returns HTTP 301 to `https://internal.team-stuttgart.org$request_uri` (same host, not to `teamwerk.*`)

#### Scenario: Alias hostname serves the full app during transition
- **WHEN** an authenticated client sends `GET https://internal.team-stuttgart.org/api/dashboard` with a valid bearer token
- **THEN** the request is proxied to the Go binary and the JSON dashboard payload is returned (identical to the response on `teamwerk.*`)

#### Scenario: Single certificate covers both hostnames
- **WHEN** the operator runs `certbot certificates` after issuance
- **THEN** exactly one certificate is listed, with SANs including both `teamwerk.team-stuttgart.org` and `internal.team-stuttgart.org`, and its renewal cronjob is active

#### Scenario: Certificate auto-renewal
- **WHEN** the Let's Encrypt certificate is within 30 days of expiry
- **THEN** Certbot's renewal Cronjob renews the certificate without downtime and keeps both SANs

### Requirement: SQLite database persisted outside binary
The system SHALL store the SQLite database file at a fixed path outside the binary directory (`/var/lib/vereinswerk/vereinswerk.db`) so that deployments do not overwrite data.

#### Scenario: Data survives binary update
- **WHEN** a new binary is deployed via `make deploy`
- **THEN** the SQLite database file is unchanged and all data is intact after service restart

#### Scenario: WAL mode enabled
- **WHEN** the application starts and opens the database
- **THEN** it executes `PRAGMA journal_mode=WAL` to enable concurrent reads during writes

### Requirement: Deployment via Makefile
The system SHALL provide a `Makefile` with targets for local build, remote deployment, and migration execution.

#### Scenario: Build target
- **WHEN** `make build` is executed locally
- **THEN** a production Go binary is compiled with the React build embedded

#### Scenario: Deploy target
- **WHEN** `make deploy` is executed locally
- **THEN** the binary is transferred to the VPS via rsync, the systemd service is restarted, and pending database migrations are applied

### Requirement: Scheduled tasks via system Cronjob
The system SHALL expose a CLI subcommand (`./vereinswerk scheduler:run`) that executes all pending scheduled tasks. This command SHALL be invoked every minute via a system Cronjob.

#### Scenario: Cronjob invokes scheduler
- **WHEN** the system Cronjob fires every minute
- **THEN** `./vereinswerk scheduler:run` executes all due tasks (e.g., invitation expiry cleanup, reminder e-mails) and exits

#### Scenario: Scheduler run is idempotent
- **WHEN** `scheduler:run` is called while a previous run is still executing
- **THEN** the second invocation exits immediately without duplicate processing (file lock or DB lock)

### Requirement: Transitional-hostname migration banner

The frontend SHALL display a persistent, non-dismissable banner on every page load when the browser's origin is the transitional alias `internal.team-stuttgart.org`. The banner SHALL instruct the user to switch to the primary hostname `teamwerk.team-stuttgart.org`, reinstall the PWA there, and log in again once. The banner SHALL provide a primary call-to-action link to `https://teamwerk.team-stuttgart.org` that preserves the current path and query string.

The banner SHALL NOT appear on the primary hostname or in local development.

The application SHALL remain fully functional on the transitional alias while the banner is shown — the banner is an in-app hint, not a hard block.

#### Scenario: Banner shown on transitional alias
- **WHEN** the SPA is loaded at `https://internal.team-stuttgart.org/dashboard?tab=x`
- **THEN** the banner is rendered above the app shell, its CTA link points to `https://teamwerk.team-stuttgart.org/dashboard?tab=x`, and the dashboard beneath the banner is fully interactive

#### Scenario: Banner hidden on primary hostname
- **WHEN** the SPA is loaded at `https://teamwerk.team-stuttgart.org/dashboard`
- **THEN** no migration banner is rendered

#### Scenario: Banner hidden in local development
- **WHEN** the SPA is loaded at `http://localhost:5173/` (or any other host)
- **THEN** no migration banner is rendered

### Requirement: Backend-generated deep links use the primary hostname

All URLs constructed by the backend for delivery to users (e-mail bodies from mailer/notify, scheduler reminders, invitation links, password-reset links, push-notification target URLs where absolute) SHALL be built from `cfg.BaseURL`. `cfg.BaseURL` SHALL default to `https://teamwerk.team-stuttgart.org` when the `BASE_URL` environment variable is unset. No handler or scheduler component SHALL construct a link containing the substring `internal.team-stuttgart.org`.

#### Scenario: Config default points at the primary hostname
- **WHEN** the application starts without a `BASE_URL` environment variable
- **THEN** `cfg.BaseURL` is `"https://teamwerk.team-stuttgart.org"`

#### Scenario: Duty reminder e-mail uses configured base URL
- **WHEN** the scheduler emits a duty-board reminder e-mail with `cfg.BaseURL = "https://example.test"`
- **THEN** the e-mail body contains the deep link `https://example.test/duty-board` and does not contain the substring `internal.team-stuttgart.org`

#### Scenario: Notify helper uses configured base URL
- **WHEN** `notify.Send(cfg, …, url = "/foo")` is called with `cfg.BaseURL = "https://example.test"`
- **THEN** the outbound e-mail body's `Direktlink:` line reads `https://example.test/foo`


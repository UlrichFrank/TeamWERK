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

#### Scenario: HTTP redirected to HTTPS
- **WHEN** a client accesses `http://intern.team-stuttgart.org`
- **THEN** Nginx returns HTTP 301 to the HTTPS URL

#### Scenario: Certificate auto-renewal
- **WHEN** the Let's Encrypt certificate is within 30 days of expiry
- **THEN** Certbot's renewal Cronjob renews the certificate without downtime

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

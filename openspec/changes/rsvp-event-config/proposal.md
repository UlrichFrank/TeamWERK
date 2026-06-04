## Why

Trainer haben keine Möglichkeit festzulegen, ob Spieler bei einem Termin aktiv zusagen müssen
oder standardmäßig als zugesagt gelten (Opt-Out). Außerdem ist die Pflicht zur Begründung
bei Absage/Vielleicht nicht konfigurierbar — für lockere Events wie ein Sommerfest ist ein
Pflichtkommentar sinnlos, für Pflichttrainings aber essenziell.

## What Changes

- Neue Felder `rsvp_opt_out` und `rsvp_require_reason` auf `training_series`, `training_sessions`
  und `games`
- Sessions erben die Flags einmalig von der Serie beim Anlegen (keine Live-Vererbung);
  beim Bearbeiten bestehender Sessions sind die Flags nicht änderbar
- Bei `rsvp_opt_out = 1`: Spieler ohne Eintrag in der Response-Tabelle gelten als "confirmed";
  auf der `/termine`-Seite erscheint der Zusagen-Button in der Termin-Kachel bereits aktiv/
  ausgewählt — auch ohne dass der Spieler je reagiert hat (visuelle Rückmeldung des impliziten Status)
- Bei `rsvp_require_reason = 0`: Klick auf Absagen/Vielleicht sendet die RSVP sofort ohne
  Modal-Dialog
- Bei `rsvp_require_reason = 1`: Klick auf Absagen/Vielleicht öffnet ein Modal mit
  Pflichtbegründung (ersetzt den noch offenen `rsvp-reason-modal`-Change vollständig)
- Die confirmed-Zählung im Backend berücksichtigt bei Opt-Out-Terminen alle Spieler ohne Eintrag
- Standard für generische Events (`event_type = 'generisch'`): `rsvp_require_reason = 0`

## Capabilities

### New Capabilities

- `rsvp-event-config`: Konfiguration von RSVP-Verhalten pro Termin — Opt-Out-Modus und
  Begründungspflicht; steuert Modal-Anzeige und confirmed-Zählung

### Modified Capabilities

- `rsvp`: RSVP-Verhalten für Trainings und Spiele ändert sich: Modal erscheint nur noch
  wenn `rsvp_require_reason = 1`; Zusagen-Button kann vorausgewählt sein

## Impact

- **DB**: Migration 015 — `rsvp_opt_out` und `rsvp_require_reason` auf `training_series`,
  `training_sessions`, `games`
- **Backend**: `internal/trainings/handler.go` (RSVP-Response, confirmed-Count),
  `internal/games/handler.go` (analog); Create-Handler kopiert Flags von Serie auf Session
- **Frontend**: `TerminePage.tsx` (Modal-Logik konditionalisieren, Opt-Out-UI),
  Kalender-Wizard / `AdminTrainingsPage.tsx` (Flags beim Anlegen von Serie/Spiel konfigurierbar)
- Ersetzt den nicht implementierten `rsvp-reason-modal`-Change

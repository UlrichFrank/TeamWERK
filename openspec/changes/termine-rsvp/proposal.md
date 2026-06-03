## Why

Spieler und Eltern müssen aktuell Trainings über `/trainings` bestätigen — für Spiele gibt es kein
äquivalentes RSVP. Trainer haben keine zentrale Übersicht über Zu-/Absagen zu Spielterminen.
Eine gemeinsame Seite für Trainings und Spiele vereinfacht Navigation und reduziert doppelten Code.

## What Changes

- Neue Route `/termine`: Chronologisch gemischte Liste aus Trainings und Spielen des eigenen Teams,
  jeweils mit RSVP-Buttons (Zusagen / Vielleicht / Absagen + optionaler Grund)
- Neue Detailrouten `/termine/training/:id` und `/termine/spiel/:id` für Trainer-Übersicht
  (Rückmeldungs-Tabelle pro Termin)
- Neue DB-Tabelle `game_responses` für Spieler-Rückmeldungen zu Spielen
- Neue API-Endpunkte: `POST /api/games/{id}/respond`, `GET /api/games/my`,
  `GET /api/games/{id}/responses`
- Bestehende `/trainings` und `/trainings/:id` Routen werden durch `/termine` ersetzt
  und aus der Navigation entfernt
- Bei Spielen ist **keine Status-Vorauswahl** gesetzt — Spieler müssen aktiv reagieren
  (konsistent mit dem bestehenden Verhalten bei Trainings)
- Trainer-Detailseite für Spiele hat kein Anwesenheits-Tracking (nur RSVP-Übersicht);
  Trainings behalten das bestehende Anwesenheits-Tracking

## Capabilities

### New Capabilities

- `game-rsvp`: Spieler und Eltern können zu Spielen zu-/absagen (confirmed/declined/maybe)
  mit optionalem Grund; Trainer sehen Rückmeldungs-Übersicht pro Spiel
- `termine-unified-view`: Gemeinsame chronologische Liste aus Trainings und Spielen
  mit RSVP-Interaktion an einem Ort

### Modified Capabilities

- `training-rsvp`: Bestehende Training-RSVP-Funktionalität wird in `/termine` integriert;
  `/trainings` und `/trainings/:id` werden abgelöst (gleiche Anforderungen, neue URL-Struktur)

## Impact

- **DB**: Migration 012 — neue Tabelle `game_responses`
- **Backend**: `internal/games/handler.go` — 3 neue Handler-Methoden; neue Routen in `main.go`
- **Frontend**: Neue Seiten `TerminePage.tsx` und `TermineDetailPage.tsx`;
  `TrainingsPage.tsx` und `TrainingsDetailPage.tsx` werden obsolet;
  `AppShell.tsx` Navigation: „Trainings" → „Termine"
- **Routing**: `App.tsx` — `/termine`, `/termine/training/:id`, `/termine/spiel/:id`

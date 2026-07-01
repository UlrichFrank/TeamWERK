## Why

Spieler und Eltern können heute ihren RSVP-Status (zusagen / absagen / vielleicht) für Trainings und Spiele bis unmittelbar vor Beginn ändern. In der Praxis führt das zu Spätabsagen ohne Reibung („30 Min vor Training noch schnell absagen"), die für Trainer und Mannschaftsplanung nicht mehr verwertbar sind. Wir wollen einen Cutoff einführen, vor dem normale RSVP-Änderungen möglich sind und nach dem die Mitglieder ihre Absage direkt bei Trainer oder Vorstand melden müssen.

## What Changes

- **Training-RSVP:** `POST /api/training-sessions/{id}/respond` lehnt jeden Statuswechsel ab, wenn weniger als **2 Stunden** bis Session-Beginn (Europe/Berlin) verbleiben — neue Antwort, Statuswechsel und Reason-Änderung sind alle gesperrt. Antwort: HTTP **422** mit Fehlertext.
- **Spiel-RSVP:** `POST /api/games/{id}/respond` analog mit **18 Stunden** Cutoff.
- **Override** für `admin`, `vorstand` (System-Rolle / Vereinsfunktion) und Vereinsfunktionen `trainer` / `sportliche_leitung`: dürfen auch nach Cutoff RSVP für beliebige Members pflegen.
- **API-Listing** liefert pro Termin `rsvp_locks_at` (RFC3339, Europe/Berlin → UTC) mit, damit das Frontend ohne Eigenrechnung weiß, wann gesperrt ist:
  - `GET /api/training-sessions`, `GET /api/training-sessions/{id}`
  - `GET /api/games`, `GET /api/games/my`, `GET /api/games/{id}`
- **Frontend (Variante B — Hard Lock):** Nach Cutoff sind die RSVP-Buttons disabled mit dem Hinweis „Bis HH:MM Uhr änderbar — danach bitte direkt beim Trainer melden". Vor Cutoff zeigt ein subtiler Hinweis die Sperrzeit.
- **Zeitberechnung in Go** (`time.LoadLocation("Europe/Berlin")`), nicht in SQL — wegen DST.
- **Konstanten** statt Konfiguration: `TrainingCutoff = 2h`, `GameCutoff = 18h`. Pro-Verein-Konfigurierbarkeit kommt erst, wenn ein zweiter Verein das anders haben will.

**Nicht betroffen:**

- Spielabsage durch den Verein (`status='cancelled'` am Game/Session): eigene Mechanik, nicht vom RSVP-Cutoff betroffen.
- `member_absences`-Lock (`absence_id IS NOT NULL`): bleibt HTTP 403 und hat Vorrang vor dem Cutoff-Check (Absence-Sperre prüft zuerst).
- Anwesenheits-Eintragung (`/attendances`-Endpoints, separate Tabellen `training_attendances` / `game_attendances`): unverändert.

## Capabilities

### New Capabilities

_(keine — der Cutoff hängt fachlich an den bestehenden RSVP-Capabilities)_

### Modified Capabilities

- `training-rsvp`: neuer Cutoff (2 h vor Session-Beginn) für Spieler/Eltern; Trainer/Vorstand/Admin-Override; `rsvp_locks_at` in Listing & Detail.
- `game-rsvp`: neuer Cutoff (18 h vor Spielbeginn) für Spieler/Eltern; Trainer/Vorstand/Admin-Override; `rsvp_locks_at` in Listing & Detail.

## Impact

- **Code (Backend):**
  - `internal/trainings/handler.go` — `Respond` + Listing/Detail-Queries
  - `internal/games/handler.go` — `RespondToGame` + Listing/Detail-Queries (`ListMyGames`, `ListGames`, `GetGame`)
  - `internal/auth` — vermutlich neuer Helper `claims.CanManageTeamEvents()` (oder ähnlich) zur Bündelung des Override-Checks
  - neue Konstanten + Test-Helper für „Zeit einfrieren" (Clock-Injection im Handler-Struct)
- **Code (Frontend):**
  - `web/src/pages/Termine.tsx` (Liste) und ggf. Detail-Pages für Game und Training
  - Komponenten, die die drei RSVP-Buttons rendern (z. B. `RsvpButtons`, falls vorhanden)
  - `lib/api.ts` — keine Änderung
- **API:**
  - Neue Felder in mehreren Listing-/Detail-Responses (`rsvp_locks_at`).
  - Neuer HTTP-Status 422 mit definiertem Fehlertext auf den beiden `/respond`-Endpoints.
- **DB:** keine Migration nötig (Cutoff ergibt sich aus `date`+`time`/`start_time` zur Laufzeit).
- **Tests:** je Route Happy-Path, Spieler nach Cutoff (422), Eltern nach Cutoff (422), Trainer nach Cutoff (204), bestehender `absence_lock` weiterhin 403.

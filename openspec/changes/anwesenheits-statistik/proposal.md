## Why

Trainer haben heute keine Übersicht über die Beteiligung ihrer Spieler an Trainings und Spielen — `training_attendances` wird zwar erfasst, aber nirgends aggregiert. Spiel-Anwesenheit ist überhaupt nicht modelliert (nur RSVP und Nominierung). Spieler und Eltern haben keinen Einblick in die eigene Quote, und sportliche Leitung kann Teams nicht vergleichen. Eine Statistik schafft Transparenz für faire Spielerentscheidungen und Selbstreflexion.

## What Changes

- **Neue Tabelle `game_attendances`** (analog `training_attendances`): post-hoc Erfassung von Spiel-Anwesenheit durch den Trainer.
- **Neuer Erfassungs-Endpoint** `POST /api/games/{id}/attendances` + Lese-Endpoint `GET /api/games/{id}/attendances` (analog Trainings; nur vergangene Spiele, Trainer eigenes Team / sL / admin).
- **Drei-Säulen-Statistik-Aggregation** (anwesend / entschuldigt / fehlt) als API:
  - `GET /api/teams/{id}/attendance-stats?season=<id>` — Team-Aggregat (Stammkader + erweiterter Kader separat, plus Team-Durchschnitte).
  - `GET /api/members/{id}/attendance-stats?season=<id>` — Einzel-Statistik plus Termin-Detailliste.
  - `GET /api/teams/{id}/attendance-open` — vergangene Termine ohne Erfassung (für Trainer-Banner).
- **Frontend-Seiten:**
  - Trainer-Sicht `/team/{id}/anwesenheit` (Säulen+Quote-Tabelle, offene-Erfassungen-Banner, Termin-Drilldown).
  - Spieler-/Eltern-Sicht `/profil/anwesenheit` (Selbstsicht + Kind-Auswahl, Termin-Listen).
  - Sportliche Leitung nutzt Trainer-Sicht mit Team-Auswahl.
- **Push-Erinnerung** an Trainer: täglicher Scheduler-Job sendet aggregierte Push mit Liste aller offenen Erfassungen seines Teams; Stopp pro Termin sobald irgendein Trainer erfasst hat; Cut-off am Ende der aktiven Saison; idempotent via `notification_log` (1 Push/Trainer/Tag).
- **Live-Updates:** `POST /api/games/{id}/attendances` broadcastet `attendance-changed` über `h.hub.Broadcast`.

## Capabilities

### New Capabilities

- `game-attendance`: Post-hoc Anwesenheitserfassung durch den Trainer für vergangene Spiele (parallel zur bestehenden `training-attendance`).
- `attendance-statistics`: Aggregat- und Detailstatistiken (Drei-Säulen-Modell anwesend/entschuldigt/fehlt) über Trainings und Spiele, bezogen auf den aktiven Kader der aktiven Saison.
- `attendance-reminders`: Tägliche aggregierte Push-Erinnerungen an Trainer mit offenen Anwesenheits-Erfassungen, mit Idempotenz und Saisonende-Cut-off.

### Modified Capabilities

_Keine. Bestehende `training-attendance`-Capability wird nicht modifiziert (nur gelesen). RSVP- und Kader-Capabilities bleiben unverändert._

## Impact

- **Backend:**
  - Neue Migration `012_game_attendances.up.sql/.down.sql`.
  - Neue Routen im `internal/games/`-Handler (POST/GET attendances).
  - Neues `internal/attendance/`-Package (oder Erweiterung `internal/games/`/`internal/trainings/`) für die Aggregations-Endpoints, Stats-Logik, Drei-Säulen-Berechnung.
  - Erweiterung `internal/scheduler/` um den Reminder-Job, plus Eintrag in `notification_log` (`kind='attendance-reminder'`).
  - `internal/app/router.go`: neue Routen unter passenden Auth-Tiers (Trainer+sL für Schreiben/Lesen, Spieler/Eltern für Member-Stats nach Authz-Check).
- **Frontend:**
  - Neue Seiten `web/src/pages/TeamAnwesenheitPage.tsx`, `web/src/pages/ProfilAnwesenheitPage.tsx`.
  - Erweiterung Spiel-Detailseite um Attendance-Erfassung (Bulk-Form, analog Training).
  - Live-Updates via `useLiveUpdates('attendance-changed')`.
  - Neuer Nav-/Profil-Tab-Eintrag.
- **Datenbank:**
  - Nur additiv (neue Tabelle, ein neuer `notification_log.kind`-Wert).
- **Push:**
  - Neue Notification-Kategorie; nutzt bestehende VAPID-/`push`-Infrastruktur.
- **Tests:** zwingend Happy-Path + Fehlerfall pro neuer Route, plus Aggregations- und Scheduler-Tests.
- **VPS-Footprint:** vernachlässigbar (zusätzliche tägliche Aggregation, keine neuen Dienste).

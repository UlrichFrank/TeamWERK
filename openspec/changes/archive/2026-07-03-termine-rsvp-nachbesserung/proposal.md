## Why

Zwei kleine Korrekturen an der frisch ausgelieferten RSVP-Funktion (`rsvp-defaults-per-rolle` + `termine-trainer-rsvp`):

1. **Die Konflikt-Sperre zwischen „Standardmäßig abgesagt" und „Begründung bei Absage erforderlich" ist unnötig restriktiv.** Beide Einstellungen betreffen **disjunkte** Personengruppen und stehen nie im Widerspruch:
   - `rsvp_default_players/extended='declined'` gilt für Mitglieder, die **nicht** reagieren → virtuelle Absage, es wird **nie** eine `training_responses`/`game_responses`-Zeile erzeugt und **nie** ein Grund erfragt.
   - `rsvp_require_reason=1` greift **nur**, wenn ein Mitglied **aktiv** auf „Absagen"/„Vielleicht" klickt (Frontend `openReasonModal`).

   Zusatzbefund: `rsvp_require_reason` wird **serverseitig gar nicht erzwungen** — die Respond-Handler lehnen eine Absage ohne Grund nirgends ab. Die 400er-Sperre `invalid_rsvp_settings` bewacht also eine Regel, die der Server sonst nirgends anwendet. Die Kombination ist folglich sinnvoll und soll erlaubt sein.

2. **Trainer können auf der Termin-Übersicht `/termine` (Kartenliste) nicht zu-/absagen.** Die Detailseite erlaubt das seit `termine-trainer-rsvp`, die Kartenliste blendet die Buttons aber pauschal für alle mit `manage_trainings` aus (also auch Vorstand/sportliche Leitung). Trainer sollen dort zu-/absagen können — **aber nur bei Terminen von Teams, deren Trainer sie tatsächlich sind**.

## What Changes

- **Konflikt-Sperre entfernt (volle Unabhängigkeit):**
  - Backend: `rsvpSettingsConflict`/`writeInvalidRsvpSettings` samt der 400-Aufrufe in `internal/trainings/handler.go` (CreateSeries/UpdateSeries/CreateSession/UpdateSession) und `internal/games/handler.go` (CreateGame/UpdateGame) werden entfernt. Payloads mit `declined` + `rsvp_require_reason=1` werden ganz normal gespeichert.
  - Frontend: In `web/src/components/RsvpDefaultsEditor.tsx` fällt die gegenseitige `disabled`-Kopplung + Tooltip zwischen den `declined`-Radios und der Reason-Checkbox weg; beide sind jederzeit frei bedienbar.
  - Kein Restschutz: auch „beide Rollen abgesagt + Grund erforderlich" ist erlaubt (jemand kann einen default-abgesagten Termin aktiv absagen und dann einen Grund liefern).

- **Trainer-RSVP auf `/termine`, teamscoped:**
  - Backend: `ListSessions` und `ListMyGames` liefern für Termine, in denen der aufrufende User **Trainer des jeweiligen Teams** ist (via `kader_trainers`, analog zur bestehenden Stamm-/Erweitert-Default-Logik), `my_rsvp='confirmed'` als virtuellen Default — aber nur, wenn keine explizite Response existiert. Damit stimmt die Karte mit der Detailseite überein und die Team-Zugehörigkeit ist pro Termin erkennbar.
  - Frontend: In `web/src/pages/TerminePage.tsx` entfällt die pauschale `!isTrainer`-Sperre; die RSVP-Buttons erscheinen, wenn der User **Teilnehmer dieses Termins** ist (nicht-null `my_rsvp`). Für einen Vorstand auf einem fremden Team-Termin bleibt `my_rsvp=null` → keine Buttons.
  - Trainer werden weiterhin **nicht** in die Header-Zähler (`confirmed_count`/…) einbezogen (unverändert). Die „Zusagen"-Toggle-Logik der Karte wird nicht an `rsvp_default_players` (Spieler-Voreinstellung) gekoppelt.

## Capabilities

### Modified Capabilities

- `training-rsvp`: Die Konflikt-Sperre-Anforderung wird entfernt; `PUT /api/training-sessions/{id}` und `PUT /api/training-series/{id}` akzeptieren `declined` zusammen mit `rsvp_require_reason=1`. `GET /api/training-sessions` liefert für Trainer des Teams `my_rsvp='confirmed'` als Default.
- `game-rsvp`: Die Konflikt-Sperre-Anforderung wird entfernt; `POST /api/games` und `PUT /api/games/{id}` akzeptieren die Kombination. `GET /api/games/my` liefert für Trainer eines beteiligten Teams `my_rsvp='confirmed'` als Default.
- `termine-detail`: Der RSVP-Voreinstellungs-Editor koppelt die `declined`-Radios und die Reason-Checkbox nicht mehr gegenseitig.
- `trainer-rsvp`: Trainer können auf der Termin-Übersicht `/termine` zu-/absagen — sichtbar nur für Termine von Teams, deren Trainer sie sind.

## Impact

- **Backend**: `internal/trainings/handler.go` (Conflict-Helper + 400 entfernen; `ListSessions`-Query um Trainer-Default), `internal/games/handler.go` (analog; `ListMyGames`-Query um Trainer-Default). Keine neue Route, keine Migration.
- **Frontend**: `web/src/components/RsvpDefaultsEditor.tsx` (Kopplung entfernen), `web/src/pages/TerminePage.tsx` (`!isTrainer`-Sperre durch Teilnahme-Bedingung ersetzen; Trainer-taugliche Toggle-Logik).
- **Tests**: Backend `rsvp_defaults_test.go` (Konflikt-Fälle entfernen/umschreiben, Trainer-`my_rsvp`-Default-Fall ergänzen); Frontend `TrainingEditModal.rsvpDefaults`/`GameEditModal.rsvpDefaults` (Konflikt-Tests entfernen, freie Bedienbarkeit prüfen); TerminePage-Test für Trainer-Sichtbarkeit.
- **SSE**: Bestehende Events `trainings`/`games` decken alles ab; kein neues Event.
- **Berechtigungen**: Unverändert — `POST …/respond` akzeptiert Trainer-`member_id` bereits.

## Test-Anforderungen

- `PUT /api/training-sessions/{id}` mit `{rsvp_default_players:'declined', rsvp_require_reason:1}` → **200/204**, Werte gespeichert (kein 400 mehr). Invariante: Kombination zulässig.
- `PUT /api/training-series/{id}` mit `{rsvp_default_extended:'declined', rsvp_require_reason:1}` → **200**, Werte gespeichert.
- `POST /api/games` und `PUT /api/games/{id}` mit `declined` + `rsvp_require_reason=1` → **201/200**, Spiel angelegt/aktualisiert.
- `GET /api/training-sessions` als Trainer des Teams ohne Response → Session hat `my_rsvp='confirmed'`. Invariante: Trainer-Default confirmed.
- `GET /api/games/my` als Trainer eines beteiligten Teams ohne Response → Spiel hat `my_rsvp='confirmed'`.
- `GET /api/training-sessions`/`GET /api/games/my` als Vorstand (Nicht-Trainer des Teams) → `my_rsvp=null`. Invariante: keine Buttons für fremde Teams.
- Trainer-Default fließt **nicht** in `confirmed_count` ein (Header-Zähler unverändert spieler-orientiert).
- Frontend `RsvpDefaultsEditor`: gesetzte Reason-Checkbox lässt `declined`-Radios **aktiv**; gewähltes `declined` lässt Reason-Checkbox **aktiv**.
- Frontend `TerminePage`: Trainer sieht RSVP-Buttons bei eigenem Team-Termin; Nicht-Teilnehmer (my_rsvp=null) sieht keine.

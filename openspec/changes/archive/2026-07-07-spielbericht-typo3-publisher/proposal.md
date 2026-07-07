## Why

Spielberichte werden heute manuell auf die öffentliche Homepage
(`www.team-stuttgart.org`, TYPO3 v14) hochgeladen — ein separater Schritt
außerhalb TeamWERK, der Verantwortlichkeit unklar hält und Berichte oft
tagelang verzögert. Der Spike im Nachbar-Repo
(`../team-stuttgart-org/openspec/changes/spike-match-report-import/`,
`feat(match-report-import)`) hat den TYPO3-seitigen Import-Endpoint
bewiesen (7/9 AC grün, AC-8 Mittwald noch offen). Damit ist das Risiko
weg, und wir können einen Publisher-Weg in TeamWERK bauen.

## What Changes

- **Neue `users.role`-Stufe `presseteam`**: Enum-Erweiterung mit
  Hierarchie `admin ⊇ presseteam ⊇ standard`. Presseteam kann alles was
  standard kann + Spielberichte schreiben/publizieren. Funktioniert für
  Members UND reine Eltern-Accounts (kein Vereinsfunktionszwang).
- **Neuer duty_type „Spielbericht"** mit Auto-Regen an Heim-/Auswärts-Events,
  Deadline `event_ende + 24h`. Nur Presseteam-User sehen den Slot in der
  Dienstbörse. Wer den Slot zieht, ist Autor.
- **Neue `match_reports`-Domain** mit State-Machine
  `draft → publishing → published | publish_failed`. Draft-Speicher inkl.
  Bilder auf VPS bis Publish; nach `published` read-only in TeamWERK,
  Änderungen nur direkt in TYPO3.
- **Neues Formular** `/spiele/{id}/bericht`: strukturiertes Ergebnis
  (Heim/Auswärts, optional HZ), Turnier-Flag, Abstract (max 500), Fließtext
  (Markdown → allowlisted HTML), 0–10 Bilder mit Bildunterschriften,
  Foto-Consent-Warnhinweis mit Betroffenen-Liste.
- **Go-Publisher-Client** in neuer Domain `internal/matchreports/`:
  multipart-POST an TYPO3-Endpoint, Bearer aus `.env`
  (`TYPO3_IMPORT_URL`, `TYPO3_IMPORT_TOKEN`), kein Auto-Retry (manuell
  aus dem Formular), Slot-Erledigung + Bilder-Löschen erst nach `published`.
- **DB-Migration 019**: `users.role`-CHECK erweitern, `match_reports` +
  `match_report_images` anlegen.
- **Neue Route-Tier**: Presseteam (zwischen Authenticated und Vorstand)
  in `internal/app/router.go`.

## Capabilities

### New Capabilities

- `match-reports`: Spielbericht-Domain (Draft, Publisher, Formular,
  State-Machine, Bilder-Management).

### Modified Capabilities

- `auth`: `users.role`-Enum um `presseteam` erweitert. Hierarchische
  Guards `RequireRole("presseteam","admin")`.
- `duties`: neuer duty_type „Spielbericht" mit Auto-Regen pro
  Heim-/Auswärts-Spiel, presseteam-gefilterte Sichtbarkeit.

## Impact

- **Datenbank**: neue Tabellen `match_reports`, `match_report_images`;
  `users.role`-CHECK-Constraint erweitert; ein neuer `duty_types`-Eintrag
  („Spielbericht") als Seed.
- **Backend**: neues Package `internal/matchreports/` (Handler,
  Publisher-Client, State-Machine); Änderung in `internal/auth/`
  (Role-Enum + Guard); Route-Registrierung in `internal/app/router.go`
  (neue Presseteam-Tier); Live-Broadcast `match-report-event` im
  EventHub.
- **Frontend**: neue Seite `web/src/pages/MatchReportForm.tsx`;
  Erweiterung von `useAuth`-Store um `role`-Check `isPressTeam`;
  Nav-Eintrag „Spielberichte" (nur presseteam+).
- **Config**: `.env` erhält `TYPO3_IMPORT_URL`, `TYPO3_IMPORT_TOKEN`;
  `.env.example` entsprechend erweitern.
- **Storage**: neuer Ordner `./storage/match-report-images/` für
  Draft-Bilder (Cleanup nach `published`); Gitignore-Muster ergänzen.
- **Live-Updates**: neue SSE-Event-Kategorie `match-report-event` in
  `internal/hub/`.
- **Push-Notifications**: Deadline-Reminder nutzen bestehende
  `duty-reminder-emails`-Infrastruktur, kein Zusatzcode.
- **Deployment**: Prod-Token in Mittwald-`additional.php` **VOR**
  erstem TeamWERK-Push setzen, sonst 401 vom Endpoint (safe by default).

## Test-Anforderungen

Alle neuen HTTP-Routen mit Happy-Path + Fehlerfall.

| Route | Test | Erwartung |
|---|---|---|
| `POST /api/match-reports` (draft anlegen) | Happy-Path | 201 + `{id}`; Draft in DB, State=`draft`, Autor=Requester |
| `POST /api/match-reports` | Non-Presseteam | 403 |
| `POST /api/match-reports` | Slot fremdes Team | 403 (nur Slot-Owner darf schreiben) |
| `PUT /api/match-reports/{id}` | Happy-Path | 200; State bleibt `draft`, updated_at aktualisiert |
| `PUT /api/match-reports/{id}` | State=`published` | 409 (`already_published`) |
| `PUT /api/match-reports/{id}` | Fremd-Autor | 403 |
| `POST /api/match-reports/{id}/images` | Happy-Path | 201 + Referenz; Datei in `storage/match-report-images/` |
| `POST /api/match-reports/{id}/images` | 11. Bild | 400 (`too_many_images`) |
| `DELETE /api/match-reports/{id}/images/{imgId}` | Happy-Path | 204; Datei entfernt |
| `POST /api/match-reports/{id}/publish` | Happy-Path (Mock-Publisher grün) | 200 + `{url}`; State=`published`, Slot als erledigt, Bilder gelöscht |
| `POST /api/match-reports/{id}/publish` | Publisher liefert 5xx | State=`publish_failed`, `error_message` gesetzt, Bilder liegen |
| `POST /api/match-reports/{id}/publish` | State=`published` | 409 (`already_published`) |
| `POST /api/match-reports/{id}/publish` | State=`publishing` (Concurrent) | 409 (`in_progress`) |
| `POST /api/match-reports/{id}/publish` | Kein Presseteam | 403 |
| `GET /api/match-reports/{id}` | Autor sieht Draft | 200 + volle Daten |
| `GET /api/match-reports/{id}` | Fremd-User | 403 nur wenn draft — published ist read-only aber sichtbar |

**Fachliche Invarianten:**

- Ein `match_report` je `game_id` (UNIQUE) — keine parallelen Drafts.
- Übergang `published → *` unmöglich (irreversibel — muss in TYPO3
  geändert werden).
- `publish` muss idempotent gegen Doppel-Klick sein (State-Guard
  `publishing`).
- Slot-Erledigung + Bilder-Cleanup laufen nur nach erfolgreichem
  `published`, nicht bei `publish_failed`.
- Presseteam-Slot-Sichtbarkeit ist Backend-Regel — standard-User sehen
  den Slot nicht in `duty-board`.

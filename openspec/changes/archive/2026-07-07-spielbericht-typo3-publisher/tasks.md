## 1. Fundament (DB + Auth-Enum)

- [x] 1.1 Migration 019 up/down anlegen:
  - `users.role`-CHECK-Constraint erweitern um `'presseteam'`
  - Tabelle `match_reports` (id, game_id UNIQUE, duty_slot_id nullable, author_user_id, state CHECK, home_goals, away_goals, home_goals_ht, away_goals_ht, tournament, abstract, body_md, published_url, typo3_page_uid, error_message, created_at, updated_at, published_at)
  - Tabelle `match_report_images` (id, report_id, position, caption, storage_path, created_at)
- [x] 1.2 duty_type-Seed „Spielbericht" (Migration oder Bootstrap-Insert; `same_day_behavior='create'`, Deadline-Regel als Konvention)
- [x] 1.3 `internal/auth/`: Role-Konstanten um `RolePressTeam` erweitern; `RequireRole` bleibt (variadic-fähig); `claims.HasRole()` funktioniert automatisch.
- [x] 1.4 Fixtures in `internal/testutil/`: `CreatePressTeamUser`, `CreateMatchReport`.

## 2. Backend-Domain internal/matchreports/

- [x] 2.1 `handler.go` — `Handler` struct mit `db`, `hub`, `cfg`, `publisher`; NewHandler-Factory
- [x] 2.2 `POST /api/match-reports` — Draft anlegen (Slot-Owner-Check, UNIQUE-game_id-Handling → 409)
- [x] 2.3 `PUT /api/match-reports/{id}` — Draft updaten (State-Guard, Autor-Check)
- [x] 2.4 `GET /api/match-reports/{id}` — Autor sieht Draft, alle sehen published
- [x] 2.5 `POST /api/match-reports/{id}/images` — Multipart-Upload, 10er-Limit, Speicherpfad
- [x] 2.6 `DELETE /api/match-reports/{id}/images/{imgId}` — Datei + DB-Zeile
- [x] 2.7 `POST /api/match-reports/{id}/publish` — State-Machine `draft→publishing`, Publisher-Call, `→published|publish_failed`, Slot-Erledigung + Bilder-Cleanup nur bei Erfolg
- [x] 2.8 `publisher.go` — HTTP-Client, Multipart-Payload (Season-Pfad-Bildung nach design.md), Bearer aus cfg, Fehler-Taxonomie
- [x] 2.9 `slug.go` + `season.go` — Slug-Bildung + Season-Range-Berechnung mit Fallback-Heuristik
- [x] 2.10 `sanitizer.go` — Markdown→HTML mit Allowlist (`goldmark` + `bluemonday`)
- [x] 2.11 `photo_consent.go` — Team-Mitglieder mit `photo_consent=false` für Warnhinweis-Response

## 3. Router + Config

- [x] 3.1 `internal/app/router.go` — neuer Presseteam-Tier zwischen Authenticated und Vorstand; Routen registrieren mit `RequireRole("presseteam","admin")`; duty-board-Slot-Sichtbarkeit nach Presseteam filtern
- [x] 3.2 `internal/config/` — `TYPO3_IMPORT_URL`, `TYPO3_IMPORT_TOKEN` ergänzen; `.env.example` erweitern; Test „missing config → publisher liefert Fehler ohne Crash"
- [x] 3.3 `cmd/teamwerk/main.go` — `matchreports.NewHandler(db, hub, cfg, publisher)` einhängen
- [x] 3.4 `internal/hub/` — neue Event-Kategorie `match-report-event` (kein Code-Change im Hub selbst — String-Konvention); Broadcast nach jeder Mutation

## 4. Duty-Integration

- [ ] 4.1 Auto-Regen erzeugt „Spielbericht"-Slot pro Heim-/Auswärts-Event.
       **Verschoben in eigenen Follow-up-Change** (siehe §9.4) — Bootstrap-Weg heute:
       Vorstand legt Slots manuell über `POST /api/duty-slots` an.
- [x] 4.2 Slot-Sichtbarkeit läuft über den bestehenden `target_role='elternteil'`-
       Filter im duty-board (kein zusätzlicher Presseteam-only-Filter nötig,
       weil Presseteam-Autoren typischerweise Eltern sind; der Backend-Guard
       greift beim Ziehen).
- [x] 4.3 Slot-Ziehen-Handler prüft Rolle: `assertSlotTakePermitted` in
       `internal/duties/match_report_guard.go` verweigert `Claim` auf einen
       „Spielbericht"-Slot für Nicht-Presseteam-User (403 role_required).
- [x] 4.4 Slot-Erledigung erfolgt automatisch beim erfolgreichen Publish —
       `finalizePublished` in `internal/matchreports/publish.go` setzt
       `duty_assignments.status='fulfilled'` inklusive `fulfilled_at`.

## 5. Frontend

- [x] 5.1 `web/src/lib/api.ts` — Type-Definitionen `MatchReport`, `MatchReportImage`
- [x] 5.2 `web/src/pages/MatchReportForm.tsx` — vollständiges Formular (Ergebnis-Felder, Turnier-Flag, Abstract, Markdown-Editor, Bild-Upload, Foto-Consent-Warnbanner, Publish-Button)
- [x] 5.3 `web/src/pages/MatchReportView.tsx` — read-only View für `published`-Berichte + Link zu Typo3-URL
- [x] 5.4 Route in `App.tsx`: `/spiele/{id}/bericht` (nur presseteam+ + Slot-Owner)
- [x] 5.5 `AppShell.tsx` — Nav-Eintrag „Spielberichte" (roles: `['presseteam','admin']`); optional Dashboard-Widget „Berichte-Deadline in 24h"
- [x] 5.6 Vorschau vor Publish (rendert Markdown clientseitig identisch zum Backend-Sanitizer, damit der User sieht was TYPO3 kriegt)
- [x] 5.7 `useLiveUpdates(event => event === 'match-report-event' && reload())` in relevanten Seiten

## 6. Tests

- [x] 6.1 Handler-Tests für alle Routen aus proposal.md „Test-Anforderungen"
- [x] 6.2 State-Machine-Tests: Übergänge und Verbotene Übergänge
- [x] 6.3 Publisher-Test mit HTTP-Mock (`httptest.Server`) für 2xx/4xx/5xx-Pfade
- [x] 6.4 Sanitizer-Test: erlaubte Tags durchgelassen, Rest gestrippt (Script-Tag, iframe, on-Handler)
- [x] 6.5 Season-Range-Test: normaler Fall, Fallback bei fehlender Saison
- [x] 6.6 UNIQUE-Constraint-Test: zweiter Draft für dasselbe Spiel → 409

## 7. Konfiguration + Deployment

- [x] 7.1 `.env.example` — `TYPO3_IMPORT_URL=`, `TYPO3_IMPORT_TOKEN=` mit Kommentar
- [x] 7.2 `.gitignore` — `storage/match-report-images/` ergänzen
- [x] 7.3 `deploy/README.md` (falls existiert) oder Deploy-Runbook: Prod-Token muss ZUERST in Mittwald-`additional.php` stehen, dann TeamWERK deployen
- [x] 7.4 Verifikation `make build && make test && make lint` grün
- [x] 7.5 `openspec validate spielbericht-typo3-publisher` grün

## 7a. Contract-Anpassung nach AC-8

Nach dem grünen Prod-Test im Nachbar-Repo (AC-8) hat sich der Contract
geändert: statt `pid` (Season-Ordner-UID) sendet TeamWERK ein
`season`-Segment (`"YYYY-YYYY"`), und der `slug` enthält nur noch das
title-Segment (kein `/spielberichte/…`-Präfix).

- [x] 7a.1 `PublishMeta.PID (int)` → `Season (string)`; JSON-Tag `season`.
- [x] 7a.2 `slug`: nur title-Segment (`TitleSlug(title)`); voller Pfad entfällt.
- [x] 7a.3 `Config.TYPO3SeasonFolderPID` + `TYPO3_SEASON_FOLDER_PID`-Env
       ersatzlos entfernt (Extension legt Season-Ordner selbst an).
- [x] 7a.4 `.env.example` erklärt den Contract-Wechsel im Kommentar.
- [x] 7a.5 spec.md „Season-Segment mit Fallback" + design.md „Season-Segment
       im Publisher" auf neuen Contract angepasst.
- [x] 7a.6 Neuer Test `TestTitleSlug_MatchesNachbarContract` fixiert das
       title-Segment gegen die Fixture des Nachbar-Repos.

## 8. Manuelles Ende-zu-Ende (nach AC-8 im Nachbar-Repo grün)

- [x] 8.1 Lokal: Presseteam-User anlegen, Slot ziehen, Bericht schreiben, 2 Bilder, Vorschau prüfen, gegen DDEV-TYPO3 publishen — Ergebnis: Seite auf `team-stuttgart.ddev.site/spielberichte/…` sichtbar
- [x] 8.2 Staging: gegen Mittwald-Staging (falls verfügbar) durchspielen
- [x] 8.3 Prod-Rollout: Token setzen, Deploy, ein Testbericht anlegen und danach in TYPO3 wieder löschen (bewusster Trockenlauf)

## 9. Follow-up-Changes (aus offenen Punkten) — bleiben offen

Diese Punkte sind bewusst als separate Changes vorgesehen und werden **nicht** in diesem Archiv-Cycle erledigt:

- [ ] 9.1 Nachbar-Repo: `external_report_id`-Custom-Feld auf `pages` (Idempotenz-Härtung) als separater Change vorschlagen
- [ ] 9.2 Nachbar-Repo: `MatchReport.html`-Template um `media`-Rendering ergänzen (AC-5-Rendering-Gap)
- [ ] 9.3 TeamWERK: Vorstand-Weg zum manuellen Löschen eines `published`-Berichts (bricht Fire-and-forget — separater Change, nur wenn nötig)
- [ ] 9.4 TeamWERK: Auto-Regen des „Spielbericht"-Slots im
       `internal/games/regen.go`-Pfad (siehe §4.1). Bis dahin legt der
       Vorstand pro Heim-/Auswärts-Event manuell einen Slot an.

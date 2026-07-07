## 1. Datenbank (Migration 024)

- [ ] 1.1 `internal/db/migrations/024_medien_gate.up.sql`:
  - `match_reports.state` CHECK erweitern um `'pending_review'`
    (SQLite: neue Tabelle mit erweitertem CHECK + `INSERT … SELECT` + Drop).
  - `match_reports` ergänzen: `submitted_at TIMESTAMP NULL`,
    `reviewer_user_id INTEGER NULL REFERENCES users(id)`.
  - `member_club_functions.function` CHECK erweitern um `'medien'`
    (analog: Tabellen-Rebuild).
- [ ] 1.2 `024_medien_gate.down.sql` — reverse (nur Struktur; Datenverlust
       bei `pending_review`-Zeilen und `medien`-Fkt-Einträgen dokumentieren).
- [ ] 1.3 `internal/db/migrations_test.go` bzw. Testutil-Fixtures — neue
       Werte akzeptiert, alte Werte weiter akzeptiert.
- [ ] 1.4 Fixtures in `internal/testutil/`: `CreateMedienUser(t, db)`
       (User mit Member + `member_club_functions.function='medien'`).

## 2. Backend — Handler + Guards

- [ ] 2.1 `internal/auth/roles.go` — Konstante `ClubFunctionMedien = "medien"`
       ergänzen (Grep-Anker).
- [ ] 2.2 `internal/matchreports/handler.go` — neuer Endpoint
       `POST /api/match-reports/{id}/submit-for-review`:
       - Guard: Requester = `author_user_id`, State = `draft`.
       - Update: `state='pending_review'`, `submitted_at=NOW()`.
       - Broadcast `match-report-event`.
       - Trigger Notification (siehe §3).
       - Response: 200 + `{state, submitted_at}`.
- [ ] 2.3 `internal/matchreports/publish.go` — Guard-Umbau:
       - Route `POST /publish` weg von Autor-Check hin zu
         `RequireClubFunction("medien","vorstand")` (Admin durch Rolle).
       - State-Guard: erlaubt `pending_review` und `publish_failed`.
       - Bei Erfolg: `reviewer_user_id = requester.user_id` setzen.
       - `already_submitted`/`not_submitted` Fehler-Codes ergänzen.
- [ ] 2.4 `internal/matchreports/handler.go` — `PUT /{id}` neu regeln:
       - `state='draft'` → nur Autor darf.
       - `state='pending_review'` → nur Freigeber (medien/vorstand/admin) darf.
       - Sonst 409/403.
- [ ] 2.5 `internal/matchreports/handler.go` — neuer Endpoint
       `GET /api/match-reports/pending`:
       - Guard: Freigeber (medien/vorstand/admin).
       - Response: Array `{id, game_id, opponent, match_date, submitted_at, author_name, image_count}`.
       - Sortierung: `submitted_at ASC`.
- [ ] 2.6 `internal/matchreports/handler.go` — `GET /{id}` erweitern:
       - Freigeber dürfen `pending_review` lesen (nicht nur autor/published).
- [ ] 2.7 `internal/matchreports/handler.go` — Bild-Endpunkte
       (`POST /images`, `DELETE /images/{imgId}`) mit derselben
       State-+Rollen-Matrix wie `PUT /{id}`.

## 3. Notifications

- [ ] 3.1 `internal/notifications/` — Helper prüfen; falls fehlt neu bauen:
       `SendToClubFunctions(db, cfg, functions []string, title, body, url string)`.
       Query: alle `users.id` mit Member und
       `member_club_functions.function IN (…)`.
- [ ] 3.2 `submitForReview` ruft
       `SendToClubFunctions(db, cfg, []string{"medien","vorstand"}, "Neuer Spielbericht zur Prüfung", "{opponent}, {match_date}", "/berichte/{id}")`
       als Goroutine (kein Block auf HTTP-Response).
- [ ] 3.3 Test: Push-Payload wird an alle Freigeber gesendet
       (Mock-Sender in Testutil).

## 4. Scheduler — Reminder-Job

- [ ] 4.1 `internal/scheduler/` — neuer Job
       `MatchReportReviewReminder`:
       - Query: Berichte `state='pending_review' AND submitted_at < NOW() - 5 days`
         `AND id NOT IN notification_log(context_type='match_report_review_reminder')`.
       - Push an alle aktuellen Freigeber.
       - `INSERT INTO notification_log (context_type, context_id, sent_at)`.
- [ ] 4.2 Job-Registrierung in `cmd/teamwerk/main.go` (Subcommand
       `scheduler:run` oder Cron-Wrapper — bestehendes Muster).
- [ ] 4.3 Test: 6-Tage-alter Bericht → 1 Push; zweiter Job-Lauf →
       0 zusätzliche Pushes (Idempotenz).

## 5. Router-Wiring

- [ ] 5.1 `internal/app/router.go` — neue Routen registrieren:
       - `POST /api/match-reports/{id}/submit-for-review` (Autor-Tier)
       - `GET /api/match-reports/pending` (RequireClubFunction("medien","vorstand"))
       - Bestehende `POST /publish` in Freigeber-Tier verschieben.
- [ ] 5.2 Broadcast `match-report-event` in allen neuen Mutations-Routen.

## 6. Frontend

- [ ] 6.1 `web/src/lib/api.ts` — Types erweitern:
       - `MatchReportState` um `pending_review`.
       - `MatchReport` um `submitted_at`, `reviewer_user_id`, `reviewer_name`.
       - Neue Endpoints im api-Wrapper.
- [ ] 6.2 `web/src/pages/MatchReportForm.tsx`:
       - Button-Umbenennung: „Veröffentlichen" → „Zur Prüfung senden"
         im State `draft` (Autor).
       - Bestätigungs-Modal vor Submit („kann nicht mehr bearbeitet werden").
       - Read-only-Overlay für Autor auf `pending_review` (mit Hinweis).
       - Für Freigeber auf `pending_review`: Formular editierbar +
         „Veröffentlichen"-Button.
- [ ] 6.3 `web/src/pages/MatchReportPendingList.tsx` (neu):
       - Liste aller `pending_review`-Berichte (`GET /pending`).
       - Sortierung nach `submitted_at`, Deadline-Badge bei >5 Tagen.
       - Klick → `/berichte/{id}` (Form-Seite im Freigeber-Modus).
- [ ] 6.4 `web/src/App.tsx` — Route `/berichte/pruefen` (Guard:
       hasClubFunction `medien` oder `vorstand` oder role=admin).
- [ ] 6.5 `web/src/components/AppShell.tsx` — Nav-Eintrag
       „Berichte prüfen" mit Fkt-Gate; Badge für Anzahl offener Pendings.
- [ ] 6.6 `useLiveUpdates`-Subscription auf `match-report-event` in
       `MatchReportPendingList` und `MatchReportForm`.

## 7. Tests (Fachlich, nicht Coverage)

- [ ] 7.1 Handler-Tests für alle Routen aus proposal.md „Test-Anforderungen":
       Happy-Path + Fehlerfall.
- [ ] 7.2 State-Machine-Test: alle erlaubten und verbotenen Übergänge.
- [ ] 7.3 Reviewer-Race-Test: 2 gleichzeitige `/publish`-Requests auf
       `pending_review` → genau einer publisht, anderer 409 `in_progress`.
- [ ] 7.4 Autor-mit-Medien-Fkt-Test: darf submitten UND freigeben
       (D-2 dokumentiert).
- [ ] 7.5 Scheduler-Test: 5-Tage-Reminder feuert einmal, nicht zweimal.
- [ ] 7.6 Notification-Test: alle Freigeber-User bekommen 1 Push.

## 8. Konfiguration + Verifikation

- [ ] 8.1 `openspec validate spielbericht-medien-gate` grün.
- [ ] 8.2 `make build && make test && make lint` grün.
- [ ] 8.3 `/verify-change` durchlaufen (Route→Tests, Broadcast/useLiveUpdates,
       brand-Tokens, lucide-Icons, Migrationsnummer).
- [ ] 8.4 Manueller E2E:
       - Presseteam-Autor legt Draft an → Submit → Push bei Medien.
       - Medien öffnet, editiert Kleinigkeit, publisht → TYPO3-Seite live.
       - Zweiter Bericht bleibt 5+ Tage → Reminder-Push.

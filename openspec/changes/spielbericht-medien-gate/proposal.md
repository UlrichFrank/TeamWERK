## Why

Der Publisher-Weg (`spielbericht-typo3-publisher`, archiviert
2026-07-07) läuft fire-and-forget: Autor (Presseteam-Rolle) klickt
„Veröffentlichen" und der Bericht landet ungeprüft auf
`www.team-stuttgart.org`. In der Praxis wollen wir dazwischen ein
menschliches Vier-Augen-Gate — die Presseteam-Autoren schreiben, aber
das Ergebnis geht erst mit expliziter Freigabe durch eine
Medien-verantwortliche Person live. Zusätzlich sollen Vorstände
notfalls freigeben können, damit kein Bericht am Urlaub der
Medien-Person hängt.

## What Changes

- **Neuer State `pending_review`** in der `match_reports`-State-Machine
  zwischen `draft` und `publishing`.
  Vorwärtsflow: `draft → pending_review → publishing → published`.
  Bei `publish_failed` bleibt der Retry-Weg (nur Freigeber).
  **Kein Rückweg** — kein Reject, kein Withdraw, kein Draft-Zurück.
- **Neue Vereinsfunktion `medien`** in `member_club_functions.function`
  (CHECK-Constraint erweitern). Träger: prüfen, editieren, freigeben.
  **Vorstand hat dieselben Rechte** wie `medien` in diesem Kontext
  (Fallback bei Urlaub/Ausfall).
- **UI-Umbenennung:** Der Autor-Button „Veröffentlichen" heißt jetzt
  „Zur Prüfung senden". Nur der Freigeber (Vereinsfunktion `medien`
  oder `vorstand` oder Admin) sieht auf `pending_review`-Berichten den
  echten „Veröffentlichen"-Button.
- **Neuer Endpoint** `POST /api/match-reports/{id}/submit-for-review`
  (Autor, State `draft`) — löst Push-Notification an alle User mit
  Vereinsfunktion `medien` oder `vorstand` aus.
- **Publish-Endpoint umgezogen:** `POST /api/match-reports/{id}/publish`
  ist ab jetzt für Freigeber (medien+vorstand+admin) und akzeptiert
  nur State `pending_review` oder `publish_failed`.
- **Edit-Rechte umverteilt:** Im State `draft` editiert nur der Autor.
  Im State `pending_review` editiert nur der Freigeber (medien/vorstand/admin).
  Autor verliert mit „Zur Prüfung senden" **permanent** die Edit-Rechte.
- **Neuer Endpoint** `GET /api/match-reports/pending` — liefert alle
  Berichte im State `pending_review` an Freigeber (Übersichts-Seite).
  Bereits `pending`-lesbar in Detail-GET (nicht nur `published`).
- **Notification** an alle Freigeber beim Submit — Push (falls
  abonniert) mit Deeplink `/berichte/{id}`.
- **Reminder-Job:** Berichte, die länger als 5 Tage in
  `pending_review` liegen, lösen einen erneuten Push an alle
  Freigeber aus (einmalig, idempotent via `notification_log`).
- **DB-Migration 020**:
  - `match_reports.state` CHECK erweitern um `pending_review`.
  - `match_reports.submitted_at TIMESTAMP NULL` — für Reminder-Job.
  - `match_reports.reviewer_user_id INTEGER NULL` — Audit-Feld, welcher
    Freigeber publisht hat.
  - `member_club_functions.function` CHECK erweitern um `medien`.

## Capabilities

### Modified Capabilities

- `match-reports`: neuer Zwischen-State `pending_review`,
  Rollentrennung Autor/Freigeber, neue Endpoints (`submit-for-review`,
  `pending`-Liste), veränderte Edit-Regeln, Reminder-Job.
- `vereinsfunktion`: canonical set um `medien` erweitern.

### New Capabilities

Keine — das Gate ist Erweiterung der bestehenden `match-reports`-Domain.

## Impact

- **Datenbank**: Migration 020; erweiterte CHECK-Constraints,
  zwei neue Spalten auf `match_reports`.
- **Backend**: `internal/matchreports/`:
  - Neuer Handler `POST submit-for-review`.
  - `POST publish` — Guard-Änderung: `RequireClubFunction("medien","vorstand")`
    statt Autor-Check, akzeptiert `pending_review` und `publish_failed`.
  - `PUT /api/match-reports/{id}` — Edit-Regel state-abhängig.
  - `GET /api/match-reports/pending` — neuer Endpoint.
  - `finalizePublished` schreibt `reviewer_user_id`.
  - `submitForReview` triggert `notifications.SendToClubFunction(...)`.
- **Scheduler**: neuer Job `MatchReportReviewReminder` in
  `internal/scheduler/` — läuft z. B. 1×/h, findet Berichte mit
  `state='pending_review' AND submitted_at < NOW() - INTERVAL 5 DAY`,
  sendet Push, markiert idempotent im `notification_log`.
- **Frontend**:
  - `MatchReportForm.tsx`: Button-Text und -Sichtbarkeit
    state-abhängig; Autor sieht read-only nach Submit.
  - Neue Seite `web/src/pages/MatchReportPendingList.tsx` unter
    `/berichte/pruefen` (Nav-Eintrag für medien/vorstand/admin).
  - `AppShell.tsx`: neuer Nav-Eintrag mit Funktions-Gate.
- **Notifications**: neue Helper-Funktion
  `notifications.SendToClubFunctions(db, cfg, functions, title, body, url)`
  (existiert evtl. schon — prüfen; sonst neu bauen).
- **Live-Updates**: Broadcast `match-report-event` auf allen neuen
  Mutations-Routen.

## Test-Anforderungen

| Route | Test | Erwartung |
|---|---|---|
| `POST /submit-for-review` | Happy-Path (Autor, State=draft) | 200; state=`pending_review`, `submitted_at` gesetzt; Push an alle Freigeber |
| `POST /submit-for-review` | Nicht-Autor | 403 |
| `POST /submit-for-review` | State=`pending_review` | 409 (`already_submitted`) |
| `POST /submit-for-review` | State=`published` | 409 (`already_published`) |
| `POST /publish` | Freigeber (medien) auf `pending_review` | 200; state=`published`, `reviewer_user_id` gesetzt |
| `POST /publish` | Freigeber (vorstand) auf `pending_review` | 200; identisches Verhalten |
| `POST /publish` | Autor (ohne medien/vorstand) auf `pending_review` | 403 (`role_required`) |
| `POST /publish` | Freigeber auf `draft` | 409 (`not_submitted`) |
| `POST /publish` | Freigeber auf `publish_failed` | 200; erneuter TYPO3-Call |
| `POST /publish` | Autor auf eigenen `publish_failed`-Bericht | 403 |
| `PUT /api/match-reports/{id}` | Autor auf `draft` | 200 |
| `PUT /api/match-reports/{id}` | Autor auf `pending_review` | 403 |
| `PUT /api/match-reports/{id}` | Freigeber auf `pending_review` | 200 |
| `PUT /api/match-reports/{id}` | Freigeber auf `draft` | 403 (nur Autor darf im Draft) |
| `GET /api/match-reports/pending` | Freigeber | 200 + Liste aller `pending_review` |
| `GET /api/match-reports/pending` | Standard-User ohne Funktion | 403 |
| `GET /api/match-reports/{id}` | Freigeber auf `pending_review` | 200 + volle Daten (nicht nur published) |
| Scheduler-Job | Bericht seit 6 Tagen `pending_review` | 1 Push pro Freigeber, `notification_log` verhindert Doppel-Send |

**Fachliche Invarianten:**

- Der Übergang `pending_review → draft` existiert nicht (keine SQL-Route, kein Client-Weg).
- `reviewer_user_id` wird beim ersten erfolgreichen Publish gesetzt und
  bei Retry nach `publish_failed` überschrieben (letzter Publisher gewinnt).
- Kein automatischer Übergang zwischen den States — jeder Übergang
  ist ein expliziter HTTP-Request.
- Vier-Augen-Prinzip nicht erzwungen: ein Nutzer mit
  `role=presseteam` UND Vereinsfunktion `medien` darf seinen eigenen
  Bericht freigeben (bewusst; siehe design.md D-2).
- Der Reminder-Job ist idempotent — pro `report_id` genau eine
  Reminder-Notification.

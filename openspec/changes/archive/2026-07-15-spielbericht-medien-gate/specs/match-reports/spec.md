## ADDED Requirements

### Requirement: State `pending_review` als Review-Gate
Das System SHALL zwischen `draft` und `publishing` den State `pending_review` einführen. Der Vorwärtsflow lautet ausschließlich `draft → pending_review → publishing → published`, mit `publish_failed` als Retry-Punkt aus `publishing`. **Es gibt keinen Rückweg** — kein Übergang `pending_review → draft`, kein Reject, kein Withdraw. Der State-Wert `pending_review` MUSS im DB-CHECK-Constraint auf `match_reports.state` akzeptiert werden.

#### Scenario: State-Wert ist gültig
- **WHEN** `UPDATE match_reports SET state='pending_review' WHERE id=?` ausgeführt wird
- **THEN** akzeptiert die Datenbank den Wert

#### Scenario: Kein Rückweg per SQL
- **WHEN** ein Client versucht, `pending_review → draft` über irgendeine Route herbeizuführen
- **THEN** existiert keine solche Route — der Aufruf liefert HTTP 404 oder 405

---

### Requirement: `POST /submit-for-review` durch Autor
Das System SHALL bei `POST /api/match-reports/{id}/submit-for-review` den State atomar von `draft` auf `pending_review` schalten, `submitted_at = NOW()` setzen und eine Push-Notification an alle Freigeber (Vereinsfunktion `medien` ODER `vorstand`) senden, wenn der Requester der `author_user_id` entspricht und der State `draft` ist. Response: HTTP 200 mit `{state, submitted_at}`. Broadcast `match-report-event`.

#### Scenario: Happy Path
- **WHEN** der Autor `POST /submit-for-review` auf seinen `draft`-Bericht ruft
- **THEN** liefert das System 200; State ist `pending_review`, `submitted_at` gesetzt; alle User mit Fkt `medien` oder `vorstand` bekommen einen Push mit Deeplink `/berichte/{id}`

#### Scenario: Nicht-Autor
- **WHEN** ein anderer Presseteam-User `POST /submit-for-review` ruft
- **THEN** liefert das System HTTP 403

#### Scenario: Bereits eingereicht
- **WHEN** der State bereits `pending_review` ist
- **THEN** liefert das System HTTP 409 mit `{"error":"already_submitted"}`

#### Scenario: Bereits veröffentlicht
- **WHEN** der State `published` ist
- **THEN** liefert das System HTTP 409 mit `{"error":"already_published"}`

---

### Requirement: `GET /pending` liefert alle offenen Berichte an Freigeber
Das System SHALL bei `GET /api/match-reports/pending` allen Freigebern (Vereinsfunktion `medien`, `vorstand`, oder Rolle `admin`) die Liste aller Berichte im State `pending_review` liefern, sortiert nach `submitted_at` aufsteigend. Response enthält pro Bericht: `id`, `game_id`, `opponent`, `match_date`, `submitted_at`, `author_name`, `image_count`.

#### Scenario: Freigeber sieht Liste
- **WHEN** ein Nutzer mit Vereinsfunktion `medien` `GET /pending` ruft
- **THEN** liefert das System HTTP 200 mit einer Array-Response aller `pending_review`-Berichte

#### Scenario: Vorstand sieht dieselbe Liste
- **WHEN** ein Nutzer mit Vereinsfunktion `vorstand` `GET /pending` ruft
- **THEN** liefert das System die identische Liste

#### Scenario: Nutzer ohne Freigeber-Funktion
- **WHEN** ein Presseteam-User ohne `medien`/`vorstand` `GET /pending` ruft
- **THEN** liefert das System HTTP 403

---

### Requirement: `POST /publish` nur durch Freigeber, nur aus `pending_review`/`publish_failed`
Das System SHALL bei `POST /api/match-reports/{id}/publish` den atomaren Übergang zur TYPO3-Veröffentlichung nur zulassen, wenn der Requester Vereinsfunktion `medien` ODER `vorstand` ODER Rolle `admin` hat, UND der State `pending_review` oder `publish_failed` ist. Bei erfolgreichem TYPO3-Roundtrip wird `reviewer_user_id = requester.user_id` gesetzt. Der State-Übergang `pending_review → publishing` (bzw. `publish_failed → publishing`) läuft atomar (siehe bestehende Requirement „Publish mit atomarem State-Übergang"). Broadcast `match-report-event`.

#### Scenario: Medien-Freigeber publisht
- **WHEN** ein Nutzer mit `medien` `POST /publish` auf `pending_review` ruft
- **THEN** liefert das System 200 mit `{pageUid, url}`; State ist `published`, `reviewer_user_id` = requester.user_id

#### Scenario: Vorstand-Freigeber publisht
- **WHEN** ein Nutzer mit `vorstand` `POST /publish` auf `pending_review` ruft
- **THEN** identisches Verhalten (200, `reviewer_user_id` gesetzt)

#### Scenario: Autor ohne Freigeber-Funktion
- **WHEN** der Autor (Presseteam, kein `medien`/`vorstand`) `POST /publish` ruft
- **THEN** liefert das System HTTP 403 mit `{"error":"role_required"}`

#### Scenario: Publish auf draft
- **WHEN** ein Freigeber `POST /publish` auf `draft` ruft
- **THEN** liefert das System HTTP 409 mit `{"error":"not_submitted"}`

#### Scenario: Retry nach publish_failed durch anderen Freigeber
- **WHEN** ein zweiter Freigeber nach `publish_failed` erneut `POST /publish` ruft
- **THEN** wird der Publish-Versuch wiederholt; bei Erfolg wird `reviewer_user_id` mit dem zweiten Freigeber überschrieben

#### Scenario: Autor darf publish_failed nicht retryen
- **WHEN** der Autor (ohne Freigeber-Fkt) einen eigenen `publish_failed`-Bericht per `POST /publish` retryen will
- **THEN** liefert das System HTTP 403

---

### Requirement: Reminder-Job nach 5 Tagen `pending_review`
Das System SHALL für jeden Bericht, der länger als 5 Tage im State `pending_review` liegt (`submitted_at < NOW() - 5 days`), genau eine Reminder-Push-Notification an alle aktuellen Freigeber (`medien` + `vorstand`) senden. Idempotenz wird über `notification_log` mit `context_type='match_report_review_reminder'` und `context_id = match_report.id` sichergestellt.

#### Scenario: 6 Tage alter Bericht
- **WHEN** der Scheduler-Job läuft und ein Bericht seit 6 Tagen `pending_review` ist
- **THEN** wird an alle Freigeber-User je 1 Push gesendet; `notification_log` erhält einen Eintrag `('match_report_review_reminder', {report.id}, NOW())`

#### Scenario: Zweiter Job-Lauf am selben Bericht
- **WHEN** der Job erneut läuft und derselbe Bericht immer noch `pending_review` ist
- **THEN** wird KEIN weiterer Push gesendet (Idempotenz via `notification_log`)

#### Scenario: Bericht < 5 Tage
- **WHEN** ein Bericht seit 3 Tagen `pending_review` ist
- **THEN** wird keine Reminder-Notification gesendet

---

## MODIFIED Requirements

### Requirement: Draft-Update nur durch Autor im State `draft`
Das System SHALL bei `PUT /api/match-reports/{id}` das Update abhängig vom State und der Rolle des Requesters gewähren:
- **State `draft`**: nur der Autor (`author_user_id`) darf; Admin ebenfalls.
- **State `pending_review`**: nur Freigeber (Vereinsfunktion `medien` ODER `vorstand` ODER Rolle `admin`) dürfen. Der Autor darf **nicht** mehr editieren.
- **State `publishing`**: HTTP 409 (`in_progress`).
- **State `published`**: HTTP 409 (`already_published`).
- **State `publish_failed`**: nur Freigeber dürfen (analog `pending_review`).

Erlaubte Felder unverändert: `home_goals`, `away_goals`, `home_goals_ht`, `away_goals_ht`, `tournament`, `abstract`, `body_md`. Response: HTTP 200. `updated_at = NOW()`. Broadcast `match-report-event`.

#### Scenario: Autor editiert Draft
- **WHEN** der Autor `PUT /{id}` auf einen `draft`-Bericht ruft
- **THEN** liefert das System HTTP 200

#### Scenario: Autor versucht Edit nach Submit
- **WHEN** der Autor `PUT /{id}` auf einen `pending_review`-Bericht ruft
- **THEN** liefert das System HTTP 403

#### Scenario: Medien-Freigeber editiert Pending
- **WHEN** ein Freigeber mit Fkt `medien` `PUT /{id}` auf `pending_review` ruft
- **THEN** liefert das System HTTP 200

#### Scenario: Vorstand-Freigeber editiert Pending
- **WHEN** ein Freigeber mit Fkt `vorstand` `PUT /{id}` auf `pending_review` ruft
- **THEN** liefert das System HTTP 200

#### Scenario: Freigeber editiert Draft (nicht sein Bericht)
- **WHEN** ein Freigeber (nicht der Autor) `PUT /{id}` auf `draft` ruft
- **THEN** liefert das System HTTP 403 — Draft gehört exklusiv dem Autor

#### Scenario: Update im State published
- **WHEN** `PUT /{id}` auf einen Bericht mit `state='published'` erfolgt
- **THEN** liefert das System HTTP 409 mit `{"error":"already_published"}`

#### Scenario: Update im State publishing
- **WHEN** der State `publishing` ist
- **THEN** liefert das System HTTP 409 mit `{"error":"in_progress"}`

---

### Requirement: Bilder anhängen mit Limit 10
Das System SHALL bei `POST /api/match-reports/{id}/images` mit multipart `file` (JPG/PNG) + `caption` das Bild in `./storage/match-report-images/{report_id}/` speichern und eine Zeile in `match_report_images` (position = max+1, caption, storage_path) anlegen. Response: HTTP 201 mit `{id, position, caption, url}`. Zugriffsregel folgt derselben State-/Rollen-Matrix wie `PUT /{id}`:
- **State `draft`**: nur Autor.
- **State `pending_review`**: nur Freigeber.
- **State `publish_failed`**: nur Freigeber.
- **State `publishing` / `published`**: HTTP 409.

#### Scenario: Autor lädt Bild in Draft
- **WHEN** der Autor ein Bild auf einen `draft`-Bericht hochlädt
- **THEN** liefert das System HTTP 201

#### Scenario: Medien-Freigeber lädt Bild in Pending
- **WHEN** ein Freigeber ein Bild auf einen `pending_review`-Bericht hochlädt
- **THEN** liefert das System HTTP 201

#### Scenario: Autor versucht Bild-Upload nach Submit
- **WHEN** der Autor auf einen `pending_review`-Bericht ein Bild hochlädt
- **THEN** liefert das System HTTP 403

#### Scenario: Elftes Bild
- **WHEN** bereits 10 Bilder am Bericht hängen und ein weiteres hochgeladen wird
- **THEN** liefert das System HTTP 400 mit `{"error":"too_many_images"}`

#### Scenario: Falscher MIME-Type
- **WHEN** eine Datei ohne `image/jpeg` oder `image/png` MIME-Type hochgeladen wird
- **THEN** liefert das System HTTP 400 mit `{"error":"unsupported_mime"}`

#### Scenario: Bild-Anhängen im State published
- **WHEN** der Bericht bereits `published` ist
- **THEN** liefert das System HTTP 409

---

### Requirement: Bild-Löschen im State draft/publish_failed
Das System SHALL bei `DELETE /api/match-reports/{id}/images/{imgId}` das Bild aus DB und Filesystem entfernen. Zugriff nach derselben State-/Rollen-Matrix:
- **State `draft`**: nur Autor.
- **State `pending_review`**: nur Freigeber.
- **State `publish_failed`**: nur Freigeber.
- **State `publishing` / `published`**: HTTP 409.

Response: HTTP 204.

#### Scenario: Medien-Freigeber löscht Bild aus Pending
- **WHEN** ein Freigeber ein Bild von einem `pending_review`-Bericht löscht
- **THEN** liefert das System HTTP 204

#### Scenario: Bild-Löschen im State published
- **WHEN** der Bericht `published` ist
- **THEN** liefert das System HTTP 409

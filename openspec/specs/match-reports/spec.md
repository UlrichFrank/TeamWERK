# match-reports Specification

## Purpose
TBD - created by archiving change spielbericht-typo3-publisher. Update Purpose after archive.
## Requirements
### Requirement: Draft-Erstellung durch Slot-Owner
Das System SHALL bei `POST /api/match-reports` mit Body `{ game_id, duty_slot_id }` einen neuen Draft anlegen, wenn der authentifizierte User Rolle `presseteam` oder `admin` hat, den referenzierten Duty-Slot besitzt (`duty_slots.assigned_user_id = user.id`) und noch kein `match_report` für dieses Spiel existiert. Response: HTTP 201 mit `{id}`. State-Initial: `draft`. `author_user_id = user.id`.

#### Scenario: Nicht-Presseteam
- **WHEN** ein User mit Rolle `standard` `POST /api/match-reports` aufruft
- **THEN** liefert das System HTTP 403

#### Scenario: Slot gehört anderem User
- **WHEN** ein Presseteam-User einen `duty_slot_id` referenziert, den er nicht besitzt
- **THEN** liefert das System HTTP 403

#### Scenario: Zweiter Draft für dasselbe Spiel
- **WHEN** bereits ein `match_report` mit `game_id=X` existiert und ein weiterer Draft angelegt werden soll
- **THEN** liefert das System HTTP 409 mit `{"error":"report_exists"}`

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

### Requirement: Publish mit atomarem State-Übergang
Das System SHALL bei `POST /api/match-reports/{id}/publish` folgende Schritte in dieser
Reihenfolge ausführen:
1. Atomarer Übergang `pending_review|publish_failed → publishing` via
   `UPDATE match_reports SET state='publishing' WHERE id=? AND state IN (?, ?)`.
   Wenn 0 Zeilen betroffen: HTTP 409 (`already_published`, `not_submitted` oder
   `in_progress`).
2. Payload zusammensetzen. Diese enthält u.a.:
   - `title`: aus `match_reports.title` (nicht regeneriert),
   - `slug`: aus `TitleSlug(title)`,
   - `season`: aus `ParseSeasonName(active_season.name)` — Fehler
     `no_active_season` (HTTP 500) wenn keine aktive Saison existiert,
   - `team_category_name`: aus `db.TeamDisplayShort()` für das eine Team des Berichts,
   - `abstract`, `match_date`, `match_score`, `match_teams`, `tournament`, `body_html`.
3. HTTP-POST an `TYPO3_IMPORT_URL` mit Bearer-Auth (Multipart mit `meta`-JSON + Bildern).
4. Bei HTTP 200 vom Publisher: `state='published'`, `published_url`, `typo3_page_uid`,
   `published_at`, `reviewer_user_id` setzen; Duty-Slot als erledigt markieren;
   Bilder-Dateien + `match_report_images`-Zeilen löschen.
5. Bei allen anderen Fällen: `state='publish_failed'`, `error_message` befüllen;
   Bilder liegen lassen.

Bei Erfolg: HTTP 200 mit `{"pageUid": int, "url": string}`. Bei Publisher-Fehler:
HTTP 502 mit `{"error":"publisher_failed","detail":"..."}`.

#### Scenario: Doppel-Publish (Race)
- **WHEN** zwei gleichzeitige `POST /publish`-Requests auf denselben Bericht kommen
- **THEN** liefert genau einer den Erfolg (State atomar auf `publishing` gesetzt), der andere HTTP 409

#### Scenario: Publisher liefert 5xx
- **WHEN** der TYPO3-Endpoint HTTP 500 liefert
- **THEN** ist der State danach `publish_failed`, `error_message` gefüllt, Bilder bleiben in `./storage/match-report-images/`

#### Scenario: Publisher liefert 422 (category_not_found)
- **WHEN** der TYPO3-Endpoint HTTP 422 mit `{"error":"category_not_found","detail":"mZ9"}` liefert
- **THEN** ist der State `publish_failed`, `error_message` enthält den TYPO3-Fehlerdetail (mindestens den Kürzel-Wert), Bilder bleiben liegen

#### Scenario: Retry nach publish_failed
- **WHEN** ein Bericht `publish_failed` ist und `POST /publish` erneut aufgerufen wird (durch Freigeber)
- **THEN** wird der Publish-Versuch wiederholt, Bilder werden nicht doppelt gesendet (dieselben Dateien wie beim ersten Versuch)

#### Scenario: Publish mit korrekter Payload-Form
- **WHEN** ein Bericht mit `title="Sieg vs. HB Ludwigsburg"` in einer aktiven Saison `"2026/27"` publiziert wird und das Team-Kürzel `"mC2"` ist
- **THEN** enthält die Publisher-Payload alle Felder: `title="Sieg vs. HB Ludwigsburg"`, `slug="sieg-vs-hb-ludwigsburg"`, `season="2026-2027"`, `team_category_name="mC2"`

### Requirement: Read-Only nach Publish
Das System SHALL nach dem State-Wechsel auf `published` alle mutierenden Routen für diesen Bericht mit HTTP 409 (`already_published`) beantworten. Löschung, Update, Bild-Änderungen sind nicht mehr möglich. Der Bericht bleibt in TeamWERK sichtbar als Referenz mit Link zur TYPO3-URL.

#### Scenario: DELETE auf published Bericht
- **WHEN** `DELETE /api/match-reports/{id}` auf `state='published'` erfolgt
- **THEN** liefert das System HTTP 409

### Requirement: Fire-and-forget — kein Update-Weg
Das System SHALL keinen Update-Weg zur TYPO3-Seite anbieten. Es gibt keinen `PUT /api/match-reports/{id}/republish` oder ähnliches. Änderungen an einem `published`-Bericht erfolgen ausschließlich direkt in der TYPO3-Backend-Redaktion.

#### Scenario: Kein Republish-Endpoint
- **WHEN** ein Client `PUT`, `PATCH` oder `POST` auf einen imaginären Republish-Pfad ausführt
- **THEN** liefert das System HTTP 404 oder 405 (Route existiert nicht)

### Requirement: Foto-Consent-Warnhinweis vor Publish
Das System SHALL bei `GET /api/match-reports/{id}` in der Response die Liste der Team-Mitglieder mit `photo_consent=false` liefern (Feld `photo_consent_missing: [{first_name, last_name}, ...]`). Das Feld dient dem Formular als Warnhinweis. Kein Publish-Block — der Autor entscheidet.

#### Scenario: Team mit Mitgliedern ohne Foto-Freigabe
- **WHEN** das über `game_id → game_teams → team_members` gefundene Team drei Mitglieder mit `photo_consent=false` hat
- **THEN** enthält die GET-Response `photo_consent_missing` mit drei Einträgen

### Requirement: HTML-Sanitizer mit Allowlist
Das System SHALL beim Publish `body_md` durch Markdown-Renderer + Allowlist-Sanitizer laufen lassen. Erlaubte Tags: `p, h2, h3, strong, em, ul, ol, li, a[href], br`. Alle anderen Tags werden gestrippt (Inhalt bleibt). `<script>`, `<iframe>`, `on*`-Attribute werden **immer** entfernt, unabhängig von der Allowlist.

#### Scenario: Script-Injection wird gestrippt
- **WHEN** `body_md` enthält `<script>alert(1)</script>`
- **THEN** ist der gesendete HTML-Body frei von Script-Tags

#### Scenario: Erlaubte Tags durchgelassen
- **WHEN** `body_md` enthält `## Erste Halbzeit\n\nDer Auftakt war zäh.`
- **THEN** ist der gesendete HTML-Body `<h2>Erste Halbzeit</h2><p>Der Auftakt war zäh.</p>`

### Requirement: Season-Segment aus aktiver Saison (ohne Fallback)
Das System SHALL das Feld `season` (Format `"YYYY-YYYY"`) aus dem Namen der **aktiven**
Saison (`SELECT name FROM seasons WHERE is_active = 1 LIMIT 1`) via
`ParseSeasonName()` ableiten und zusammen mit `slug` (nur title-Segment) an die
TYPO3-Extension senden. Der Saisonname folgt dem Format `YYYY/YY` (z.B. `"2026/27"`)
und wird zu `YYYY-YYYY` expandiert (z.B. `"2026-2027"`); Jahrhundert-Wechsel wird
korrekt behandelt (`"99/00"` → `"1999-2000"`). Die Saisonzuordnung erfolgt zum
Publish-Zeitpunkt und ist unabhängig von der Saison, in der das Spiel stattfand
(`games.season_id`). Existiert keine aktive Saison, liefert der Publish-Handler
HTTP 500 mit `{"error":"no_active_season"}` und `state` bleibt unverändert.
Es gibt keinen heuristischen Fallback aus dem Spieldatum mehr.

#### Scenario: Reguläre Saison-Bildung
- **WHEN** die aktive Saison den Namen `"2026/27"` hat und ein Publish erfolgt
- **THEN** enthält die Publisher-Payload `season="2026-2027"`

#### Scenario: Spiel in alter Saison, Publish in neuer aktiver Saison
- **WHEN** ein Spiel am 29.06.2026 gespielt wurde (`games.season_id` verweist auf Saison „2025/26") und die aktive Saison zum Publish-Zeitpunkt „2026/27" ist
- **THEN** wird `season="2026-2027"` gesendet (aktive Saison hat Vorrang)

#### Scenario: Jahrhundert-Wechsel
- **WHEN** die aktive Saison `"99/00"` heißt
- **THEN** wird `season="1999-2000"` gesendet

#### Scenario: Keine aktive Saison
- **WHEN** `POST /api/match-reports/{id}/publish` erfolgt und in `seasons` kein Eintrag mit `is_active=1` existiert
- **THEN** liefert das System HTTP 500 mit `{"error":"no_active_season"}` und der State bleibt unverändert (nicht `publishing`, nicht `publish_failed`)

#### Scenario: Slug enthält nur das title-Segment
- **WHEN** der Bericht-Titel „Furiose Aufholjagd gegen HB Ludwigsburg" ist
- **THEN** ist `slug="furiose-aufholjagd-gegen-hb-ludwigsburg"` (kein `/spielberichte/…`-Präfix)

### Requirement: Editierbarer Berichts-Titel
Das System SHALL beim Anlegen eines Drafts (`POST /api/match-reports`) das Feld
`match_reports.title` mit dem Default-Wert `BuildTitle(matchDate, opponent)` (Format
`"DD.MM.YYYY — <Gegner>"`) befüllen. Autor:in kann den Titel danach via
`PUT /api/match-reports/{id}` überschreiben (Feld `title`, max. 200 Zeichen).
Der Publisher SHALL beim Publish den aus der DB gelesenen Titel verwenden — kein
frisches Regenerieren. Der TYPO3-Slug wird aus `title` per `TitleSlug()`
(kebab-case, Umlaut-Normalisierung) abgeleitet.

#### Scenario: Draft-Anlage setzt Default-Titel
- **WHEN** ein Presseteam-User `POST /api/match-reports` mit `{game_id: X, duty_slot_id: Y}` aufruft und das Spiel am 29.06.2026 gegen „HB Ludwigsburg" ist
- **THEN** liefert die anschließende `GET /api/match-reports/{id}`-Response `title="29.06.2026 — HB Ludwigsburg"`

#### Scenario: Autor überschreibt Titel
- **WHEN** der Autor `PUT /api/match-reports/{id}` mit Body `{"title": "Furiose Aufholjagd"}` sendet und der State `draft` ist
- **THEN** liefert `GET /api/match-reports/{id}` danach `title="Furiose Aufholjagd"`

#### Scenario: Publish nutzt gespeicherten Titel
- **WHEN** der Titel in der DB `"Furiose Aufholjagd"` ist und `POST /api/match-reports/{id}/publish` erfolgt
- **THEN** enthält die Publisher-Payload `title="Furiose Aufholjagd"` und `slug="furiose-aufholjagd"`

#### Scenario: Titel-Feld zu lang
- **WHEN** `PUT /api/match-reports/{id}` mit einem `title`-Wert von 201 Zeichen erfolgt
- **THEN** liefert das System HTTP 400 mit `{"error":"title_too_long"}`

### Requirement: Team-Category per Kürzel
Das System SHALL beim Publish das kanonische Team-Kürzel (via
`db.TeamDisplayShort()`, z.B. `"mB1"`, `"wC2"`) für das Team des Berichts ermitteln
und als Feld `team_category_name` (String) in der Publisher-Payload senden. Die
Publisher-Seite (TYPO3-Middleware) SHALL das Kürzel gegen `sys_category.title`
auflösen; kein Treffer führt zu HTTP 422 vom Publisher. TeamWERK behandelt das wie
jeden anderen Publisher-Fehler (State → `publish_failed`, `error_message` befüllt).

Ein Bericht bezieht sich immer auf genau **ein** Team (Invariante). Bei mehreren
Einträgen in `game_teams` (theoretisch, sollte nicht vorkommen) wird deterministisch
das erste per SQL-`ORDER BY t.id LIMIT 1` verwendet.

#### Scenario: Publish mit vorhandenem Kürzel
- **WHEN** das Team des Berichts das Kürzel `"mC2"` hat und eine `sys_category` mit `title="mC2"` in TYPO3 existiert
- **THEN** ist die Publisher-Payload `team_category_name="mC2"`, TYPO3 verknüpft die Seite mit der Kategorie, Response HTTP 200

#### Scenario: Publish ohne matchende Kategorie
- **WHEN** das Team-Kürzel `"mZ9"` ist, aber keine passende `sys_category` in TYPO3 existiert
- **THEN** liefert TYPO3 HTTP 422 mit `{"error":"category_not_found","detail":"mZ9"}`, TeamWERK setzt `state='publish_failed'` und `error_message` enthält die TYPO3-Fehlermeldung

### Requirement: Bild-URL-Format ohne /api-Prefix in der Response
Das System SHALL die `url`-Werte in Bilder-Objekten (in `GET /api/match-reports/{id}`
und `POST /api/match-reports/{id}/images`-Responses) als relativen Pfad **ohne**
`/api`-Prefix liefern, z.B. `"/match-reports/42/images/7/blob"`. Das Frontend nutzt
die Axios-Instanz mit `baseURL='/api'`, die den Prefix zentral setzt.

#### Scenario: Bild-Response liefert relativen Pfad
- **WHEN** `POST /api/match-reports/42/images` erfolgreich ein Bild mit ID 7 anlegt
- **THEN** enthält die Response `url="/match-reports/42/images/7/blob"` (nicht `"/api/match-reports/…"`)

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

### Requirement: Client verkleinert Bilder vor dem Upload
Das Frontend SHALL vor jedem `POST /api/match-reports/{id}/images` die ausgewählte Datei clientseitig verkleinern: Zielgröße ≤ 1 MB, längste Kante ≤ 1920 px, Ausgabeformat **JPEG** (nur JPEG — WebP ist im Server-MIME-Filter `image/jpeg`+`image/png` nicht enthalten und würde `HTTP 400 unsupported_mime` erzeugen). Bereits kleine Dateien (`file.size ≤ 1 MB`) werden unverändert übernommen. Der Server-seitige 8-MB-Cap in `internal/matchreports/images.go` bleibt unverändert und dient als Backstop.

#### Scenario: Kamera-JPG > 8 MB wird akzeptiert
- **WHEN** die/der Nutzer:in ein 12 MB großes JPG aus der Handy-Galerie auswählt
- **THEN** verkleinert das Frontend die Datei auf ≤ 1 MB JPEG, sendet `POST /api/match-reports/{id}/images` und der Server antwortet mit HTTP 201

#### Scenario: PNG unter Zielgröße bleibt unverändert
- **WHEN** die/der Nutzer:in ein 400 KB großes PNG auswählt
- **THEN** wird die Datei unverändert (ohne Recompression) an den Server gesendet und der Server antwortet mit HTTP 201

#### Scenario: HEIC vom iPhone kann nicht verkleinert werden
- **WHEN** die/der Nutzer:in ein HEIC/HEIF-Foto auswählt, das der Browser nicht als `ImageBitmap` dekodieren kann
- **THEN** überspringt das Frontend die Verkleinerung, sendet die Datei unverändert, der Server antwortet mit HTTP 400 `unsupported_mime`, und das Frontend zeigt „IMG_XXXX.HEIC — Format nicht unterstützt (nur JPG/PNG)"

### Requirement: Multi-Select-Upload mit Gesamt-Cap 10
Das Frontend SHALL im Bild-Upload-Auswahldialog des Spielbericht-Formulars die gleichzeitige Auswahl mehrerer Dateien erlauben (`<input type="file" multiple>`). Die ausgewählten Dateien werden **sequenziell** (nicht parallel) hochgeladen. Übersteigt die Summe aus bereits am Bericht hängenden Bildern (`report.images.length`) plus Anzahl der neu gewählten Dateien den Gesamt-Cap von 10, kürzt das Frontend die Auswahl **vor** dem ersten Upload auf die noch freien Slots und zeigt eine sichtbare Meldung („Nur die ersten N Bilder werden hochgeladen — Limit 10 erreicht"). Der Server-seitige Cap `MaxImages=10` (HTTP 400 `too_many_images`) bleibt der Backstop.

#### Scenario: Auswahl von 3 Bildern in leerem Bericht
- **WHEN** die/der Nutzer:in bei 0 vorhandenen Bildern drei Dateien im Picker auswählt
- **THEN** lädt das Frontend sie sequenziell nacheinander hoch, jeder Upload sendet einen eigenen `POST /images`-Request und das Formular zeigt am Ende drei Bild-Kacheln

#### Scenario: Auswahl übersteigt Cap
- **WHEN** der Bericht 7 Bilder hat und die/der Nutzer:in 5 Dateien im Picker auswählt
- **THEN** kürzt das Frontend die Auswahl auf die ersten 3 Dateien, lädt diese hoch, zeigt die Meldung „Nur die ersten 3 Bilder werden hochgeladen — Limit 10 erreicht" und sendet keine weiteren `POST /images`-Requests

#### Scenario: Vollständig blockiert wenn Cap bereits erreicht
- **WHEN** der Bericht bereits 10 Bilder hat
- **THEN** rendert das Frontend den „Bild wählen"-Button nicht, sodass keine weiteren Uploads gestartet werden können

#### Scenario: Upload-Reihenfolge folgt Auswahl
- **WHEN** die/der Nutzer:in drei Dateien in Reihenfolge A, B, C auswählt
- **THEN** wird A zuerst hochgeladen, dann B, dann C — jeder folgende Upload beginnt erst nach Abschluss (Erfolg oder Fehler) des vorigen

### Requirement: Sichtbare Fehleranzeige bei Bild-Upload
Das Frontend SHALL bei jedem gescheiterten `POST /api/match-reports/{id}/images` eine sichtbare, persistente Fehlermeldung im Bilder-Bereich anzeigen — pro fehlgeschlagene Datei ein Eintrag mit Dateiname und der übersetzten Fehlerursache. Die Anzeige wird beim nächsten Upload-Klick zurückgesetzt. Server-Fehlercodes werden wie folgt in deutsche User-Texte übersetzt:

| Server-`error`          | User-Text                                            |
|-------------------------|------------------------------------------------------|
| `too_many_images`       | Limit von 10 Bildern erreicht                        |
| `unsupported_mime`      | Format nicht unterstützt (nur JPG/PNG)               |
| `image_too_large`       | Datei ist zu groß nach Verkleinerung                 |
| `bad_multipart`         | Datei konnte nicht gelesen werden                    |
| `in_progress` / `already_published` / `not_found` | Bericht ist nicht mehr editierbar |
| _sonstiges / Netzfehler_| Upload fehlgeschlagen — bitte erneut versuchen       |

#### Scenario: Server lehnt HEIC ab
- **WHEN** ein `POST /images` mit HTTP 400 `{"error":"unsupported_mime"}` antwortet
- **THEN** zeigt das Frontend „<dateiname> — Format nicht unterstützt (nur JPG/PNG)" als sichtbaren Alert-Eintrag

#### Scenario: Mehrere Fehler bei Multi-Select
- **WHEN** drei Dateien hochgeladen werden und Dateien 1 und 3 mit `unsupported_mime` scheitern, Datei 2 erfolgreich ist
- **THEN** zeigt das Frontend nach Abschluss zwei Alert-Einträge (für Datei 1 und 3), Datei 2 erscheint als neue Bild-Kachel

#### Scenario: Netzfehler
- **WHEN** ein `POST /images` mit einem Netzwerkfehler abbricht (keine HTTP-Antwort)
- **THEN** zeigt das Frontend „<dateiname> — Upload fehlgeschlagen — bitte erneut versuchen"

#### Scenario: Fehleranzeige wird beim nächsten Klick zurückgesetzt
- **WHEN** nach einer Fehleranzeige die/der Nutzer:in den „Bild wählen"-Button erneut betätigt und eine weitere Datei auswählt
- **THEN** wird die alte Fehlerliste geleert und nur neue Fehler (falls welche entstehen) werden angezeigt

### Requirement: Upload-Fortschritt im Auswahl-Button
Das Frontend SHALL während eines laufenden Multi-Uploads am „Bild wählen"-Button den Fortschritt als `x/y` anzeigen (z. B. „Lade 3/5…") und den Button-Zustand `disabled` setzen, damit während der Sequenz keine parallele Auswahl gestartet werden kann.

#### Scenario: Fortschrittsanzeige bei 5-fach-Auswahl
- **WHEN** die/der Nutzer:in 5 Dateien auswählt und der Loop gerade Datei 3 hochlädt
- **THEN** zeigt der Button den Text „Lade 3/5…" und ist deaktiviert

#### Scenario: Button wieder aktiv nach Abschluss
- **WHEN** alle 5 Uploads abgeschlossen sind (unabhängig von Erfolg/Fehler pro Datei)
- **THEN** zeigt der Button wieder „Bild wählen" und ist bedienbar


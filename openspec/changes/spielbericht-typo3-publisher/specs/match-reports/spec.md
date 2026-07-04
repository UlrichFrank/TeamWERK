## ADDED Requirements

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
Das System SHALL bei `PUT /api/match-reports/{id}` das Draft aktualisieren, wenn der Requester der `author_user_id` entspricht (oder Admin ist) UND der State `draft` ist. Erlaubte Felder: `home_goals`, `away_goals`, `home_goals_ht`, `away_goals_ht`, `tournament`, `abstract`, `body_md`. Response: HTTP 200. `updated_at = now()`.

#### Scenario: Update im State published
- **WHEN** `PUT /api/match-reports/{id}` auf einen Bericht mit `state='published'` erfolgt
- **THEN** liefert das System HTTP 409 mit `{"error":"already_published"}`

#### Scenario: Update durch Fremd-User
- **WHEN** ein anderer Presseteam-User als der `author_user_id` versucht zu aktualisieren
- **THEN** liefert das System HTTP 403

#### Scenario: Update im State publishing (Race-Guard)
- **WHEN** der State `publishing` ist
- **THEN** liefert das System HTTP 409 mit `{"error":"in_progress"}`

### Requirement: Bilder anhängen mit Limit 10
Das System SHALL bei `POST /api/match-reports/{id}/images` mit multipart `file` (JPG/PNG) + `caption` das Bild in `./storage/match-report-images/{report_id}/` speichern und eine Zeile in `match_report_images` (position = max+1, caption, storage_path) anlegen. Response: HTTP 201 mit `{id, position, caption, url}`.

#### Scenario: Elftes Bild
- **WHEN** bereits 10 Bilder am Bericht hängen und ein weiteres hochgeladen wird
- **THEN** liefert das System HTTP 400 mit `{"error":"too_many_images"}`

#### Scenario: Falsches Mimetype
- **WHEN** eine Datei ohne `image/jpeg` oder `image/png` MIME-Type hochgeladen wird
- **THEN** liefert das System HTTP 400 mit `{"error":"unsupported_mime"}`

#### Scenario: Bild-Anhängen im State published
- **WHEN** der Bericht bereits `published` ist
- **THEN** liefert das System HTTP 409

### Requirement: Bild-Löschen im State draft/publish_failed
Das System SHALL bei `DELETE /api/match-reports/{id}/images/{imgId}` das Bild aus DB und Filesystem entfernen, wenn der State `draft` oder `publish_failed` ist. Response: HTTP 204.

#### Scenario: Bild-Löschen im State published
- **WHEN** der Bericht `published` ist
- **THEN** liefert das System HTTP 409

### Requirement: Publish mit atomarem State-Übergang
Das System SHALL bei `POST /api/match-reports/{id}/publish` folgende Schritte in dieser Reihenfolge ausführen:
1. Atomarer Übergang `draft → publishing` via `UPDATE match_reports SET state='publishing' WHERE id=? AND state='draft'`. Wenn 0 Zeilen betroffen: HTTP 409 (`already_published` oder `in_progress`).
2. Season-Pfad, Slug, Meta-Blob und Bilder in Multipart-Payload zusammensetzen.
3. HTTP-POST an `TYPO3_IMPORT_URL` mit Bearer-Auth.
4. Bei HTTP 200 vom Publisher: `state='published'`, `published_url`, `typo3_page_uid`, `published_at` setzen; Duty-Slot als erledigt markieren; Bilder-Dateien + `match_report_images`-Zeilen löschen.
5. Bei allen anderen Fällen: `state='publish_failed'`, `error_message` befüllen; Bilder liegen lassen.

Bei Erfolg: HTTP 200 mit `{"pageUid": int, "url": string}`. Bei Publisher-Fehler: HTTP 502 mit `{"error":"publisher_failed","detail":"..."}`.

#### Scenario: Doppel-Publish (Race)
- **WHEN** zwei gleichzeitige `POST /publish`-Requests auf denselben Bericht kommen
- **THEN** liefert genau einer den Erfolg (State atomar auf `publishing` gesetzt), der andere HTTP 409

#### Scenario: Publisher liefert 5xx
- **WHEN** der TYPO3-Endpoint HTTP 500 liefert
- **THEN** ist der State danach `publish_failed`, `error_message` gefüllt, Bilder bleiben in `./storage/match-report-images/`

#### Scenario: Retry nach publish_failed
- **WHEN** ein Bericht `publish_failed` ist und `POST /publish` erneut aufgerufen wird (durch Autor)
- **THEN** wird der Publish-Versuch wiederholt, Bilder werden nicht doppelt gesendet (dieselben Dateien wie beim ersten Versuch)

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

### Requirement: Season-Pfad-Bildung mit Fallback
Das System SHALL für den TYPO3-`slug` den Pfad `/spielberichte/{start-year}-{end-year}/{report-slug}` aus `seasons.start_date` und `seasons.end_date` (via `game_id → games.season_id`) bilden. Fehlt die Saison-Referenz oder liefert kein Ergebnis, wird als Fallback `{match_date.year}-{match_date.year+1}` verwendet und eine Warnung geloggt.

#### Scenario: Reguläre Saison-Bildung
- **WHEN** ein Spiel gehört zu einer Saison mit `start_date=2025-08-01, end_date=2026-06-30`
- **THEN** wird der Slug-Pfad `/spielberichte/2025-2026/{report-slug}` erzeugt

#### Scenario: Fallback ohne Saison
- **WHEN** `games.season_id IS NULL` und `match_date` ist im März 2026
- **THEN** wird der Fallback-Pfad `/spielberichte/2026-2027/{report-slug}` erzeugt und eine Warnung geloggt

### Requirement: HTML-Sanitizer mit Allowlist
Das System SHALL beim Publish `body_md` durch Markdown-Renderer + Allowlist-Sanitizer laufen lassen. Erlaubte Tags: `p, h2, h3, strong, em, ul, ol, li, a[href], br`. Alle anderen Tags werden gestrippt (Inhalt bleibt). `<script>`, `<iframe>`, `on*`-Attribute werden **immer** entfernt, unabhängig von der Allowlist.

#### Scenario: Script-Injection wird gestrippt
- **WHEN** `body_md` enthält `<script>alert(1)</script>`
- **THEN** ist der gesendete HTML-Body frei von Script-Tags

#### Scenario: Erlaubte Tags durchgelassen
- **WHEN** `body_md` enthält `## Erste Halbzeit\n\nDer Auftakt war zäh.`
- **THEN** ist der gesendete HTML-Body `<h2>Erste Halbzeit</h2><p>Der Auftakt war zäh.</p>`

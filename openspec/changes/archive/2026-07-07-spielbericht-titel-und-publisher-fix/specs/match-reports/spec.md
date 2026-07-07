## ADDED Requirements

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

## RENAMED Requirements

- FROM: `### Requirement: Season-Segment mit Fallback`
- TO: `### Requirement: Season-Segment aus aktiver Saison (ohne Fallback)`

## MODIFIED Requirements

### Requirement: Draft-Update nur durch Autor im State `draft`
Das System SHALL bei `PUT /api/match-reports/{id}` das Draft aktualisieren, wenn der
Requester der `author_user_id` entspricht (oder Admin ist) UND der State `draft` ist.
Erlaubte Felder: `title`, `home_goals`, `away_goals`, `home_goals_ht`, `away_goals_ht`,
`tournament`, `abstract`, `body_md`. Response: HTTP 200. `updated_at = now()`. Das
`title`-Feld hat eine Höchstlänge von 200 Zeichen (HTTP 400 bei Überschreitung).

#### Scenario: Update im State published
- **WHEN** `PUT /api/match-reports/{id}` auf einen Bericht mit `state='published'` erfolgt
- **THEN** liefert das System HTTP 409 mit `{"error":"already_published"}`

#### Scenario: Update durch Fremd-User
- **WHEN** ein anderer Presseteam-User als der `author_user_id` versucht zu aktualisieren
- **THEN** liefert das System HTTP 403

#### Scenario: Update im State publishing (Race-Guard)
- **WHEN** der State `publishing` ist
- **THEN** liefert das System HTTP 409 mit `{"error":"in_progress"}`

#### Scenario: Titel-Update im Draft
- **WHEN** der Autor `PUT /api/match-reports/{id}` mit `{"title":"Neuer Titel"}` sendet und der State `draft` ist
- **THEN** liefert das System HTTP 200 und `GET /api/match-reports/{id}` zeigt `title="Neuer Titel"`

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

## Why

Autor:innen können den Bericht-Titel derzeit nicht beeinflussen — er wird beim Publish
aus Datum + Gegner generiert (`29-06-2026-hb-ludwigsburg`), was in der URL hässlich
landet. Gleichzeitig produziert der Publisher zwei sichtbare Fehler:

1. Der **Saison-Pfad** greift die Saison des Spiels ab (`games.season_id`-Daten) — ein
   Spiel am 29.06.2026 fällt kalendarisch noch in Saison 2025/26 und landet unter
   `/spielberichte/2025-2026/…`, obwohl die aktive Saison bereits 2026/27 ist.
   Redaktion und Homepage-Filter sehen den Bericht dadurch nicht am erwarteten Ort.
2. Die **Team-Category** wird auf TYPO3-Seite nie gesetzt: `teams.typo3_category_uid`
   ist in der DB leer, die TYPO3-Middleware bricht bei `uid <= 0` still ab. Berichte
   erscheinen ohne Kategorisierung — keine Team-Seiten-Filterung möglich.

Zusätzlich zeigt das Formular Bild-Uploads nur als leeren grauen Kasten mit totem Link
(`/api/api/…` doppelter Prefix in `MatchReportFormPage.tsx:436`), was Autor:innen
denken lässt, das Bild sei nicht hochgeladen worden. Der Upload läuft technisch
korrekt, nur das Anzeigen ist kaputt.

## What Changes

- **Editierbarer Titel:** `match_reports.title` als neue Spalte. Beim Draft-Anlegen mit
  `BuildTitle(matchDate, opponent)` vorbelegt; Autor:in kann im Formular überschreiben.
  Publisher nimmt den gespeicherten Titel; Slug wird aus `title` per `TitleSlug()`
  abgeleitet (bleibt kebab-case, wird durch bessere Eingaben automatisch schöner).
- **Bild-Preview:** Doppelten `/api`-Prefix im Frontend entfernen. Statt Text-Link
  echtes `<img>`: Datei per authenticated `fetch` als Blob laden,
  `URL.createObjectURL()` → `src`. Kein Backend-Change am Bild-Endpoint.
- **BREAKING (Publisher-Contract): `team_category_uid` → `team_category_name`.**
  TeamWERK sendet das kanonische Kürzel des einen Teams (z.B. `"mC2"`) via
  `db.TeamDisplayShort()`. TYPO3-Middleware in `team-stuttgart-org` löst gegen
  `sys_category.title` auf; kein Treffer → HTTP 422, TeamWERK schlägt in
  `publish_failed` fehl. Spalte `teams.typo3_category_uid` wird per Migration
  entfernt (harter Cut — kein Übergangs-Fallback). Deploy-Reihenfolge:
  `team-stuttgart-org` zuerst, damit ein evtl. TeamWERK-Publish nicht 422 bekommt.
- **BREAKING (Season-Segment):** Statt Ableitung aus `games.season_id`-Datumsbereich
  wird `seasons.name` der **aktiven Saison** (`is_active=1`) geparst
  (`"2026/27"` → `"2026-2027"`). Fehlt die aktive Saison, schlägt Publish mit
  `no_active_season` (HTTP 500) fehl. Kein heuristischer Fallback mehr.
- **Ein Team pro Bericht — Invariante festhalten:** Der Kürzel-Lookup nimmt
  deterministisch das eine Team via `game_teams`; Kooperations-/Mehrteam-Spiele werden
  nicht unterstützt (bewusste Vereinfachung, in Domäne Team-Stuttgart aktuell auch
  nicht relevant).

## Capabilities

### New Capabilities

_Keine neuen Capabilities — die Änderungen betreffen ausschließlich die
bestehende `match-reports`-Spec._

### Modified Capabilities

- `match-reports`: Requirement „Draft-Update" wird um `title` erweitert. Requirement
  „Season-Segment mit Fallback" wird durch „Season-Segment aus aktiver Saison
  (ohne Fallback)" ersetzt. Neue Requirements „Editierbarer Berichts-Titel",
  „Team-Category per Kürzel" und „Bild-Preview per Blob-URL im Frontend" kommen
  hinzu. Requirement „Publish mit atomarem State-Übergang" wird um die neue
  Payload-Form angepasst.

## Impact

**TeamWERK — Backend (Go):**
- Neue Migration (nächste freie Nummer): `ALTER TABLE match_reports ADD COLUMN title TEXT NOT NULL DEFAULT ''`, `ALTER TABLE teams DROP COLUMN typo3_category_uid` (SQLite → Tabellen-Rebuild wie 019).
- `internal/matchreports/slug.go`: neue Funktion `ParseSeasonName("2026/27") → "2026-2027"` + Validierung. Alte `LoadSeasonRange` bleibt als Toter Code / kann entfernt werden.
- `internal/matchreports/create.go`: Default-Titel via `BuildTitle()` beim Draft-Anlegen setzen.
- `internal/matchreports/update.go`: `title` in Update-Whitelist aufnehmen (max. sinnvolle Länge, z.B. 200).
- `internal/matchreports/get.go`: `title` in Response.
- `internal/matchreports/publish.go`: Payload-Assembly ändert
  1. Titel aus DB (nicht `BuildTitle()`),
  2. Season aus `seasons` mit `is_active=1` (JOIN via aktive Saison, nicht `g.season_id`),
  3. `team_category_name` aus `db.TeamDisplayShort()`-Ausdruck (statt `t.typo3_category_uid`),
  4. Fehler `no_active_season` (HTTP 500), wenn keine aktive Saison existiert.
- `internal/matchreports/publisher.go`: `PublishMeta.TeamCategoryUID int` → `PublishMeta.TeamCategoryName string`.

**TeamWERK — Frontend (React/TS):**
- `web/src/pages/MatchReportFormPage.tsx`:
  - `MatchReport`-Typ um `title: string` erweitern; Titel-Input als erstes Feld im Formular (readOnly-Rules wie andere Felder).
  - `saveDraft()` sendet `title` mit.
  - `ImageTile`: Bild per `api.get(url, { responseType: 'blob' })` laden, in `URL.createObjectURL()`; `<img src>` statt Text-Link. Cleanup mit `URL.revokeObjectURL()` beim Unmount.
  - Doppelter `/api`-Prefix in `href` entfernt (URL kommt schon inkl. `/api` vom Server).

**team-stuttgart-org (TYPO3, separates Repo):**
- `MatchReportImportMiddleware.php`:
  - Required-Feld `team_category_uid` → `team_category_name`.
  - Neue Methode `resolveCategoryUidByName(string $name): int` — `SELECT uid FROM sys_category WHERE title = ? AND deleted = 0 LIMIT 1`.
  - Kein Treffer → HTTP 422 mit `{"error":"category_not_found","detail":"<name>"}`.
  - `attachTeamCategory($pageUid, $categoryUid)` bleibt (kriegt jetzt die resolvete UID).

**Deployment:**
- Harter Cut, aber Deploy-Reihenfolge: **team-stuttgart-org zuerst**, dann TeamWERK. Wenn TeamWERK vor dem TYPO3-Deploy publisht, bekommt es das alte Required-Feld `team_category_uid` mitgeteilt (fehlt jetzt) → 400/422 und `publish_failed`. Kein Datenverlust, aber ärgerlich.
- Keine Auswirkung auf bereits publizierte Berichte (`typo3_page_uid` + `published_url` sind eingefroren, keine Nachbearbeitung nötig).

**Tests:**
- Server: neue Testfälle für `PUT` mit `title`, Publish-Payload trägt `title`+`team_category_name`+`season` korrekt, Publish ohne aktive Saison → 500, `ParseSeasonName()` Grenzfälle (`25/26` → `2025-2026`, `99/00` → `1999-2000`).
- Frontend: Titel-Feld im Draft, Blob-URL-Preview lädt Bild als `<img>`, doppelter Prefix ist weg.
- team-stuttgart-org: Payload mit gültigem Kürzel verknüpft `sys_category`, unbekannter Name → 422.

## Context

Spielbericht-Publishing existiert seit `spielbericht-typo3-publisher` (archiviert). Die
Publisher-Pipeline in `internal/matchreports/publish.go` baut ein `PublishRequest` mit
Metadaten (Titel, Slug, Saison, Team-Category-UID, HTML-Body) und Bildern zusammen und
schickt das per Multipart an die TYPO3-Middleware in `team-stuttgart-org`. Nach dem
kürzlich abgeschlossenen `spielbericht-medien-gate` gibt es außerdem einen
`pending_review`-State und einen Freigeber-Gate (Medien/Vorstand).

Aktueller Stand für diese Change relevant:

- `match_reports` hat keine `title`-Spalte — `BuildTitle(matchDate, opponent)` wird
  beim Publish jedes Mal frisch erzeugt.
- `LoadSeasonRange(seasonStart, seasonEnd, matchDateUnix)` erwartet Datumsstrings der
  Saison (via `JOIN seasons ON s.id = g.season_id`) und leitet das Format `YYYY-YYYY`
  ab. Bei fehlender Saison-Referenz gibt es einen Fallback aus dem Spieldatum
  (Sommer-Sommer-Heuristik).
- `teams.typo3_category_uid` (Migration 021) hält die per Hand gepflegte sys_category-UID.
  In der Produktions-DB ist die Spalte für alle Teams NULL — deshalb wird die Category
  auf TYPO3-Seite nie gesetzt (Middleware skipt bei `uid <= 0` still).
- Der Frontend-`ImageTile` in `MatchReportFormPage.tsx` verlinkt per `<a href>` auf das
  Blob-Endpoint. Der `href` hat einen doppelten `/api`-Prefix; der Server liefert
  `image.url` bereits mit `/api/…`, das Frontend baut daraus `/api/api/…`. Zusätzlich
  fehlt eine echte Preview — die Kachel zeigt nur Text.

Constraints:
- SQLite kann kein `ALTER TABLE ... DROP COLUMN` mit gleichzeitigem CHECK-Constraint-Erhalt;
  DROP läuft aber seit SQLite 3.35 direkt. Migration 021 nutzte reines `ADD COLUMN`;
  für die DROP-Migration verwenden wir das direkte `DROP COLUMN` (SQLite 3.35+, im
  Repo bereits als Basis vorausgesetzt).
- Keine externen Bibliotheken für Bild-Vorschau (kein Blob-Wrapper, kein React-Query) —
  nur Fetch + `URL.createObjectURL`.
- Das Publisher-Contract ändert sich **BREAKING**. Deploy-Reihenfolge muss stimmen.

## Goals / Non-Goals

**Goals:**
- Autor:innen können den Bericht-Titel selbst setzen; Default aus Datum+Gegner.
- Slug in der TYPO3-URL wird aus dem gepflegten Titel abgeleitet und sieht dadurch
  automatisch besser aus.
- Saison-Segment im URL-Pfad kommt aus der aktiven Saison zum Publish-Zeitpunkt.
- TYPO3-Category wird zuverlässig gesetzt, indem TeamWERK das kanonische Team-Kürzel
  schickt und TYPO3 gegen `sys_category.title` auflöst. Kein Treffer = klarer Fehler,
  kein stilles Skippen.
- Bild-Uploads werden im Formular als echte Miniaturen angezeigt, damit Autor:innen
  sofort sehen, dass der Upload erfolgreich war.

**Non-Goals:**
- Multi-Image-Upload (bleibt Einzel-Upload — bewusst).
- Bild-Cropping, -Rotation oder Client-Downscale (kein Bedarf identifiziert).
- Übergangs-Kompatibilität zwischen altem und neuem Publisher-Contract (harter Cut).
- Migration bestehender publizierter Berichte (die frieren `typo3_page_uid` +
  `published_url` sowieso ein).
- Kooperations-/Mehrteam-Berichte (Invariante: ein Team pro Bericht).
- Änderung des `sys_category`-Auflösungsmodells auf TYPO3-Seite (z.B. Case-Insensitive
  Match, Fuzzy-Match) — exakter Titel-Match reicht.

## Decisions

### D-1: `match_reports.title` als NOT NULL DEFAULT '', Default bei Create

**Entscheidung:** Neue Spalte `title TEXT NOT NULL DEFAULT ''`, im `Create`-Handler
wird direkt nach `INSERT` ein `UPDATE ... SET title = ?` mit dem generierten Default
via `BuildTitle(matchDate, opponent)` gesetzt (oder direkt im `INSERT` mit vorbereitetem
Wert). Alternativ: Draft immer ohne Titel anlegen und beim ersten `PUT` durch den Autor
setzen.

**Rationale:** Sofortiger Default vermeidet Sonderfall „Draft ohne Titel". Autor sieht
im ersten Formular-Load bereits einen Vorschlag und muss nur überschreiben, wenn er
will. `NOT NULL DEFAULT ''` erlaubt bestehende Zeilen (Migration greift, altes
Testdaten-Setup bleibt kompatibel).

**Alternativen erwogen:**
- Nullable-Spalte: Bringt keinen Mehrwert, verkompliziert die Publish-Assembly (dann
  wieder Fallback nötig).
- Titel erst zum Submit-Zeitpunkt: Schlechte UX — Autor sieht bis dahin nichts.

### D-2: `ParseSeasonName` statt `LoadSeasonRange`

**Entscheidung:** Neue Funktion `ParseSeasonName(name string) (string, error)` parst
`seasons.name` im Format `YYYY/YY` (z.B. `"2026/27"`) zu `YYYY-YYYY` (z.B.
`"2026-2027"`). Regex `^(\d{4})/(\d{2})$`, `end = (start/100)*100 + end2`, aber mit
Century-Handling für Jahreswechsel (`99/00` → `1999-2000`).

Publisher fragt gezielt die aktive Saison ab:
```sql
SELECT name FROM seasons WHERE is_active = 1 LIMIT 1
```
Kein Match → `ErrNoActiveSeason` → HTTP 500 im Publish-Handler.

`LoadSeasonRange` in `slug.go` wird nicht gelöscht (kein Aufwand für Tests, die die
Funktion noch checken), aber im Publisher nicht mehr aufgerufen. Bei einer späteren
Aufräum-Runde kann sie raus.

**Rationale:** Saisonname ist die semantische Wahrheit; Datumsbereich ist eine
Ableitung. Die aktive Saison ist ohnehin die semantisch gewollte Zuordnung —
Publisher-Zeitpunkt-basiert ist konsistenter als Spieldatum-basiert.

**Alternativen erwogen:**
- `games.season_id` beibehalten, nur die aktive Saison mit dazu-JOINen und beim
  Publish überschreiben: Wortmenge, unklarer Vorteil.
- Autor kann Saison im Formular auswählen: bringt Konfigurations-Aufwand ohne
  ersichtlichen Nutzen (aktive Saison ist eh der 99%-Fall).

### D-3: `team_category_name` als String, TYPO3 löst UID auf

**Entscheidung:** `PublishMeta.TeamCategoryUID int` wird zu `TeamCategoryName string`.
Wert kommt aus `db.TeamDisplayShort()`-SQL-Expression (existiert schon, liefert
`mB1`, `wC2`, `mixed E`, etc.). TYPO3-Middleware macht:
```sql
SELECT uid FROM sys_category WHERE title = ? AND deleted = 0 LIMIT 1
```
Kein Treffer → HTTP 422 `{"error":"category_not_found","detail":"<name>"}`. Erfolg →
weitermachen wie bisher (`attachTeamCategory` mit resolveter UID).

Auf TeamWERK-Seite reicht dann der existierende Publisher-Error-Pfad
(`publish_failed` + `error_message`) — kein neuer Code.

**Rationale:** Kanonischer Name ist bereits im Rest der App der Identifier; TYPO3-UIDs
sind lokal an die TYPO3-Instanz gebunden und mussten bisher redundant in TeamWERK
gepflegt werden. Verlagerung des Lookups zu TYPO3 vermeidet Datenpflege in zwei Repos.

**Alternativen erwogen:**
- Lookup-Endpoint auf TYPO3 (TeamWERK ruft vor Publish `GET /sys-categories?name=…`
  auf, sendet dann UID): zusätzlicher Roundtrip, TeamWERK-Cache-Problem.
- Local-Cache in TeamWERK-DB für `kürzel → uid`: bringt die Datenpflege nur woandershin.

### D-4: Bild-Preview per Fetch → `URL.createObjectURL`

**Entscheidung:** `ImageTile` lädt das Bild im `useEffect` via `api.get(url, {
responseType: 'blob' })`, konvertiert per `URL.createObjectURL(blob)` in eine
`blob:`-URL und setzt sie in `<img src>`. Cleanup mit `URL.revokeObjectURL(previewUrl)`
beim Unmount und bei Wechsel des `image.id`.

Doppelter `/api`-Prefix (`href={`/api${image.url}`}`) wird korrigiert — der Server
liefert bereits die vollständige API-URL, wir übergeben sie direkt an die Axios-Instanz
(die den `/api`-Prefix intern nochmal drankleben würde — deshalb entweder direkt
Fetch mit dem vollen Pfad, oder wir strippen das `/api/` vom Server-Response und lassen
Axios den Prefix setzen). Cleanere Variante: der Server liefert die URL **ohne**
`/api/`-Prefix, Frontend übergibt an `api.get()` und Axios setzt den Prefix.
Diese Änderung ist minimal (`images.go:146`, `images.go:288`) und macht das Vertragsmodell konsistent
mit anderen Endpunkten.

**Rationale:** Fetch-basiert vermeidet den signierten-Token-Aufwand (siehe
HLS-Streaming-Muster) für einen niedrigfrequenten Preview-Case (Bilder verschwinden
nach Publish sowieso). `blob:`-URLs cachen nicht — bei erneutem Formular-Load lädt
das Bild neu, das ist bei ~1–2 MB Miniaturen unkritisch.

**Alternativen erwogen:**
- Signierter Query-Token (`?st=…`) wie bei HLS-Streaming: Overkill für Draft-Preview.
- Cookie-Auth für `/blob`-Endpoint: würde bei Freigebern (Vorstand) ohne aktive
  TeamWERK-Session brechen.

### D-5: URL-Konvention „ohne /api-Prefix in Server-Response"

**Entscheidung:** `images.go` liefert `image.url` als **relativer** Pfad
`/match-reports/{id}/images/{imgId}/blob` (ohne `/api`). Frontend nutzt
`api.get(image.url, { responseType: 'blob' })`, Axios setzt den `/api`-Prefix
automatisch.

**Rationale:** Konsistent mit dem Rest der Codebase, wo `lib/api.ts`-Instance den
Prefix zentral setzt. Das aktuelle Server-Design mit `/api/`-Prefix in der Response
ist eine Ausnahme, die wir korrigieren.

**Alternativen erwogen:**
- Frontend nutzt `fetch(image.url)` direkt (mit Bearer-Header manuell): dupliziert
  Auth-Logik.

## Risks / Trade-offs

- **Risk:** TeamWERK deployt vor `team-stuttgart-org` → Publish bekommt HTTP 400/422 vom
  alten `team_category_uid`-Required-Check.
  → **Mitigation:** Deploy-Reihenfolge in der PR-Beschreibung dokumentieren:
  `team-stuttgart-org` zuerst. Falls doch andersrum: `publish_failed` mit klarem
  Fehler, kein Datenverlust, manueller Retry nach TYPO3-Deploy möglich.

- **Risk:** `sys_category` auf TYPO3 hat keine Einträge, die zu den TeamWERK-Kürzeln
  passen — sofort alle Publishes brechen.
  → **Mitigation:** Vor Deploy manuell prüfen, dass für jedes aktive Team eine passende
  `sys_category` mit exaktem Titel existiert. In der PR-Description als Checkliste.
  Fehlermeldung im Frontend ist klar (`error_message` sichtbar), sodass Nutzer weiß
  wann's an fehlender Category liegt.

- **Risk:** `URL.createObjectURL` ohne Cleanup leakt Speicher (jedes Bild bleibt bis
  Page-Reload im Browser-Heap).
  → **Mitigation:** `useEffect`-Cleanup mit `URL.revokeObjectURL()` beim Unmount. Test:
  10-Bild-Formular öffnen, schließen, per DevTools `blob:`-URL-Zähler prüfen.

- **Trade-off:** Kein Fallback bei fehlender aktiver Saison → Publish scheitert hart.
  Ist im aktuellen Workflow praktisch unmöglich (Spiele anlegen setzt aktive Saison
  voraus, Gotcha in CLAUDE.md), aber theoretisch könnte jemand die
  `seasons.is_active`-Flags auf 0 setzen und danach publishen. Absichtlich hart —
  wir wollen keine Rate-Ergebnisse in URLs.

- **Trade-off:** `LoadSeasonRange` bleibt vorerst im Code — kein aktives Refactor, aber
  auch nicht mehr in Nutzung durch den Publisher. Dead-Code, Bestandsspecs (Season-
  Segment mit Fallback) werden per MODIFY überschrieben, Bestandstests bleiben.

## Migration Plan

1. **team-stuttgart-org PR + Deploy:**
   - Middleware akzeptiert `team_category_name` (required) statt `team_category_uid`.
   - `resolveCategoryUidByName` als neue private Methode.
   - Kein Retter für alten Feldnamen — harter Cut.
   - Test-Fixture-Payload updaten (`scripts/spike-match-report-import/fixture-payload.json`).
   - Deploy auf Produktion.

2. **TeamWERK PR + Deploy:**
   - Migration (nächste freie Nummer nach 021):
     - `ALTER TABLE match_reports ADD COLUMN title TEXT NOT NULL DEFAULT ''`
     - `ALTER TABLE teams DROP COLUMN typo3_category_uid` (SQLite 3.35+)
     - Down-Migration: `title` DROP; `typo3_category_uid` wieder ADD (`INTEGER`, nullable).
   - Publisher-Payload umschreiben.
   - Frontend-Änderungen.
   - Deploy auf Produktion.

3. **Verifikation:**
   - Ein Draft anlegen, Bild hochladen — Preview muss sichtbar sein.
   - Publish durchführen (aktives Team-Kürzel muss in TYPO3-`sys_category` existieren).
   - URL prüfen: `/spielberichte/{aktive-saison}/{user-titel-slug}`.
   - `sys_category`-Verknüpfung auf TYPO3 prüfen.

**Rollback:**
- Wenn TYPO3-Deploy scheitert: TeamWERK **nicht** deployen, alle Publishes gehen
  gegen den alten Contract weiter.
- Wenn TeamWERK-Deploy scheitert: `git revert` + Migration `down` (löscht `title`,
  fügt `typo3_category_uid` wieder ein).
- Bereits publizierte Berichte sind unbetroffen (eingefroren nach TYPO3-Insert).

## Open Questions

_Keine offenen Fragen — alle Design-Entscheidungen sind in der Explore-Konversation
mit dem Nutzer geklärt worden._

## 1. TYPO3-Middleware (team-stuttgart-org, separates Repo, ZUERST deployen)

- [ ] 1.1 `MatchReportImportMiddleware.php`: required-Feldliste — `team_category_uid` durch `team_category_name` ersetzen
- [ ] 1.2 Private Methode `resolveCategoryUidByName(string $name): int` mit `SELECT uid FROM sys_category WHERE title = ? AND deleted = 0 LIMIT 1`, wirft `RuntimeException` bei No-Match
- [ ] 1.3 Aufruf-Site vor `attachTeamCategory`: erst Name auflösen; No-Match → HTTP 422 mit `{"error":"category_not_found","detail":"<name>"}`
- [ ] 1.4 Fixture-Payload `scripts/spike-match-report-import/fixture-payload.json` auf `team_category_name` umstellen
- [ ] 1.5 OpenSpec-Proposal auf `team-stuttgart-org`-Seite spiegeln (kurzer Change, verweist auf TeamWERK-Proposal)
- [ ] 1.6 PR erstellen, mergen, auf Produktion deployen; sys_category-Titel prüfen (jedes aktive Team-Kürzel muss dort existieren)

## 2. Datenbank-Migration (TeamWERK)

- [x] 2.1 `internal/db/migrations/025_match_report_title_und_typo3_cat_drop.up.sql`: `ALTER TABLE match_reports ADD COLUMN title TEXT NOT NULL DEFAULT ''`, `ALTER TABLE teams DROP COLUMN typo3_category_uid`
- [x] 2.2 `025_match_report_title_und_typo3_cat_drop.down.sql`: umgekehrt (`title` DROP, `typo3_category_uid INTEGER` wieder ADD)
- [x] 2.3 Migration lokal testen: `make migrate-up && make migrate-down && make migrate-up` — keine Fehler
- [ ] 2.4 Commit: `feat(db): match_reports.title + teams.typo3_category_uid entfernt`

## 3. Backend — Slug/Season-Helper

- [x] 3.1 `internal/matchreports/slug.go`: `ParseSeasonName(name string) (string, error)` mit Regex `^(\d{4})/(\d{2})$` und Century-Handling
- [x] 3.2 Tests `slug_test.go`: `TestParseSeasonName_Standard` (`"2026/27"` → `"2026-2027"`), `TestParseSeasonName_CenturyBoundary` (`"99/00"` → `"1999-2000"`), `TestParseSeasonName_Invalid` (`""`, `"2026-27"`, `"foo"` → Fehler)
- [x] 3.3 Commit: `feat(matchreports): ParseSeasonName für Saison-Namen "YYYY/YY"`

## 4. Backend — Publisher-Payload

- [ ] 4.1 `internal/matchreports/publisher.go`: `PublishMeta.TeamCategoryUID int` → `PublishMeta.TeamCategoryName string`; JSON-Tag `team_category_name`
- [ ] 4.2 `internal/matchreports/publish.go` `assemblePublishRequest`: SQL-Query erweitern um `db.TeamDisplayShort("t")`-Ausdruck (Alias `team_short_name`); statt `typo3_category_uid` selecten
- [ ] 4.3 `assemblePublishRequest`: SQL erweitern um `title` aus `match_reports`
- [ ] 4.4 `assemblePublishRequest`: Aktive-Saison-Query hinzufügen: `SELECT name FROM seasons WHERE is_active = 1 LIMIT 1`; Ergebnis via `ParseSeasonName`; kein Match → `ErrNoActiveSeason`
- [ ] 4.5 `Publish`-Handler: `ErrNoActiveSeason` → HTTP 500 `{"error":"no_active_season"}` OHNE `finalizeFailed` (State bleibt `pending_review`/`publish_failed`, kein zusätzlicher State-Wechsel)
- [ ] 4.6 `assemblePublishRequest`: statt `BuildTitle(...)` den gespeicherten `title` aus DB nutzen; `TitleSlug(title)` bleibt
- [ ] 4.7 Publisher-Tests: neue Payload-Form (TeamCategoryName, Season aus aktiver Saison, Titel aus DB)
- [ ] 4.8 Commit: `feat(matchreports): Publisher-Payload — Titel/Saison/Kürzel-Kategorie`

## 5. Backend — Handler-Anpassungen

- [ ] 5.1 `internal/matchreports/create.go`: nach Draft-INSERT `UPDATE match_reports SET title = ? WHERE id = ?` mit `BuildTitle(matchDate, opponent)` (oder direkt im INSERT)
- [ ] 5.2 `internal/matchreports/update.go`: `title` in Whitelist; Validierung `len(title) <= 200` → HTTP 400 `{"error":"title_too_long"}`
- [ ] 5.3 `internal/matchreports/get.go`: `title` in SELECT + Response-Struct
- [ ] 5.4 `internal/matchreports/list.go`: falls Listen-Response `title` benötigt → dazunehmen (nur wenn UI's List-Ansicht das braucht; ansonsten skippen)
- [ ] 5.5 Handler-Tests (`handler_test.go`): Happy-Path `POST` → `GET` liefert Default-Titel; `PUT` mit Custom-Titel → `GET` zeigt neuen Titel; `PUT` mit 201-Zeichen-Titel → 400
- [ ] 5.6 Commit: `feat(matchreports): title-Feld in Create/Update/Get`

## 6. Backend — Bild-URL-Format

- [ ] 6.1 `internal/matchreports/images.go` (`UploadImage`-Response, `listImages`): `img.URL = fmt.Sprintf("/match-reports/%d/images/%d/blob", reportID, imgID)` (ohne `/api`-Prefix)
- [ ] 6.2 Tests `handler_test.go` / `images_test.go` (falls vorhanden): URL-Format prüfen
- [ ] 6.3 Publisher-Test: Test-fixture aktualisieren, falls betroffen (Bilder gehen als Multipart, nicht per URL)
- [ ] 6.4 Commit: `fix(matchreports): image URL ohne /api-Prefix (baseURL setzt der Client)`

## 7. Backend — Publish no-active-season Fehlerfall

- [ ] 7.1 Test `handler_test.go`: `Publish` ohne aktive Saison (Fixture setzt `seasons.is_active=0` überall) → HTTP 500 mit `{"error":"no_active_season"}`, State bleibt `pending_review`
- [ ] 7.2 Commit: `test(matchreports): Publish schlägt fehl ohne aktive Saison`

## 8. Frontend — Titel-Feld

- [ ] 8.1 `web/src/pages/MatchReportFormPage.tsx`: `MatchReport`-Typ um `title: string`; neuer `useState<string>('')` für Titel
- [ ] 8.2 `load()` befüllt `setTitle(r.title)`; `saveDraft()` sendet `title` mit
- [ ] 8.3 Neuer `<input>` als erstes Formularfeld — brand-Tokens, disabled bei readOnly
- [ ] 8.4 Commit: `feat(matchreports): Titel-Feld im Bericht-Formular`

## 9. Frontend — Bild-Preview per Blob

- [ ] 9.1 `ImageTile` in `MatchReportFormPage.tsx`: `useState<string|null>` für Blob-URL; `useEffect(() => { … }, [props.image.id])` lädt via `api.get(image.url, { responseType: 'blob' })`
- [ ] 9.2 Cleanup mit `URL.revokeObjectURL(url)` beim Unmount und bei Wechsel `image.id`
- [ ] 9.3 `<a href>` durch `<img src={previewUrl} alt="Bild {position}" />` ersetzen; grauer Kasten weg
- [ ] 9.4 Doppelter `/api`-Prefix in `MatchReportFormPage.tsx:436` entfernen (URL kommt jetzt ohne Prefix vom Server; Axios setzt ihn)
- [ ] 9.5 Fehlerfall: wenn `api.get` fehlschlägt → `<div>` mit brand-danger-Icon anzeigen statt kaputtem Bild
- [ ] 9.6 Commit: `fix(matchreports): Bild-Preview via Blob-URL statt totem Link`

## 10. Frontend — Fehlermeldungen sichtbar machen

- [ ] 10.1 `MatchReportFormPage.tsx`: `error_message`-Anzeige bei `publish_failed`-State soll den Publisher-Fehler klar zeigen (auch TYPO3-422-Detail)
- [ ] 10.2 Falls TeamWERK-Publish HTTP 500 `no_active_season` liefert: Toast oder `error`-State setzen mit dt. Übersetzung „Keine aktive Saison — bitte im Verein/Saisonen setzen"
- [ ] 10.3 Commit: `feat(matchreports): klare Fehleranzeige bei publish_failed / no_active_season`

## 11. Verifikation & Deploy

- [ ] 11.1 `make test` — alle Server-Tests grün (inkl. neue Payload-, Titel-, Season-Tests)
- [ ] 11.2 `pnpm -C web test` — Frontend-Tests grün (falls vorhanden für MatchReportForm)
- [ ] 11.3 `make lint` grün
- [ ] 11.4 `openspec validate spielbericht-titel-und-publisher-fix` grün
- [ ] 11.5 `/verify-change` durchlaufen — brand-Tokens, lucide-Icons, Broadcast+useLiveUpdates, Migrationsnummer, Route→Tests-Coverage
- [ ] 11.6 Lokaler End-to-End-Test: Draft anlegen → Titel überschreiben → Bild hochladen (Preview sichtbar) → Submit → Publish → URL enthält aktive Saison + user-Titel-Slug + sys_category-Verknüpfung auf TYPO3 sichtbar
- [ ] 11.7 Deploy team-stuttgart-org (falls in 1.6 noch nicht erledigt), dann `make deploy` für TeamWERK

## 12. Archivierung

- [ ] 12.1 Nach erfolgreicher Verifikation: `openspec archive spielbericht-titel-und-publisher-fix` (schlägt die MODIFIED-Requirements auf `openspec/specs/match-reports/spec.md` durch)
- [ ] 12.2 Commit: `docs(openspec): spielbericht-titel-und-publisher-fix archiviert`

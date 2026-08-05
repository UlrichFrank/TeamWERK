## 1. Migration — DB-Felder

- [ ] 1.1 `internal/db/migrations/042_h4a_import_ids.up.sql`: `ALTER TABLE venues ADD COLUMN hall_number INTEGER;` + `CREATE UNIQUE INDEX idx_venues_hall_number ON venues(hall_number) WHERE hall_number IS NOT NULL;`
- [ ] 1.2 Gleiche Migration: `ALTER TABLE games ADD COLUMN external_id TEXT;` + `CREATE INDEX idx_games_external_id ON games(external_id) WHERE external_id IS NOT NULL;` (kein globaler UNIQUE — Bestandsspiele ohne external_id koexistieren; Eindeutigkeit pro Saison später fachlich prüfen)
- [ ] 1.3 `042_h4a_import_ids.down.sql`: Indizes + Spalten via Tabellen-Rebuild entfernen (SQLite kennt kein DROP COLUMN mit Index sauber — Rebuild-Muster wie in `018_*` verwenden)
- [ ] 1.4 `make migrate-up` lokal + `make migrate-down`/`up` Roundtrip prüfen

## 2. Backend — Hallenlisten-Import erweitert (venue-csv-import)

- [ ] 2.1 `internal/venues/handler.go` `Import`: Spalte `Nummer` (`row[1]`) lesen und in eine `hallNumber`-Struktur je Zeile übernehmen (bislang bewusst verworfen)
- [ ] 2.2 Backfill-Match auf `(name, city, street)` statt nur `(name, city)`; bei eindeutigem Treffer `venues.hall_number` setzen
- [ ] 2.3 Mehrdeutigkeit (mehrere Nummern für gleiche Adresse) und No-Match erkennen → `hall_number` NULL lassen, in `importResult` als `ambiguous`/`unmatched` zählen
- [ ] 2.4 `importResult`-Struct um `HallNumbersAssigned`, `HallNumbersAmbiguous`, `HallNumbersUnmatched` erweitern
- [ ] 2.5 `internal/venues/handler_test.go`: Test „eindeutiger Backfill setzt hall_number", „mehrdeutige Adresse bleibt NULL + im Report", „manuelles Venue bleibt NULL", „neue Halle mit Nummer angelegt"

## 3. Backend — Neues Package internal/h4aimport (H4A-Client + Parser)

- [ ] 3.1 `internal/h4aimport/client.go`: HTTP-Client mit `cookiejar`, `Login(user,pw)` (Formular-POST an `index.php`, PHPSESSID), TLS-Pflicht (nur https), Timeout; Credentials nie loggen
- [ ] 3.2 `internal/h4aimport/client.go`: `FetchPeriods()` (GET edit.php → `ge_periods`-Optionen parsen) und `FetchGames(periodId)` (POST xajax `xajax_update`, S-kodierte Args + `opOwnGames=Son`, JSON→`gametable_container`-HTML extrahieren)
- [ ] 3.3 `internal/h4aimport/encode.go`: xajax-Argument-Encoder (`S`-Präfix + CDATA für Sonderzeichen), Unit-Test gegen den in `design.md` dokumentierten erwarteten Payload
- [ ] 3.4 `internal/h4aimport/parse.go`: HTML-Tabellen-Parser (`id="game<n>"` → Zeile; Spalten Staffel/Nr./Halle/Datum/Zeit/Heim/Gast/Kommentar); defensiv, klare Fehlermeldung bei Formatbruch statt stiller Teilergebnisse
- [ ] 3.5 `internal/h4aimport/parse_test.go`: Test gegen eingecheckte HTML-Fixture `testdata/edit_owngames.html` (aus dem echten Response entnommen, ohne Zugangsdaten) — 146-Zeilen-Fixture nicht nötig, repräsentative Teilmenge; prüft Nr./Halle/Heim-Gast/Typ-Ableitung

## 4. Backend — Mapping (Staffel→Mannschaft, Halle→Venue, Typ)

- [ ] 4.1 Entscheidung + Migration/Tabelle für gelerntes Staffel→Team-Mapping (`h4a_staffel_team_map (staffel TEXT, club_alias TEXT, team_id INTEGER)` oder Spalte an `teams`) — in derselben Migration 042 oder Folge-Task
- [ ] 4.2 Vereinsnamen-Alias-Liste („Team Stuttgart", „Team Stuttgart 2") für Eigenerkennung; Typ heim/auswärts aus Heim==eigener-Verein (nicht aus is_home_venue, siehe design)
- [ ] 4.3 Halle→Venue-Auflösung über `venues.hall_number`; unaufgelöste Hallen als Warnung
- [ ] 4.4 Unit-Tests für Mapping-Logik (bekannte/unbekannte Staffel, unaufgelöste Halle)

## 5. Backend — Preview/Apply-Handler + Routen

- [ ] 5.1 `internal/h4aimport/handler.go`: `Preview(w,r)` — Auth (vorstand/admin), Credentials aus Body, Login→Fetch→Parse→Map→Diff gegen `games` (Anker external_id), Logout, Plan zurück; Credentials nie in Response/Log
- [ ] 5.2 `Diff`-Logik: new/changed/unchanged; changed mit Feld-Alt/Neu; keine Löschungen; mögliche-Dublette-Erkennung (gleiches Datum+Team+Gegner ohne external_id)
- [ ] 5.3 `Apply(w,r)`: Entscheidungen entgegennehmen, je Zeile re-validieren (aktive Saison, team/venue existiert, template gültig), INSERT/UPDATE `games` + `game_teams`, external_id setzen
- [ ] 5.4 Batch-Regen: EIN `runAutoRegen` über Vereinigungsmenge aller Datumsfenster; EIN `hub.Broadcast("games")`; Spieler-Pushes unterdrückt (nur Regen-Summary an Importeur)
- [ ] 5.5 `internal/app/router.go`: `POST /api/games/import/h4a/preview` + `.../apply` im Vorstand-Tier registrieren; `Handlers`-Struct + `main.go`-Verdrahtung (`NewHandler(db, hub, cfg)`)
- [ ] 5.6 `internal/arch/broadcast_test.go`: `preview` in `broadcastAllowlist` mit Begründung („read-only, externer Abruf, kein DB-Write"); `apply` broadcastet regulär

## 6. Backend — Tests (Route-Pflicht: Happy + Fehlerfall)

- [ ] 6.1 `preview`: Happy-Path mit gemocktem H4A-Client (kein echter Netzugriff im Test) → 200 + Plan; Fehlerfälle 403 (kein vorstand), 502 (Login-Fehler), 400 (fehlende Felder)
- [ ] 6.2 `apply`: Happy-Path (new+changed geschrieben, external_id gesetzt) → 200; Fehlerfälle 400 (keine aktive Saison), 403 (Auth), skipped-Zählung bei fehlender Mannschaft
- [ ] 6.3 Idempotenz-Test: zweimaliges Apply desselben Plans erzeugt keine Duplikate (Anker external_id)
- [ ] 6.4 Credential-Nichtpersistenz-Test: nach Preview kein Passwort in Log-Buffer/DB (Assertion über injizierten Logger)
- [ ] 6.5 Batch-Test: Apply über mehrere Tage → genau ein Broadcast, ein Regen-Lauf (Spy auf hub/regen)

## 7. Frontend — Import-Modal im Kalender

- [ ] 7.1 `web/src/components/H4AImportModal.tsx`: Schritt 1 Credentials-Eingabe (user/pw, Saison-Dropdown aus preview-Vorabruf oder festes Feld), Schritt 2 Diff-Anzeige
- [ ] 7.2 Diff-Darstellung: Abschnitte NEU / GEÄNDERT (Alt→Neu pro Feld) / UNVERÄNDERT (ausblendbar); pro Zeile Checkbox, Staffel→Mannschaft-Select, Venue/Typ-Anzeige
- [ ] 7.3 Template-Auswahl: Batch je Typ (Heim/Auswärts → Template-Select) + selektiv je Zeile; brand-Tokens, lucide-Icons, verbindliche Klassen-Strings
- [ ] 7.4 `api.post('/games/import/h4a/preview', …)` → `apply`; `useLiveUpdates('games')` nach Apply reload; Fehlerbehandlung (502 generische Meldung)
- [ ] 7.5 Import-Button in `KalenderPage.tsx` (nur vorstand/admin sichtbar), öffnet Modal
- [ ] 7.6 `web/src/pages/AdminVenuesPage.tsx`: Import-Ergebnis um hall_number-Report (zugeordnet/mehrdeutig/nicht zugeordnet) erweitern
- [ ] 7.7 Vitest: Modal rendert Diff-Abschnitte, Batch-Template setzt alle, selektive Wahl überschreibt, nicht zugeordnete Zeile ist nicht bestätigbar

## 8. Doku & Sicherheit

- [ ] 8.1 `docs/agent/06-gotchas.md`: Gotcha „H4A-Import — fremde Zugangsdaten transient, nie loggen/persistieren, HTTPS-only, admin-getriggert, HTML-Parsing brüchig"
- [ ] 8.2 `docs/agent/04-api-db.md`: neue Routen im Vorstand-Tier ergänzen; `venues.hall_number`/`games.external_id` in Schema-Konventionen
- [ ] 8.3 ToS-Hinweis (BWHV/H4A-Rücksprache vor Produktivnahme) im Deployment-/Betriebsteil vermerken

## 9. Verifikation

- [ ] 9.1 Prod-Vorbereitung: Hallennummer-Backfill-Report einmal gegen Prod-`venues` ziehen und mit der lokalen Verifikation (1017 eindeutig / 1 ambig / 7 no-match) abgleichen
- [ ] 9.2 `/verify-change` (go vet/test/lint, pnpm build/test/lint, openspec validate, Broadcast-Gate, brand/lucide/Migrationsnummer)
- [ ] 9.3 `openspec validate h4a-import --strict`
- [ ] 9.4 Manueller End-to-End-Test mit echtem H4A-Login gegen lokales Backend (ein realer Import, Diff prüfen, Apply, Idempotenz durch zweiten Lauf)

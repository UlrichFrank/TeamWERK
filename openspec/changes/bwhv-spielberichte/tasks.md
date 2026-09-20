## 1. Migration — Schema

- [x] 1.1 `internal/db/migrations/068_bwhv_spielberichte.up.sql`: `ALTER TABLE kader ADD COLUMN staffel TEXT;` (nullable; die Beschränkung auf `kind='team'` erzwingt der Handler, nicht der CHECK — ein Tabellen-Rebuild nur dafür wäre unverhältnismäßig)
- [x] 1.2 Gleiche Migration: `bwhv_staffeln` (`season_id`, `code`, `name`, `org_id`, `sub_org_id`, `period_id`, `table_json`, `polled_at`, UNIQUE `(season_id, code)`)
- [x] 1.3 Gleiche Migration: `bwhv_games` (`staffel_id`, `game_no`, `sgid`, `game_id` → `games` ON DELETE SET NULL, Datum/Zeit, Teams, Tore + Halbzeit, `hall_number`, UNIQUE `(staffel_id, game_no)`, Index auf `(staffel_id, date)`)
- [x] 1.4 Gleiche Migration: `bwhv_reports` (`bwhv_game_id` UNIQUE, `sgid`, `state` CHECK `pending|parsed|parse_failed`, `pdf_path`, `spectators`, `referees`, `warnings_json`, `failure_reason`, `attempts`, `fetched_at`, `parsed_at`)
- [x] 1.5 Gleiche Migration: `bwhv_players` (`staffel_id`, `team_name`, `name`, `birth_year`, `member_id` → `members` ON DELETE SET NULL, `conflict`, UNIQUE `(staffel_id, team_name, name)`)
- [x] 1.6 Gleiche Migration: `bwhv_player_games` (PK `(report_id, player_id)`, `jersey_number`, Tore, 7m-Versuche/-Treffer, Zeitstrafen, Karten) und `bwhv_events` (`report_id`, `seq`, `clock_time`, `game_second`, Spielstand, `kind` CHECK, `side` CHECK, `player_id`, `jersey_number`, `raw_text`)
- [x] 1.7 `068_*.down.sql`: Tabellen in Abhängigkeitsreihenfolge droppen, `kader.staffel` entfernen
- [x] 1.8 Roundtrip gegen eine Wegwerf-DB grün: `migrate up` bis Version 68, dann `068_*.down.sql` und `068_*.up.sql` direkt via sqlite3 — down entfernt alle sechs Tabellen, die Indizes und `kader.staffel`, up stellt alles wieder her. **Nicht über `make migrate-down` geprüft:** `runMigrate` in `cmd/teamwerk/main.go` ignoriert das `up`/`down`-Argument und ruft immer `db.Migrate` (= up), das Target ist also wirkungslos. Bestandsbefund, eigener Folge-Change.

## 2. Foundation — internal/bwhv: HTTP-Client

- [x] 2.1 `go get github.com/ledongthuc/pdf`; `github.com/agnivade/levenshtein` von `// indirect` auf direkt ziehen; `go mod tidy`
- [x] 2.2 `internal/bwhv/client.go`: HTTP-Client mit Timeout, Projekt-`User-Agent`, HTTPS-Zwang, serielle Ausführung mit Pause zwischen PDF-Abrufen
- [x] 2.3 `internal/bwhv/client.go`: `FetchCatalog(orgID, subOrgID, periodID)` → `cmd=po`; Antwort auf `gClassID`/`gClassSname`/`gClassLname` reduzieren. **Bezirke über `o`, `og` bleibt die Verbands-Org** (design.md §1.2)
- [x] 2.4 `internal/bwhv/client.go`: `FetchSchedule(orgID, subOrgID, periodID, classID)` → `cmd=ps&ca=1`; Spiele + Tabelle + `head.repURL` liefern
- [x] 2.5 `internal/bwhv/client.go`: `FetchReport(repURL, sGID)` → PDF-Bytes; Content-Type und `%PDF`-Magic prüfen, sonst Fehler
- [x] 2.6 `internal/bwhv/testdata/`: Katalog- (BW + SRM), Spielplan- und PDF-Fixture plus die `permission denied`-Antwort eingecheckt; `README.md` dokumentiert Herkunft und Kürzung. **Das PDF ist ein synthetischer Nachbau** mit identischer Geometrie (gleiche Textpositionen, Spaltenkoordinaten, Spielnummer, Halle, Endstand) und erfundenen Personen — verifiziert gegen das Original: 26 Kadereinträge, 18 Verlaufsschlüssel, 0 mehrdeutig, gleiche Trikotnummern-Kollisionen `[16, 46]`. Der echte Bericht trägt ~26 Klarnamen mit Jahrgang, überwiegend Minderjährige, und verstieße gegen `public-repo-hygiene` (`opensource-1-pii-cleanup`). Schiedsrichter-Namen in der Spielplan-Fixture ebenfalls ersetzt.
- [x] 2.7 `internal/bwhv/client_test.go`: Parsing der Katalog-/Spielplan-Antworten gegen die Fixtures; `sGID: 0` wird als „kein Bericht" erkannt; **kein Live-Abruf im Test**

## 3. Foundation — internal/bwhv: PDF-Parser

- [x] 3.1 `internal/bwhv/pdf.go`: Textebene mit Koordinaten extrahieren, Zeilen über Y gruppieren, Gruppen über X sortieren
- [x] 3.2 `internal/bwhv/pdf.go`: Spaltenmodell **aus der Kopfzeile** ableiten (benachbarte Kopfpositionen spannen die Bereiche auf); unbekannte Kopfzeile → Fehler mit ihrem Wortlaut, kein Raten
- [x] 3.3 `internal/bwhv/parse_header.go`: Berichtskopf lesen — Spielklasse, Spielnummer, Datum/Zeit, Spielort + Hallennummer, Teams, Endstand + Halbzeit, Zuschauer, Schiedsrichter
- [x] 3.4 `internal/bwhv/parse_roster.go`: Mannschaftslisten beider Teams über das Spaltenmodell; Platzhalter-Zeilen (`N.N. N.N.`) überspringen
- [x] 3.5 `internal/bwhv/parse_timeline.go`: Spielverlauf — Tor, 7m-Tor, 7m ohne Tor, Verwarnung, 2-min-Strafe, Disqualifikation, Auszeit; unbekannte Form als `other` mit `raw_text`; Spielzeit `mm:ss` → Sekunden
- [x] 3.6 `internal/bwhv/crosscheck.go`: gestufte Kreuzprobe — Verlaufs-Summe ≠ Kopf-Endstand → harter Fehlschlag; Listen-Summen ≠ Verlauf → Warnungsliste (design.md §5.3)
- [x] 3.7 `internal/bwhv/parse_test.go`: Fixture ergibt erwartete Kopfdaten, Kader beider Teams, Ereigniszahl und Endstand
- [x] 3.8 `internal/bwhv/crosscheck_test.go`: manipulierte Fixture mit verlorenem Tor → Fehlschlag; manipulierte Summenspalte → Warnung, kein Fehlschlag

## 4. Domain — internal/gamestats: Persistenz und Poll

- [ ] 4.1 `internal/gamestats/store.go`: Staffel-Snapshot schreiben (Katalog-Auflösung Code → `gClassID`, Spielplan, Tabelle); unbekannter Code → `slog.Error` + Staffel überspringen
- [ ] 4.2 `internal/gamestats/store.go`: Begegnungen upserten über `(staffel_id, game_no)`; `game_id` über `games.external_id` verknüpfen. **Niemals in `games` schreiben**
- [ ] 4.3 `internal/gamestats/poll.go`: Fensterlogik — poll-fähig ab frühestem heutigen Anwurf + 2 h, solange ein heutiges `sgid` fehlt; Zustand rein abgeleitet, kein Zustandsfeld
- [ ] 4.4 `internal/gamestats/poll.go`: PDF-Abruf je Begegnung mit `sgid` ohne Bericht; Ablage unter `BWHV_REPORT_DIR`, `attempts` hochzählen, Transportfehler bleibt `pending`
- [ ] 4.5 `internal/gamestats/poll.go`: Parse-Ergebnis persistieren (`bwhv_player_games`, `bwhv_events`, `warnings_json`); bei hartem Fehlschlag `state='parse_failed'` **ohne** Detailzeilen, PDF bleibt liegen
- [ ] 4.6 `internal/gamestats/store_test.go`: `TestPoll_ErzeugtKeineGamesZeilen` (90 Begegnungen, `games`-Zeilenzahl unverändert); `TestPoll_VerknuepftEigenesSpielUeberExternalId`
- [ ] 4.7 `internal/gamestats/poll_test.go`: `TestPollFenster_VorAnwurfPlusZweiStundenKeinAbruf`, `TestPollFenster_AlleSGIDVorhandenBeendetDenTag`, `TestPoll_KeinSGIDKeinAbruf`
- [ ] 4.8 `internal/gamestats/parse_persist_test.go`: `TestParse_EndstandAbweichungVerwirftBericht` (keine Zeile in `bwhv_player_games`/`bwhv_events`), `TestParse_DetailabweichungWirdGespeichertUndGewarnt`

## 5. Domain — internal/gamestats: Identitätsauflösung

- [ ] 5.1 `internal/gamestats/matching.go`: Staffel-interne Auflösung — Name primär mit Levenshtein (Schwelle als benannte Konstante: ≤ 2 ab 5 Zeichen, sonst exakt), Nummer bestätigend und gleichstandsbrechend
- [ ] 5.2 `internal/gamestats/matching.go`: Widerspruch Name/Nummer in `bwhv_players.conflict` festhalten statt auflösen
- [ ] 5.3 `internal/gamestats/matching.go`: eigene Spieler auf `members` auflösen (Name, `date_of_birth` ↔ Jahrgang, `jersey_number`, Kader der Saison); mehrdeutig → offen lassen
- [ ] 5.4 `internal/gamestats/matching_test.go`: `TestMatching_SchreibvarianteIstDieselbePerson`, `TestMatching_TrikotwechselZerreisstPersonNicht`, `TestMatching_ZweiGleicheNamenWerdenPerNummerGetrennt`, `TestMatching_PlatzhalterErzeugtKeinenSpieler`
- [ ] 5.5 `internal/gamestats/matching_test.go`: `TestMatching_EindeutigesMitgliedWirdZugeordnet`, `TestMatching_MehrdeutigBleibtOffen`, `TestMatching_ManuelleZuordnungHaelt`

## 6. Domain — internal/gamestats: Routen

- [ ] 6.1 `internal/gamestats/handler.go`: `GET /api/staffeln`, `GET /api/staffeln/{id}/tabelle`, `GET /api/staffeln/{id}/spielplan` — Fehler über `httpx.WriteError`, Listen über `httpx.Paging`
- [ ] 6.2 `internal/gamestats/handler.go`: `GET /api/staffeln/{id}/ranglisten` (Torschützen, 7m-Quote, Fair Play), eigene und fremde Spieler gleichermaßen
- [ ] 6.3 `internal/gamestats/handler.go`: `GET /api/bwhv-games/{id}/report` (404 solange kein Bericht), `GET /api/bwhv-reports/{id}/pdf` (expliziter `Content-Type`, `Content-Disposition: attachment`, 404 ohne Datei)
- [ ] 6.4 `internal/gamestats/handler.go`: `GET /api/members/{id}/saisonstatistik`; Berichte mit `parse_failed` fließen nicht ein
- [ ] 6.5 `internal/gamestats/handler.go`: `GET /api/bwhv/staffel-katalog` (Vorstand) und `POST /api/staffeln/{id}/poll` (Vorstand) — Poll läuft über `background.Go`, sendet `h.hub.Broadcast("bwhv-updated")`
- [ ] 6.6 `internal/app/router.go` + `main.go`: Lese-Routen im Authenticated-Tier, Katalog und Poll im Vorstand-Tier; `Handlers`-Struct und `NewHandler(db, hub, cfg)` verdrahten
- [ ] 6.7 `internal/gamestats/handler_test.go`: Happy-Path + Fehlerfall je Route gemäß Test-Anforderungen im Proposal (401/403/404)

## 7. Backend — Staffel am Kader

- [ ] 7.1 `internal/kader/handler.go`: `staffel` im `PUT` entgegennehmen (Tri-State: fehlt = unverändert, leer = löschen, Wert = setzen)
- [ ] 7.2 `internal/kader/handler.go`: Validierung über `h4aimport.ParseStaffel` gegen `gender`/`age_class`; nicht interpretierbar oder unpassend → HTTP 400; `kind='practice'` → HTTP 409
- [ ] 7.3 `internal/kader/handler_test.go`: `TestKaderStaffel_HappyPath`, `TestKaderStaffel_UnpassendesGeschlecht` (400), `TestKaderStaffel_UnpassendeAltersklasse` (400), `TestKaderStaffel_UebungsgruppeAbgelehnt` (409), `TestKaderStaffel_LeerenEntferntZuordnung`

## 8. Backend — Scheduler, Konfiguration, Deployment

- [ ] 8.1 `internal/scheduler/bwhv_poll.go`: Minutentakt-Job, wertet die Fenster aus §4.3 aus, startet den Lauf über `background.Go` (Goroutine-Gate)
- [ ] 8.2 `internal/scheduler/bwhv_poll.go`: täglicher Katalog-/Spielplan-Lauf 06:00 und Nachzügler-Lauf 08:00 für offene Begegnungen der Vortage
- [ ] 8.3 `internal/config`: `BWHV_REPORT_DIR` lesen (Default `./storage/bwhv-reports`), `.env.example` ergänzen
- [ ] 8.4 `deploy/setup-vps.sh`, die idempotente Storage-Schleife im `deploy`-Target des Makefile, `deploy/backup-cron.sh` und die Backup-/Umzugs-Targets um `BWHV_REPORT_DIR` erweitern (Checkliste aus `docs/agent/10-deployment.md`)
- [ ] 8.5 `internal/scheduler/bwhv_poll_test.go`: Fenster-Tests mit Datei-DB unter `t.TempDir()` (In-Memory-SQLite trägt die Goroutine nicht, siehe `docs/agent/07-testing.md`)

## 9. Harness — Architektur- und Matrix-Gates

- [ ] 9.1 `internal/arch/arch_test.go`: `bwhv` als Foundation, `gamestats` als Domain klassifizieren; prüfen, dass `bwhv` kein Domain-Package importiert
- [ ] 9.2 `internal/permissions/object_matrix_test.go`: die neuen `{id}`-Lese-Routen mit Begründung nach `openByDesign` (vereinsweit sichtbare, öffentliche Verbandsdaten)
- [ ] 9.3 `internal/permissions/matrix_test.go`: neue Routen in der Tier-Matrix ergänzen
- [ ] 9.4 `make test` grün inkl. Broadcast-Gate (`POST /api/staffeln/{id}/poll` broadcastet), Push-Fan-out-Gate (unverändert — kein Push) und Goroutine-Gate

## 10. Frontend — Staffeln-Ansicht

- [ ] 10.1 `web/src/lib/api`-Aufrufe + Typen für Staffeln, Tabelle, Spielplan, Ranglisten, Bericht, Saisonstatistik
- [ ] 10.2 `web/src/pages/StaffelnPage.tsx`: Mannschafts-Umschalter als Header-Control (`HEADER_FIELD`), Reiter Tabelle / Spielplan / Ranglisten
- [ ] 10.3 `StaffelnPage`: Tabelle und Spielplan als `<table>` mit Mobile-`MobileCard`-Variante; nur `brand-*`-Tokens, Icons aus `lucide-react`
- [ ] 10.4 `StaffelnPage`: `useLiveUpdates` auf `bwhv-updated`
- [ ] 10.5 `web/src/App.tsx` Route `/staffeln` + Nav-Eintrag in `AppShell.tsx` (alle Eingeloggten)
- [ ] 10.6 `StaffelnPage.test.tsx`: Umschalter wechselt die Staffel, Reiter rendern, Live-Update löst Reload aus

## 11. Frontend — Spielbericht, Profil, Vorbefüllung

- [ ] 11.1 Spieldetail: Reiter „Spielbericht" — Endstand/Halbzeit, Zuschauer, Schiedsrichter, Torschützen beider Teams, Strafen, PDF-Download; Warnungen aus `warnings_json` als Hinweis
- [ ] 11.2 Spieldetail: Spielverlauf als Timeline-Grafik (Torkurve über die Spielzeit), Hover/Tap zeigt Schütze und Spielstand
- [ ] 11.3 Spieldetail: nicht zugeordnete eigene Spieler kenntlich machen und Zuordnung anbieten
- [ ] 11.4 Spieler-Profil: Saisonbilanz (Spiele, Tore, 7m-Quote, Strafen, Verlauf)
- [ ] 11.5 `MatchReportEditor`: Ergebnisfelder aus dem BWHV-Bericht vorbefüllen, wenn einer vorliegt; Autor-Eingabe hat Vorrang; **kein Backend-Import** (design.md §8)
- [ ] 11.6 Vitest: Vorbefüllung bei vorhandenem Bericht, leere Felder ohne Bericht, `parse_failed` füllt nicht vor, Autor-Änderung überlebt

## 12. Dokumentation und Abschluss

- [ ] 12.1 `docs/agent/06-gotchas.md`: Absatz „BWHV-Spielberichte" — `og`/`o`-Falle, `sGID` als Bereitschaftssignal, Poll-Fenster ohne Zustandsfeld, gestufte Kreuzprobe, Identitätsregel (Name primär), Grenze zu `games`
- [ ] 12.2 `docs/agent/10-deployment.md`: Betriebsvorbehalt (User-Agent, kein Dauer-Polling, serielle Abrufe), `BWHV_REPORT_DIR` im Backup-Set, Speicherwachstum ~116 MB/Saison
- [ ] 12.3 `docs/agent/04-api-db.md`: neue Routen und Auth-Tiers ergänzen
- [ ] 12.4 `/verify-change` ausführen; `openspec validate --strict` grün
- [ ] 12.5 Vor dem ersten Prod-Lauf: Staffeln an den neun Kadern der aktiven Saison pflegen, dann einen manuellen Poll je Staffel auslösen und das Ergebnis sichten

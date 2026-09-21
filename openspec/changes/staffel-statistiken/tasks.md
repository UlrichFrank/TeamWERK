## 1. Schiedsrichter getrennt erfassen (Parser + Schema)

- [x] 1.1 Migration `069_bwhv_referees_split` (up/down) legt `bwhv_reports.referees_json TEXT NOT NULL DEFAULT ''` und `referees_uncertain INTEGER NOT NULL DEFAULT 0` an; `referees` bleibt unangetastet. Verifikation: `make migrate-up` läuft durch, `make migrate-down` nimmt beide Spalten zurück, `.schema bwhv_reports` zeigt sie.
- [x] 1.2 `parseReferees` (`internal/bwhv/parse_header.go`) liefert `([]string, bool)` — Namen aus `textLine.Groups` (ein Lauf je Name), zweiter Rückgabewert markiert die unsichere Trennung. Verifikation: neuer Test in `internal/bwhv/parse_test.go` mit zwei Läufen ⇒ zwei Namen, `uncertain=false`.
- [x] 1.3 Namensregel als Rückfall für den Einspalten-Fall (Aufteilung in zwei Vorname-Nachname-Paare), setzt `uncertain=true`. Verifikation: Test mit einem Lauf „Max Mustermann Peter Müller" ⇒ zwei Namen + `uncertain=true`; Test mit `N.N. N.N.` ⇒ leere Liste.
- [x] 1.4 Eine gar nicht interpretierbare Schiedsrichter-Zeile liefert eine leere Liste und lässt den Bericht **nicht** scheitern. Verifikation: Test belegt, dass der Bericht trotz unbrauchbarer Zeile `parsed` bleibt (Anforderung „Nicht trennbare Zeile kippt den Bericht nicht").
- [x] 1.5 `report.go` und `reports.go` schreiben `referees_json`/`referees_uncertain` mit; `ReportDetail` gibt beide aus (`referees` bleibt als Rohzeile im JSON). Verifikation: bestehende `internal/gamestats`-Tests bleiben grün, ein neuer Test liest einen Bericht mit zwei Schiedsrichtern zurück.

## 2. Aggregate im Store

- [x] 2.1 Neue Datei `internal/gamestats/aggregates.go` mit `CrossTable(ctx, staffelID)`: alle Mannschaften der Staffel plus je Paarung Ergebnis **oder** Datum, „gespielt" = `home_goals IS NOT NULL`. Verifikation: Tabellen-Test mit gespielter, torloser (`0:0`) und künftiger Begegnung deckt alle drei Zellenarten ab.
- [x] 2.2 `StandingsProgression(ctx, staffelID)`: Spieltage nach `date[:10]` gruppiert, kumulierte Punkte/Differenz/Tore, Sortierung Punkte → Differenz → Tore, Punktewertung als benannte Konstante mit Kommentar (design.md §2). Verifikation: Test über drei Spieltage prüft die Rangfolge je Spieltag, ein Test belegt das Trimmen des ISO-Timestamps.
- [x] 2.3 `TeamStats(ctx, staffelID)`: Torverhältnis/Angriff/Verteidigung aus `bwhv_games`, Fair-Play und Torverteilung aus `bwhv_player_games` (nur `state='parsed'`), je Mannschaft die Zahl der Spiele. Verifikation: Test belegt gefüllte Tor-Werte **ohne** jeden Bericht und eine leere (nicht nullwertige) Fair-Play-Wertung für eine Mannschaft ohne Bericht.
- [x] 2.4 Gini-Koeffizient über die Saisonsummen je Spieler (design.md §6), `0` bei Mittelwert `0`, dazu Median und Durchschnitt. Verifikation: Unit-Test mit gleichverteilten Werten (≈0), mit einem Alleinwerfer (nahe 1) und mit torloser Mannschaft (0).
- [x] 2.5 `RefereeStats(ctx, staffelID)`: je Schiedsrichter Spiele und die Strafen **beider** Mannschaften der von ihm geleiteten Spiele, `uncertain` durchgereicht, Berichte ohne Namen fallen raus. Verifikation: Test mit zwei Spielen desselben Gespanns prüft Summe und Spielzahl; Test mit namenlosem Bericht prüft, dass keine Zeile entsteht.
- [x] 2.6 `Affiliation(ctx, staffelID, userID)`: eigene BWHV-Mannschaftsnamen über `user_accessible_teams` → `game_teams` → `bwhv_games.game_id` → `games.is_home` (design.md §10), eigene `playerIds` über `bwhv_players.member_id` gegen eigene Mitglieder plus Kinder via `family_links`. Verifikation: Test belegt aufgelöste Mannschaft für Spieler, Trainer, erweiterten Kader und Elternteil; ein Test belegt die **leere** Menge ohne verknüpfte Begegnung trotz ähnlichen Namens.
- [x] 2.7 `StaffelStats` um `games` erweitern (Zahl der Berichte je Spieler) und die Siebenmeter-Fehlversuche ausweisen. Verifikation: Test belegt `games=3` für einen Spieler aus drei Berichten und `missed = attempts - goals`.

## 3. Routen und Handler

- [x] 3.1 Fünf Handler in `internal/gamestats/handler.go` (`GetCrossTable`, `GetProgression`, `GetTeamStats`, `GetRefereeStats`, `GetAffiliation` — letzterer liest `claims.UserID`), Fehler über `httpx.WriteError`. Verifikation: `go build ./...` und `golangci-lint` grün.
- [x] 3.2 Fünf Routen im Authenticated-Tier von `internal/app/router.go` neben `/tabelle`, `/spielplan`, `/ranglisten` eintragen (`/affiliation` englisch nach der Namenskonvention). Verifikation: Broadcast-Gate (`internal/arch/broadcast_test.go`) bleibt grün — es sind Lese-Routen, kein Allowlist-Eintrag nötig.
- [x] 3.3 Je Route Happy-Path und Fehlerfall testen (200 mit Daten, 401 ohne Token, 404 bei unbekannter Staffel-ID). Verifikation: neue Tests in `internal/gamestats` laufen grün.
- [x] 3.4 Die fünf neuen `{id}`-Routen in `openByDesign` der Objektrechte-Matrix eintragen, mit derselben Begründung wie `/tabelle` (öffentliche Verbandsdaten, vereinsweit sichtbar). Verifikation: `go test ./internal/permissions/` grün.

## 4. Frontend: Datenschicht

- [x] 4.1 Typen und Fetch-Funktionen in `web/src/lib/staffeln.ts` für die fünf neuen Routen; `PlayerStat` um `games`/`sevenMMissed` erweitern. Verifikation: `pnpm -C web build` grün (Typprüfung).
- [x] 4.2 Reiter-Definitionen in `StaffelnPage.tsx` auf neun erweitern, Reiterleiste horizontal scrollbar, gewählter Reiter weiter in der Adresse; Nachladen bei `bwhv-updated` deckt die neuen Daten mit ab. Verifikation: Vitest-Test prüft, dass ein `?tab=`-Parameter den zugehörigen Reiter rendert.

## 5. Frontend: Kreuztabelle und Fieberkurve

- [x] 5.1 Komponente `CrossTable`: Heim in der Zeile, Gast in der Spalte, gedrehte Spaltenköpfe, erste Spalte fixiert, `overflow-x-auto`, Diagonale leer, nur `brand-*`-Tokens. Verifikation: Vitest-Test prüft Endstand-Zelle, Datums-Zelle und leere Diagonale.
- [x] 5.2 Komponente `StandingsChart` als Inline-SVG: `<polyline>` je Mannschaft, Y-Achse invertiert (Rang 1 oben), Spieltage auf der X-Achse, Legende darunter mit Hervorhebung bei Hover, auf Mobile scrollbar. Verifikation: Vitest-Test prüft eine Polyline je Mannschaft und die Invertierung (Rang 1 hat das kleinere Y).
- [x] 5.3 Beide Komponenten in die Reiter einhängen, Leerzustand („noch nichts abgerufen") je Reiter. Verifikation: Vitest-Test prüft den Hinweis bei `polled=false` statt einer leeren Tabelle.

- [x] 5.4 Hervorhebung in Kreuztabelle (Zeile **und** Spalte der eigenen Mannschaft) und Verlauf (doppelte Strichstärke der eigenen Linie), jeweils `font-semibold` + `brand-table-select` + `aria-current` (design.md §11). Verifikation: Vitest-Test prüft die Markierung beider Achsen und die stärkere Linie.

## 6. Frontend: Statistik-Reiter

- [x] 6.1 Reiter Torverhältnis, Angriff, Verteidigung aus `teamstatistik`, jeweils mit Spielzahl und Sortierung wie im Vorbild. Verifikation: Vitest-Test prüft Sortierung nach Differenz bzw. Ø je Spiel.
- [x] 6.2 Reiter Fair-Play mit Punktwertung (Blau 4, Rot 3, 2-Min 2, Gelb 1), ausgewiesener Gewichtung und aufsteigender Sortierung; leere Wertung für Mannschaften ohne Bericht. Verifikation: Vitest-Test prüft die aufsteigende Sortierung und die leere Zelle.
- [x] 6.3 Reiter Verteilung mit Gini, Median, Ø und kurzer Erläuterung der Spalte. Verifikation: Vitest-Test prüft die Darstellung eines Gini-Werts und der Erläuterung.
- [x] 6.4 Reiter Torschützen um „Spiele" erweitern, Reiter 7m-Schützen mit Treffern, Versuchen, Fehlversuchen, Quote und Spielen (nur Spieler mit Versuch). Verifikation: Vitest-Test prüft, dass ein Spieler ohne Versuch in der 7m-Liste fehlt.
- [x] 6.5 Reiter Schiedsrichter mit Spielen und Strafen, Kennzeichnung unsicherer Trennung und dem Hinweis, dass die Strafen die des Spiels sind. Verifikation: Vitest-Test prüft die Kennzeichnung bei `uncertain=true`.

- [x] 6.6 Gemeinsamer Helfer für die Hervorhebung (`font-semibold` + `brand-table-select` + `aria-current="true"`) und Einsatz in **allen** Reitern inklusive Tabellenstand und Spielplan. Verifikation: Vitest-Test je Reiter prüft, dass genau die eigene Zeile markiert ist; ein Test prüft, dass ohne Zugehörigkeit keine Zeile markiert ist.
- [x] 6.7 Prüfen, dass die Hervorhebung von einer bereits fett gesetzten Wertspalte unterscheidbar bleibt (Torschützen, Angriff, Fair-Play). Verifikation: Vitest-Test belegt, dass die markierte Zeile zusätzlich zur Schriftstärke die Zeilenmarkierung trägt.

## 7. Abschluss

- [ ] 7.1 Gotcha-Absatz in `docs/agent/06-gotchas.md` ergänzen: Abdeckungs-Unterschied (Ergebnisse vs. Berichte), feste Zwei-Punkte-Annahme, Schiedsrichter-Trennung aus den PDF-Spalten, Auflösung der eigenen Mannschaft über `bwhv_games.game_id` statt Namensvergleich. Verifikation: Absatz vorhanden und nennt die Fundstellen im Code.
- [ ] 7.2 `/verify-change` ausführen: Build, Test, Lint, Route→Tests, brand-Tokens, lucide-Icons, Migrationsnummer, `openspec validate`. Verifikation: alle Prüfungen grün.

## 1. Backend: Zähl- und Gesamtsummen-Logik (`internal/dutyfairness`)

- [x] 1.1 Neues Package `internal/dutyfairness` anlegen, Handler-Struct-Pattern (`type Handler struct{ db *sql.DB }`)
- [x] 1.2 Query: `geleistet`/`vorhersage` je Kind — `COUNT(duty_assignments)` gruppiert nach `event_date < heute` vs. `>= heute`, unabhängig von `status`
- [x] 1.3 Query: `Gesamtsumme(Kader)` aus team-gebundenen `duty_slots` (via `game_id`→`game_teams` oder direkte `team_id`)
- [x] 1.4 Anteilige Verteilung generischer `duty_slots` (weder `game_id` noch `team_id`) proportional zur Spieleranzahl je Kader über alle Kader der aktiven Saison
- [x] 1.5 `Fair-Anteil(Kind) = Gesamtsumme(Kader) / Anzahl Spieler im Kader` — ohne Geschwister-Dedup; Kader-Mitglied ohne `family_links` gilt als eigene Familie
- [x] 1.6 Unit-Tests: `event_date = heute` zählt als Vorhersage; Gesamtsumme aus team-gebundenen + anteiligen generischen Slots; zwei Geschwister erhalten je vollen Anteil; `Gesamtsumme = 0` liefert `soll = 0` ohne Fehler

## 2. Backend: Rangliste-Endpoint & Sichtbarkeit

- [x] 2.1 Route `GET /api/duty-fairness/rangliste?team=<id,id>` in `internal/app/router.go` (Authenticated-Tier)
- [x] 2.2 „Eigene Teams"-Scope-Query für Standard-Nutzer (Teams mit `family_links`- oder eigener `kader_members`-Verbindung in der aktiven Saison)
- [x] 2.3 HTTP 403 wenn ein Standard-Nutzer ein `team`, zu dem er keine Verbindung hat, abfragt
- [x] 2.4 `admin`/Vereinsfunktion `vorstand`: alle aktiven Teams zulässig, keine Namens-Maskierung
- [x] 2.5 Anonymisierung für Standard-Nutzer: eigene Zeile mit echtem Namen, alle anderen nur mit Platzierung; stabile Sekundärsortierung nach `member_id` bei Gleichstand (keine geteilten Plätze)
- [x] 2.6 Mehrfachauswahl (`team=1,2`) liefert einen separaten, für sich sortierten Block pro Team — keine Vermischung
- [x] 2.7 Tests: Happy-Path (eigenes Team, anonymisiert bis auf eigene Zeile) · Fehlerfall 403 (fremdes Team, Standard-Nutzer) · Vorstand sieht alle Teams + alle Namen · Mehrfachauswahl liefert getrennte, korrekt sortierte Blöcke · Gleichstand ergibt stabile, eindeutige Reihenfolge

## 3. Backend: Dashboard-Endpoint umstellen (BREAKING)

- [x] 3.1 `internal/dashboard/handler.go`: `queryDutyAccount` liefert `dutyAccount` als Liste (eine Position pro Kind mit aktiver Kader-Mitgliedschaft; Spieler ohne Kind: eine Position für sich selbst)
- [x] 3.2 `computeSollForElternteil`/`computeAvgSlotsPerGame` entfernen, `soll` je Position aus der `dutyfairness`-Gesamtsumme-Berechnung beziehen (Wiederverwendung der Queries aus Abschnitt 1, kein Duplikat)
- [x] 3.3 Tests: ein Elternteil mit zwei Kindern in unterschiedlichen Kadern liefert zwei Positionen mit je eigenem `soll`; Kind ohne aktiven Kader erscheint nicht in der Liste; `Gesamtsumme = 0` liefert `soll = 0`

## 4. Frontend: Dashboard-Kachel

- [x] 4.1 `DashboardPage.tsx`: `dutyAccount` als Array rendern, eine Zeile pro Kind statt aggregierter Kachel
- [x] 4.2 Segment-Balken-Darstellung: `geleistet` (`brand-green`), `vorhersage` (`brand-info`), Rest bis `soll` (`brand-border-subtle`)
- [x] 4.3 Jede Zeile verlinkt auf die Rangliste-Seite des zugehörigen Kaders/Teams
- [x] 4.4 Vitest: mehrere Kinder zeigen mehrere Zeilen; `soll = 0` zeigt keinen Balken, nur den Zähler

## 5. Frontend: Rangliste-Seite

- [x] 5.1 Neue Seite `web/src/pages/DienstRanglistePage.tsx`
- [x] 5.2 Bestehenden `TeamFilter`/`lib/teamFilter.ts` wiederverwenden; Optionen kommen aus `GET /api/duty-fairness/rangliste`-Scope (eigene Teams bzw. alle Teams für Vorstand), nicht aus dem ungefilterten `GET /teams`
- [x] 5.3 Ein Ranglisten-Block pro ausgewähltem Team, Balkendiagramm absteigend nach `geleistet + vorhersage` sortiert
- [x] 5.4 Eigene Zeile mit echtem Namen hervorgehoben, fremde Zeilen nur mit Platzierung (kein erfundenes Pseudonym); für Vorstand alle Zeilen benannt
- [x] 5.5 Route in `App.tsx`, Nav-Eintrag in `AppShell.tsx`
- [x] 5.6 Vitest: Mehrfachauswahl zeigt getrennte Blöcke; Standard-Nutzer sieht nur eigene Zeile benannt; Vorstand sieht alle Namen; leerer Team-Filter (kein zugehöriges Team) zeigt sinnvollen Leerzustand

## 6. Abschluss

- [x] 6.1 `make test` + `golangci-lint` + `pnpm -C web build/test/lint` grün
- [x] 6.2 `/verify-change` bzw. manuelle Prüfung: Route→Tests vollständig (Happy-Path + Fehlerfall je neuer Route), keine rohen Tailwind-Farben, lucide-Icons statt Unicode
- [x] 6.3 `openspec validate dienste-familien-rangliste --strict`

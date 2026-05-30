## 1. DB Migration

- [x] 1.1 `internal/db/migrations/012_kader_games_per_season.up.sql`: `ALTER TABLE kader ADD COLUMN games_per_season INTEGER NOT NULL DEFAULT 0`
- [x] 1.2 `internal/db/migrations/012_kader_games_per_season.down.sql`: SQLite-safe Tabellen-Recreate ohne `games_per_season`

## 2. Backend — Kader-API

- [x] 2.1 Kader-Response-Struct in `internal/config/` (oder passendem Package) um `GamesPerSeason int` erweitern
- [x] 2.2 GET-Handler für Kader-Liste: `games_per_season` aus DB lesen und im JSON-Response mitsenden
- [x] 2.3 PUT/PATCH-Endpoint für Kader `games_per_season`: `PATCH /api/admin/kader/{id}/games-per-season` (body: `{"games_per_season": N}`) — nur admin/vorstand
- [x] 2.4 Route in `main.go` registrieren (admin+vorstand Middleware-Gruppe)

## 3. Backend — Dienstkonto-Formel

- [x] 3.1 Hilfsfunktion `computeAvgSlotsPerGame(db *sql.DB) (float64, error)`: Summiert `slots_count` aus `game_template_items` JOIN `game_templates` für `template_type='heim'` (is_active=1) und `template_type='auswärts'` (is_active=1); gibt `(heim + auswärts) / 2.0` zurück
- [x] 3.2 Hilfsfunktion `computeSollForElternteil(db *sql.DB, userID int, seasonID int, avgPerGame float64) (int, error)`: iteriert `family_links WHERE parent_user_id=userID`, sucht `kader_members → kader` für jedes Kind in der aktiven Saison, berechnet `child_soll` nach Formel (design.md D4), summiert und rundet
- [x] 3.3 In `internal/dashboard/handler.go`: bestehende pauschale `soll`-Berechnung für `elternteil` durch Aufruf von `computeSollForElternteil` ersetzen
- [x] 3.4 Edge Cases absichern: `player_count = 0` → Kind überspringen; `games_per_season = 0` → 0; kein aktives Template → avg = 0

## 4. Frontend — Dashboard

- [x] 4.1 `DashboardPage.tsx` Zeile 347: Accordion-Titel `"Nächste Spiele"` → `"Nächste Events"`
- [x] 4.2 `NextGamesList`-Komponente: Link-Ziel von `g.link` auf `/kalender?date=${g.date.slice(0,10)}` ändern
- [x] 4.3 `DutyAccountTile`-Komponente: Erklärtext für `elternteil` von `"Ziel: 5 Dienste × {children} Kinder = {soll}"` auf `"Ziel: {soll} Dienste (Saison {account.season})"` ändern
- [x] 4.4 `DutyAccountTile`: wenn `soll = 0` — Fortschrittsbalken ausblenden, nur Zähler „{ist} Dienste" anzeigen

## 5. Frontend — Kalender

- [x] 5.1 `KalenderPage.tsx`: `useSearchParams` aus `react-router-dom` importieren
- [x] 5.2 `year`/`month` State-Initialisierung: `?date=YYYY-MM-DD` auslesen, validieren (`!isNaN`), bei gültigem Wert `year`/`month` daraus setzen, sonst `new Date()`

## 6. Frontend — AdminKaderPage

- [x] 6.1 Kader-Interface um `games_per_season: number` erweitern
- [x] 6.2 In der Kader-Tabellenzeile/-Karte: nummerisches Input-Feld für `games_per_season` rechts neben Altersklasse (min=0, Schrittweite=1)
- [x] 6.3 `PATCH /api/admin/kader/{id}/games-per-season` bei Änderung aufrufen (Debounce oder onBlur)
- [x] 6.4 Feld auf Mobile korrekt eingebunden (Card-Layout, 44px Touch-Target)

## 7. Testen

- [ ] 7.1 Manuell: Admin setzt `games_per_season=20` für einen Kader → Dashboard-Elternteil sieht korrekten `soll`-Wert
- [x] 7.2 Manuell: Klick auf Event im Dashboard → Kalender öffnet richtigen Monat
- [x] 7.3 Manuell: `/kalender?date=foobar` → fällt auf aktuellen Monat zurück
- [ ] 7.4 Manuell: Kind mit 2 Elternteilen → jedes sieht halben soll-Wert
- [x] 7.5 Manuell: `games_per_season=0` → soll=0, kein Fortschrittsbalken

## 1. Backend — CarpoolingHint-Modell aktualisieren

- [x] 1.1 `CarpoolingEvent`-Struct aus `internal/dashboard/handler.go` entfernen
- [x] 1.2 `RecentEvents []CarpoolingEvent`-Feld aus `CarpoolingHint`-Struct entfernen
- [x] 1.3 Neuen Struct `CarpoolingOpenEntry` hinzufügen: `Typ string`, `Name string`
- [x] 1.4 `OpenEntries []CarpoolingOpenEntry`-Feld zu `CarpoolingHint`-Struct hinzufügen

## 2. Backend — Query für offene Einträge

- [x] 2.1 In `buildCarpoolingHint` (o.ä.): Event-Query-Block entfernen
- [x] 2.2 Neue Query für `openEntries` ergänzen: alle Einträge anderer Nutzer (user_id != ich) für das Spiel ohne `confirmed`-Pairing, JOIN auf `users` für Namen, LIMIT 5
- [x] 2.3 Ergebnis in `hint.OpenEntries` scannen (Typ + Vorname + Nachname)
- [x] 2.4 Sicherstellen, dass `hint.OpenEntries` als leeres Slice `[]` initialisiert wird (nicht nil)

## 3. Frontend — TypeScript-Interface aktualisieren

- [x] 3.1 `CarpoolingEvent`-Interface aus `DashboardPage.tsx` entfernen
- [x] 3.2 `recentEvents`-Feld aus `CarpoolingHint`-Interface entfernen
- [x] 3.3 Neues Interface `CarpoolingOpenEntry { typ: string; name: string }` hinzufügen
- [x] 3.4 `openEntries: CarpoolingOpenEntry[]`-Feld zu `CarpoolingHint`-Interface hinzufügen

## 4. Frontend — Hilfscode aufräumen

- [x] 4.1 `EVENT_TEXT`-Konstante entfernen
- [x] 4.2 `EventIcon`-Komponente entfernen
- [x] 4.3 Nicht mehr benötigte Lucide-Icons aus dem Import entfernen (`AlertTriangle`, `X`, `CircleDot` — sofern nicht anderweitig verwendet)

## 5. Frontend — CarpoolingHintCard neu gestalten

- [x] 5.1 Spielzeile kompakt: Datum + `· vs. Opponent` in einer Zeile, rechts der `Alle →`-Link
- [x] 5.2 Bestätigte Paarungen prominent mit `<Check>`-Icon und Name der Gegenseite
- [x] 5.3 `myEntry`-Statuszeile: kleines Badge „Mein Angebot" / „Mein Gesuch" wenn vorhanden
- [x] 5.4 `openEntries`-Liste: `<Car>`-Icon für `biete`, `<UserPlus>`-Icon für `suche`, Name
- [x] 5.5 „+ X weitere"-Zeile anzeigen wenn `bieteCount + sucheCount > openEntries.length + (myEntry ? 1 : 0) + paarungen.length`
- [x] 5.6 Fallback wenn keine Paarungen und keine offenen Einträge: „Noch keine Einträge" + Link

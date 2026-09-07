## 1. Aufstellungsstatus im Feed ermitteln

- [x] 1.1 `internal/calendar/handler.go` — `fetchGames`: zwei `EXISTS`-Spalten auf `game_lineup` in die Query aufnehmen (`lineup_exists` für das Spiel, `in_lineup` für `mem.member_id`) und in der Scan-Zeile mitlesen
- [x] 1.2 Einen Zustandstyp mit drei Werten (`aufgestellt` / `nicht aufgestellt` / `offen`) und einer Ableitungsfunktion aus `(lineup_exists, in_lineup)` ergänzen; der Zustand gilt nur bei `is_extended` **und** `event_type IN ('heim','auswärts')`, sonst bleibt er leer
- [x] 1.3 Prüfen, dass die bestehende Dedup (`seen[id]` + `ORDER BY … mem.is_extended`) unverändert greift: bei doppelter Zugehörigkeit gewinnt die reguläre Zeile und damit „kein Status"

## 2. Beschriftung

- [x] 2.1 `kaderLabel` kürzen: `<team> - erweiterter Kader` → `<team> · erw. Kader` (wirkt auch auf `fetchTrainings` — beabsichtigt, siehe design.md Decision 3)
- [x] 2.2 Den Status als dritten Bestandteil in die Mannschafts-Klammer hängen (`<team> · erw. Kader · <status>`), ohne Status unverändertes Verhalten
- [x] 2.3 `calEvent.Description` aus Notiz + Leerzeile + Statussatz zusammensetzen; leere Teile entfallen, die Notiz steht vorn

## 3. Tests

- [x] 3.1 `TestFeed_ErwKader_Aufgestellt` — `SUMMARY` mit `· erw. Kader · aufgestellt`, `DESCRIPTION` mit „Du bist für das Spiel aufgestellt."
- [x] 3.2 `TestFeed_ErwKader_NichtAufgestellt` — `SUMMARY` mit `· nicht aufgestellt`, `DESCRIPTION` mit dem vollständigen Absprache-Satz
- [x] 3.3 `TestFeed_ErwKader_KeineAufstellungGespeichert` — `SUMMARY` mit `· Aufstellung offen`; der Body enthält **nirgends** „NICHT aufgestellt" (die Invariante des Changes)
- [x] 3.4 `TestFeed_Stammkader_OhneStatus` — Titel ohne Kader- und ohne Status-Zusatz, `DESCRIPTION` ohne Aufstellungssatz
- [x] 3.5 `TestFeed_DoppelteZugehoerigkeit_RegulaerSchlaegtErweitert` — genau ein VEVENT, ohne Status
- [x] 3.6 `TestFeed_GenerischesEvent_OhneStatus` — `generisch` trägt keinen Status, auch für den erweiterten Kader
- [x] 3.7 `TestFeed_NotizUndStatusStehenBeide` — Notiz und Satz stehen beide im `DESCRIPTION`, Notiz zuerst
- [x] 3.8 Bestandstests zum Kader-Zusatz (Spiel **und** Training) auf die gekürzte Schreibweise ziehen

## 4. Abschluss

- [x] 4.1 `make test` + `golangci-lint` grün; `openspec validate kalender-aufstellungsstatus --strict`
- [x] 4.2 Ankündigungstext für die Trainer/Spieler entwerfen: neuer Statuszusatz, gekürzter Kader-Zusatz an Bestandsterminen, und dass der Kalender dem Stand erst beim nächsten Abruf folgt (Client-Polling)

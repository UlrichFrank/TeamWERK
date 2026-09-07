## Why

Ein Spieler des **erweiterten Kaders** erfährt von einem Heim-/Auswärtsspiel ausschließlich
über den iCal-Feed. Die Push-/E-Mail-Meldung zu neuen und geänderten Spielen erreicht ihn
nicht: `games.teamMembersAndParents` löst die Empfänger über `player_memberships` auf, und
diese View steht auf `kader_members` — der erweiterte Kader (`kader_extended_members`) ist
dort nicht enthalten. Der Kalendereintrag ist damit faktisch seine Einladung zum Spiel.

Diese Einladung sagt ihm aber nicht, was er als Erstes wissen muss: **ob er aufgestellt
ist.** Der Trainer pflegt die Aufstellung in `game_lineup`; sichtbar ist sie heute nur in
der Anwendung, und dorthin schaut niemand wegen eines Termins, der schon im Kalender steht.
Wer zum erweiterten Kader gehört, steht damit vor jedem Spieltag vor derselben Frage — und
fragt entweder nach oder fährt auf Verdacht hin.

## What Changes

- Der Kalendereintrag eines Heim-/Auswärtsspiels trägt für Nutzer, die über den
  **erweiterten Kader** am Spiel hängen, den Aufstellungsstatus — im `SUMMARY` kurz in der
  Mannschafts-Klammer und im `DESCRIPTION` als ganzer Satz:
  - aufgestellt → „Du bist für das Spiel aufgestellt."
  - nicht aufgestellt → „Du bist für das Spiel NICHT aufgestellt. Bitte mit der
    Trainerin/dem Trainer absprechen, ob eine Anwesenheit trotzdem erwünscht ist."
  - noch keine Aufstellung gespeichert → „Die Aufstellung für dieses Spiel steht noch nicht
    fest." Dieser dritte Zustand ist eigenständig: `game_lineup` ist für das Spiel leer, und
    daraus „nicht aufgestellt" zu machen, wäre eine erfundene Absage.
- **BREAKING** (nur Beschriftung, keine API): Der Kader-Zusatz im Titel wird von
  `<team> - erweiterter Kader` auf `<team> · erw. Kader` gekürzt, damit Mannschaft, Kader
  und Status zusammen in die Klammer passen. Das betrifft auch **Trainings**, die denselben
  Helfer (`kaderLabel`) benutzen — ein Termin darf nicht in zwei Schreibweisen desselben
  Zusatzes auftauchen.
- Die Kennzeichnung gilt **nur** für den erweiterten Kader und **nur** für `heim`/`auswärts`.
  Wer im Stammkader steht, bekommt keinen Status: für ihn ist die Teilnahme der Normalfall,
  und ein Vermerk an jedem Spiel wäre Rauschen. Generische Events haben keine Aufstellung.

## Capabilities

### New Capabilities

_(keine)_

### Modified Capabilities

- `ical-feed`: Spiel-Events des erweiterten Kaders tragen den Aufstellungsstatus in
  `SUMMARY` und `DESCRIPTION`; der Kader-Zusatz im Titel wird gekürzt.

## Impact

- `internal/calendar/handler.go` — `fetchGames` (zwei zusätzliche `EXISTS`-Spalten auf
  `game_lineup`), `kaderLabel`, `gameTitle`, Aufbau des `DESCRIPTION`-Feldes
- `internal/calendar/*_test.go` — neue Feed-Tests je Zustand
- `openspec/specs/ical-feed/spec.md` — zwei Requirements (Feed-Generierung, Konfigurierbare
  Feed-Inhalte) in ihren Beschriftungs-Szenarien
- Keine Migration, kein Schema, keine neue Route, keine neue Abhängigkeit. Der Feed ist ein
  Lesepfad ohne Broadcast — die Kennzeichnung erscheint beim nächsten Abruf des
  Kalender-Clients (Apple/Google pollen im Stundenraster), nicht sofort.

## Test-Anforderungen

| Route | Test | Erwartung |
|---|---|---|
| `GET /api/calendar/feed/{token}.ics` | `TestFeed_ErwKader_Aufgestellt` | `SUMMARY` enthält `· erw. Kader · aufgestellt`, `DESCRIPTION` den Satz „Du bist für das Spiel aufgestellt." |
| `GET /api/calendar/feed/{token}.ics` | `TestFeed_ErwKader_NichtAufgestellt` | `SUMMARY` enthält `· nicht aufgestellt`, `DESCRIPTION` den vollständigen Absprache-Satz |
| `GET /api/calendar/feed/{token}.ics` | `TestFeed_ErwKader_KeineAufstellungGespeichert` | `SUMMARY` enthält `· Aufstellung offen`; **kein** „NICHT aufgestellt" im Body |
| `GET /api/calendar/feed/{token}.ics` | `TestFeed_Stammkader_OhneStatus` | Titel ohne Kader- und ohne Status-Zusatz; `DESCRIPTION` ohne Aufstellungssatz |
| `GET /api/calendar/feed/{token}.ics` | `TestFeed_DoppelteZugehoerigkeit_RegulaerSchlaegtErweitert` | Ein Nutzer in beiden Kadern erhält genau ein VEVENT, ohne Status |
| `GET /api/calendar/feed/{token}.ics` | `TestFeed_GenerischesEvent_OhneStatus` | Ein `generisch`-Event trägt keinen Status, auch für den erweiterten Kader |
| `GET /api/calendar/feed/{token}.ics` | `TestFeed_NotizUndStatusStehenBeide` | Ein Spiel mit `note` liefert Notiz **und** Aufstellungssatz im `DESCRIPTION` |

Garantierte Invariante: Ein Spiel ohne jede Zeile in `game_lineup` erzeugt in keinem Feed
die Aussage „NICHT aufgestellt".

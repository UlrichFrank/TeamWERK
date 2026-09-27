## Why

Ob ein Spieler für ein Spiel aufgestellt ist, sieht man heute nur auf der Detailseite
des Spiels — und dort auch nur als Häkchen oder Strich. Der Strich steht dabei für zwei
verschiedene Dinge: „der Trainer hat dich nicht aufgestellt" und „der Trainer hat noch
gar keine Aufstellung gemacht". Auf `/termine` (Liste und Tabelle) fehlt die Aufstellung
ganz; man muss jedes Spiel einzeln öffnen.

Die richtige Unterscheidung existiert bereits an einer Stelle: der iCal-Feed kennt drei
Zustände (`aufgestellt` / `nicht aufgestellt` / `Aufstellung offen`), zeigt sie aber nur
für den erweiterten Kader. Diese drei Zustände und ihre Bezeichnungen sollen überall
gelten, wo die Aufstellung erscheint.

## What Changes

- **Ein Aufstellungsstatus mit drei Werten, überall gleich benannt:**
  `aufgestellt` · `nicht aufgestellt` · `Aufstellung offen`. Abgeleitet aus `game_lineup`
  nach der bestehenden Regel des iCal-Feeds („keine Zeile für das Spiel" ist *offen*,
  nie *nicht aufgestellt*). Gilt für Heim- und Auswärtsspiele und für alle Spieler des
  Kaders (Stamm- **und** erweiterter Kader), nicht für Trainer, Trainings und
  generische Termine.
- **Die Ableitung wandert an eine Stelle** (`internal/db`, SQL-Fragment) und wird von
  allen Lese-Routen genutzt, statt dass jede Route die beiden `EXISTS` selbst auswertet.
- **Eine Darstellung überall:** aufgestellt vollflächig Markengrün mit weißer Schrift,
  nicht aufgestellt vollflächig grau, Aufstellung offen nur grau gestrichelt umrandet
  (innen transparent). Das Zusage-Grün auf `/termine` wird dasselbe Markengrün.
- **Liste `/termine`:** Spielkarten zeigen den Status für die eigene Zeile und für jede
  Kind-Zeile (Eltern) als Text-Kennzeichen ohne Symbol, gleich breit, neben den
  Zu-/Absage-Knöpfen.
  `GET /api/games/my` liefert dazu `my_lineup` und je Kind `lineup`.
- **Tabelle `/termine?view=tabelle`:** jede Spielzelle trägt den Status als Fläche bzw.
  gestrichelte Umrandung hinter dem Rückmeldesymbol; die Legende erklärt alle drei.
  `GET /api/teams/{id}/rsvp-matrix` liefert je Spielzelle `lineup`.
- **Detailseite `/termine/spiel/{id}`:** die Spalte „Aufstellung" zeigt für jeden
  Spieler eine Checkbox im selben Schema (Trainer bedienbar, sonst nur lesend; bisher
  nur Häkchen/Strich); der Kartenkopf nennt den Gesamtzustand als Text-Kennzeichen
  („Aufstellung offen" bzw. „N aufgestellt").
  `GET /api/games/{id}/participants` liefert je Zeile `lineup` (neben dem bestehenden
  `in_lineup`), weil Spieler nicht alle Zeilen sehen und den Zustand „offen" deshalb
  nicht selbst ableiten können.
- **iCal-Feed:** der Aufstellungsstatus erscheint künftig auch für Spieler des
  **Stammkaders** (bisher nur erweiterter Kader). **Sichtbare Änderung für
  Kalender-Abos:** Stammspieler sehen im `SUMMARY` jedes Spiels ein Kennwort, bei nicht
  gepflegter Aufstellung `Aufstellung offen`.
- **Nutzerdoku** (`docs/anleitung-trainer.md`, `web/public/benutzerhandbuch.html`,
  `docs/schulung/folien.txt`) übernimmt die drei Bezeichnungen statt „nominiert" und
  streicht die überholte Aussage, eine Aufstellungsänderung verschicke keine
  Benachrichtigung.
- Keine Migration, keine neue Route, keine Mutation (kein neuer Broadcast). Die
  Aufstellung speichern bleibt `POST /api/games/{id}/lineup`, das bereits broadcastet.

## Nicht-Ziele

- **Keine explizite „Aufstellung veröffentlicht"-Markierung.** Der Zustand bleibt aus
  `game_lineup` abgeleitet; sobald der Trainer das erste Häkchen setzt, sind alle
  anderen „nicht aufgestellt". Eine Veröffentlichungs-Schaltfläche wäre ein eigener
  Change (Migration, Push).
- **Keine Aufstellung pro Mannschaft** bei Spielen mit mehreren Teams — `game_lineup`
  hängt am Spiel, das bleibt so (bestehende Semantik).
- **Keine Trainer-Zählung in der Liste** („8 von 14 aufgestellt") — die Liste zeigt die
  eigene und die Kind-Zeilen, wie bei der Rückmeldung.
- **Keine neue Benachrichtigung.** Die bestehende Meldung an aufgenommene bzw.
  herausgenommene Spieler (`notifyLineupChange`) bleibt, wie sie ist.

## Capabilities

### New Capabilities

_keine_

### Modified Capabilities

- `spiel-aufstellung`: einheitlicher dreiwertiger Aufstellungsstatus (Ableitung,
  Bezeichnungen, Geltungsbereich); Participants-Antwort trägt `lineup`; Detailseite zeigt
  drei Zustände.
- `termine-unified-view`: Spielkarten der Liste zeigen den Aufstellungsstatus (eigene
  Zeile, Kind-Zeilen).
- `termin-matrix`: Spielzellen zeigen den Aufstellungsstatus (Hintergrund), Spaltenkopf
  markiert „Aufstellung offen". (Capability liegt noch im nicht archivierten Change
  `termin-matrix`; der Delta ergänzt nur Requirements.)
- `ical-feed`: Aufstellungsstatus gilt für Stamm- und erweiterten Kader.

## Impact

- Backend: `internal/db/lineup_state.go` (neu, SQL-Fragment + Codes),
  `internal/games/handler.go` (`ListMyGames`, `attachChildrenRSVPToGames`,
  `GetParticipants`), `internal/attendance/matrix.go`, `internal/calendar/handler.go`
  (Stammkader-Zweig, Nutzung des Fragments); Tests in allen vier Packages.
- Frontend: `web/src/lib/lineup.ts` (neu, Bezeichnungen + Symbole, einzige Quelle),
  `web/src/pages/TerminePage.tsx`, `web/src/components/TerminMatrix.tsx`,
  `web/src/lib/terminMatrix.ts`, `web/src/pages/TermineDetailPage.tsx`; Vitest.
- API: additive Felder (`my_lineup`, `children_rsvp[].lineup`, `participants[].lineup`,
  `rsvp-matrix` `cells[].lineup`) — abwärtskompatibel.
- iCal: geänderte `SUMMARY`/`DESCRIPTION` für Stammspieler.
- Doku: `docs/anleitung-trainer.md`, `web/public/benutzerhandbuch.html`,
  `docs/schulung/folien.txt` (+ `make folien`).

## Test-Anforderungen

| Route | Fall | Erwartung |
|---|---|---|
| `GET /api/games/my` | Stammspieler, Spiel ohne `game_lineup`-Zeilen | 200, `my_lineup="open"` |
| | Stammspieler, Aufstellung enthält ihn | `my_lineup="in"` |
| | erweiterter Kader, Aufstellung ohne ihn | `my_lineup="out"` |
| | generisches Event / Trainer des Kaders | `my_lineup` fehlt |
| | Elternteil, Kind aufgestellt | `children_rsvp[].lineup="in"` |
| | unauthentifiziert | 401 |
| `GET /api/games/{id}/participants` | Spieler sieht Aufstellung, die nur Zeilen fremder Teams enthält | eigene Zeile `lineup="out"` (nicht `open`) |
| | keine Aufstellung | alle Spielerzeilen `lineup="open"`, Trainerzeilen ohne `lineup` |
| | fremdes Spiel | 404 (unverändert) |
| `GET /api/teams/{id}/rsvp-matrix` | Spielspalte mit Aufstellung | Zellen `in`/`out`; Trainingsspalte ohne `lineup` |
| | Spielspalte ohne Aufstellung | alle Zellen `open` |
| | Nutzer ohne Bezug | 403 (unverändert) |
| iCal-Feed | Stammspieler, Aufstellung offen | `SUMMARY` `Heim: Team (mB1 · Aufstellung offen) – …` + Satz |
| | Stammspieler aufgestellt | `(mB1 · aufgestellt)` |

Invariante: Für dasselbe Spiel und dasselbe Mitglied liefern Liste, Tabelle,
Detailseite und iCal-Feed denselben der drei Zustände; aus einer leeren Aufstellung
entsteht nirgends „nicht aufgestellt".

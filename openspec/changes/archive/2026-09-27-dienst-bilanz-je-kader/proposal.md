## Why

Die Dienst-Bilanz zählt heute je **Kind**, nicht je **Kind und Kader**. Steht ein Kind
im Stammkader zweier Mannschaften (Beispiel: Jakob Frank), erscheint es in beiden
Ranglisten mit **denselben** Zahlen — ein Dienst bei der einen Mannschaft wird in der
Rangliste der anderen mitgezählt. Gleichzeitig trägt das Kind in **jeder** Mannschaft
den vollen Fair-Anteil (`Gesamtsumme / Spieleranzahl`). Eine Familie mit so einem Kind
wirkt deshalb in beiden Mannschaften doppelt so fleißig, wie sie ist, und verdrängt
andere Familien in beiden Ranglisten nach unten.

## What Changes

- Die Zählung `geleistet`/`vorhersage` hängt künftig an der **Position (Kind, Kader)**
  statt am Kind. Eine Zuweisung zählt für die Mannschaft, deren Dienst sie ist.
- Die Rangliste eines Teams sortiert nach den Zahlen der Position in **diesem** Team;
  die Dashboard-Kachel zeigt je Position (Kind × Team) die Zahlen dieses Teams.
- Zuweisungen, die keiner einzelnen Mannschaft des Kindes eindeutig gehören — generische
  Slots (Vereinsfest), ein gemeinsames Spiel zweier Mannschaften des Kindes, und der
  Rückfall „eigene Zuweisung an einem fremden Team" (Stufe 5) — werden **gleichmäßig auf
  die Stammkader-Teams des Kindes geteilt**, auf die sie passen.
- Invariante: die Summe über alle Positionen eines Kindes ist gleich der bisherigen
  Kind-Zählung. Familiensummen, Gesamtsumme und Fair-Anteil je Team ändern sich nicht;
  Kinder in nur einem Stammkader sehen keinerlei Veränderung.
- **Verhaltensänderung (kein API-Bruch):** Die Response-Formen von
  `GET /api/duty-fairness/rangliste` und `GET /api/dashboard` (`meineDienste.dutyAccount`)
  bleiben gleich; für Kinder in mehreren Stammkadern ändern sich die Werte und damit die
  Platzierungen.
- Aushilfe (erweiterter Kader) bleibt unverändert je (Mitglied, Team) gezählt.

## Capabilities

### New Capabilities

_keine_

### Modified Capabilities

- `dienste-familien-rangliste`: Die Zählung „je Kind" wird zur Zählung „je Kind und
  Kader" samt Aufteilungsregel für nicht eindeutige Zuweisungen; Fair-Anteil und
  Sortierung beziehen sich ausdrücklich auf die Position im jeweiligen Kader.
- `dienstkonto-dynamische-soll-formel`: Die Dashboard-Position (Kind × Kader) trägt
  `geleistet`/`vorhersage` dieses Kaders statt der Gesamtzahl des Kindes.

## Impact

- **Backend:** `internal/dutyfairness/fairness.go` (Snapshot: Zähler je (Mitglied, Team),
  `countAssignments`, `Ranked`, `PositionsFor`), `internal/dutyfairness/handler.go`
  (Zeilenwerte), `internal/dashboard/handler.go` (`queryDutyAccount`).
- **Frontend:** keine Code-Änderung nötig (Response-Form unverändert); Benutzerhandbuch
  (`web/public/benutzerhandbuch.html`) erklärt die Aufteilung.
- **Doku:** Gotcha „Dienst-Bilanz je Kind" in `docs/agent/06-gotchas.md`.
- **Keine** Migration, keine neue Route, kein neues Recht; Berechtigungsmodell und
  Anonymisierung unverändert. RAM: eine Map je (Mitglied, Team) statt je Mitglied —
  vernachlässigbar.
- **Abhängigkeit:** baut auf dem umgesetzten, aber noch nicht archivierten Change
  `dienste-erweiterter-kader` auf (dessen Delta der Requirement „Geleistet- und
  Vorhersage-Zählung je Kind" ist Grundlage dieses Deltas). Dieser muss **vor** diesem
  Change archiviert werden.

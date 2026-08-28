## Why

`dienst-dauer-dynamisch` hat den Dauer-Modus eingeführt, aber zwei Entscheidungen
getroffen, die sich im Betrieb als falsch herausgestellt haben.

**1. Der Rückfall verschleiert Fehldefinitionen.** Ergibt die dynamische Auflösung keine
positive Dauer, trägt der Slot heute stumm die gepflegte `hours_value`. Das war als
Sicherheitsnetz gedacht („der Dienst fällt nie aus"), ist aber eine *falsche Zahl, die
richtig aussieht*: Die Dienstbörse zeigt eine Spanne, das Dienstkonto bucht Stunden, und
niemand erfährt je, dass die Versätze nicht zusammenpassen. Der Preis stand schon im
alten Kommentar — „der Preis ist, dass die Fehldefinition unsichtbar bleibt" — und er ist
zu hoch. Ein Rückfall ist außerdem nur nötig, solange falsche Werte überhaupt
gespeichert werden können.

**2. Die Maske sagt nicht, was sie tut.** Die Startzeit hängt an „Standard-Anker" +
„Versatz" — beide Namen verschweigen, dass sie den **Start** bestimmen. Die Dauer heißt
im dynamischen Modus „Dauer (Rückfall)" und steht *oberhalb* der Felder, die sie
ersetzen. Und die Modus-Namen (`Absolut` / `Dynamisch (folgt dem Spiel)`) beschreiben die
Implementierung, nicht die Entscheidung, die der Vorstand trifft: Gebe ich eine Dauer vor
oder ein Ende?

## What Changes

- **Der Rückfall entfällt.** Im Modus `dynamisch` ist die Dauer ausschließlich die
  Differenz aus aufgelöster End- und Startzeit. `hours_value` spielt dort keine Rolle
  mehr (der Wert bleibt gespeichert, damit ein Moduswechsel zurück nichts verliert).
- **Nicht auflösbare Kombinationen werden abgewiesen, statt aufgefangen.** Eine
  Konfiguration, deren Ende **niemals** nach dem Start liegen kann — gleicher Anker für
  Start und Ende, End-Versatz ≤ Start-Versatz — wird von `POST`/`PUT /api/duty-types`
  und von `PUT /api/duty-templates/{id}` mit HTTP 400 abgewiesen, vor jedem
  Schreibvorgang. Das Frontend prüft dieselbe Regel und blockiert das Speichern mit einer
  Meldung am Feld.
- **Bleibt beim Regen trotzdem keine positive Dauer** (nur noch erreichbar, wenn Start
  und Ende an *verschiedenen* Ankern hängen und die konkrete Spieldauer die Spanne
  zusammenschrumpfen lässt), **entsteht kein Slot.** Der Ausfall wird in der
  Regen-Zusammenfassung als eigener, rot ausgewiesener Punkt gemeldet
  (`RegenSummary.invalid_span`) — sichtbar statt stumm.
- **Die Modus-Namen benennen die Entscheidung:** `absolut` heißt in der Oberfläche
  **„Startzeit + Dauer"**, `dynamisch` heißt **„Startzeit + Endzeit"**. Die gespeicherten
  Werte (`absolut`/`dynamisch`) und die API bleiben unverändert — **keine Migration**.
- **Die Maske folgt der Rechnung** (`/diensttypen` und `/dienstplan-vorlagen`, identisch
  aufgebaut): erst der Modus, darunter **Start-Anker** + **Start-Versatz** (vorher
  „Standard-Anker" / „Versatz"), darunter je nach Modus entweder die **Dauer** oder
  **End-Anker** + **End-Versatz**. Kein Dauer-Feld mehr im Modus „Startzeit + Endzeit".
- **Der Termin-Dialog (`/kalender`) bleibt bei Startzeit + Dauer** — dort entsteht immer
  ein `is_custom=1`-Slot. Neu ist nur der Hinweis, der genau das sagt: Der Dienst wird
  dadurch manuell gepflegt und von der automatischen Regeneration nicht mehr angefasst.

## Capabilities

### Modified Capabilities

- `duties`: Eine dynamisch definierte Dienst-Dauer fällt nicht mehr auf die absolute
  zurück; unauflösbare Kombinationen werden beim Pflegen abgewiesen und beim Regen
  gemeldet statt überdeckt.

## Test-Anforderungen

| Route / Pfad | Testname | Erwartung / Invariante |
|---|---|---|
| `POST /api/duty-types` | `TestCreateType_UnmoeglicheSpanneWirdAbgewiesen` | Gleicher Anker, End-Versatz ≤ Start-Versatz, Modus `dynamisch` → 400, **kein** Diensttyp angelegt. |
| `PUT /api/duty-types/{id}` | `TestUpdateType_UnmoeglicheSpanneWirdAbgewiesen` | Wie oben → 400; der Bestand (inkl. mitgesendetem Namen) bleibt unverändert. |
| `POST /api/duty-types` | `TestCreateType_UnmoeglicheSpanneNurImDynamischenModus` | Dieselben Anker/Versätze im Modus `absolut` → 201; die Endfelder sind dort bedeutungslos. |
| `PUT /api/duty-templates/{id}` | `TestUpdateTemplate_UnmoeglicheSpanneWirdAbgewiesen` | Item mit unmöglicher Spanne → 400, **keine** Item-Zeile geschrieben (Prüfung vor `BeginTx`). |
| — (Regen) | `TestRegen_UnaufloesbareDynamischeDauerErzeugtKeinenSlot` | Auflösung ≤ 0 → kein `duty_slots`-Insert, kein Rückfall auf `hours_value`. |
| — (Regen) | `TestRegen_UnaufloesbareDynamischeDauerStehtInDerZusammenfassung` | Derselbe Fall erscheint als `invalid_span`-Eintrag mit Datum und Diensttyp. |
| — (Regen) | `TestRegen_DynamischeDauerFolgtSpieldauer` (Bestand) | Der gültige Pfad bleibt unverändert. |
| Vitest `AdminDutyTypesPage` | `blockiert Speichern bei unmöglicher Spanne` | Gleicher Anker + End-Versatz ≤ Start-Versatz → kein `PUT`, Fehlermeldung sichtbar. |
| Vitest `AdminDutyTypesPage` | `zeigt im Modus Startzeit+Endzeit kein Dauer-Feld` | Das Dauer-Eingabefeld ist nicht im DOM. |
| Vitest `AdminDutyTemplatesPage` | `blockiert Speichern bei unmöglicher Spanne` | Wie oben je Vorlagen-Zeile. |

**Garantierte Invariante:** Ein Slot trägt nach jedem Regen-Lauf eine Dauer > 0 — neu
dadurch, dass ein Slot ohne positive Dauer **gar nicht erst entsteht** und der Ausfall
gemeldet wird, nicht mehr durch eine ersatzweise eingesetzte Zahl.

## Impact

- `internal/games/regen.go` — `resolveSlotHours` liefert die reine Differenz; `regenGameItems`
  überspringt das Item bei ≤ 0 und schreibt einen `InvalidSpanEntry` in die Summary.
- `internal/games/handler.go` — Item-Validierung um die Spannen-Prüfung (vor `BeginTx`).
- `internal/duties/handler.go` — dieselbe Prüfung in `CreateType`/`UpdateType`.
- `web/src/pages/AdminDutyTypesPage.tsx`, `web/src/pages/AdminDutyTemplatesPage.tsx` —
  Maske neu geordnet, Modus-Beschriftungen, clientseitige Prüfung.
- `web/src/lib/duration.ts` — `dynamicSpanImpossible()` als geteilte Regel für beide Masken.
- `web/src/components/RegenSummaryCard.tsx` — neue Zeile für `invalid_span`.
- `web/src/components/SpieltagDetailModal.tsx` — Hinweistext „wird dadurch manuell
  gepflegt"; die Vorbelegung aus einem dynamischen Typ bleibt.
- **Unberührt:** Schema (keine Migration), `duty_slots`, `restoreAssignments`/`makeCustomKey`,
  `duty_accounts.ist`.

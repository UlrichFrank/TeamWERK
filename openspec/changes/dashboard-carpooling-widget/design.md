## Context

Das Dashboard liefert via `GET /api/dashboard` u.a. ein `CarpoolingHint`-Objekt. Dieses enthält aktuell `recentEvents` (Event-Verlauf der letzten 48h) und nur aggregierte Zähler (`bieteCount`, `sucheCount`) — aber keine Namen der offenen Einträge. Das führt zu einem unübersichtlichen Widget mit redundanten Informationen und einem „Invalid Date"-Bug.

## Goals / Non-Goals

**Goals:**
- Backend: `recentEvents` entfernen, `openEntries`-Query hinzufügen
- Frontend: Widget neu strukturieren — kompakte Spielzeile, prominente Paarungen, Namensliste offener Einträge

**Non-Goals:**
- Keine Änderungen an der Mitfahrseite (`/mitfahrgelegenheiten`)
- Keine neuen API-Endpunkte — nur der bestehende `/api/dashboard` ändert sich
- Keine Paginierung oder Sortieroptionen für `openEntries`

## Decisions

### 1. `openEntries` statt Event-Feed

**Entscheidung:** `recentEvents []CarpoolingEvent` wird durch `openEntries []CarpoolingOpenEntry` ersetzt.

**Rationale:** Der Event-Feed zeigte Verlaufsrauschen (wer hat wann ein Gesuch erstellt) — für den Nutzer irrelevant im Dashboard-Kontext. Was zählt: wer ist aktuell offen für Mitfahrt, und mit wem bin ich bereits gematcht.

**Alternativen:** Event-Feed behalten aber filtern (nur `pairing_requested` zeigen) — abgelehnt, weil dasselbe in der bestätigten Paarung sichtbar ist und die Duplikation bleibt.

### 2. Variante B für „offen"

**Entscheidung:** Ein Eintrag gilt als offen, solange er kein `confirmed`-Pairing hat. Einträge mit `pending`-Pairing erscheinen ebenfalls in der Liste.

**Rationale:** `pending` bedeutet, dass die Anfrage noch nicht bestätigt wurde — die Person ist noch verfügbar. Nur `confirmed` schließt jemanden aus der offenen Liste aus.

### 3. Limit 5 + Zähler

**Entscheidung:** `openEntries` liefert maximal 5 Einträge. `bieteCount` und `sucheCount` bleiben als Gesamtzähler für die „+ X weitere"-Anzeige im Widget.

**Rationale:** Das Dashboard-Widget ist kompakt. Mehr als 5 Namen wären unlesbar. Die Zähler ermöglichen eine korrekte „+ X weitere"-Anzeige ohne alle Einträge zu laden.

### 4. Query-Scope: alle anderen, nicht nur komplementärer Typ

**Entscheidung:** `openEntries` zeigt alle offenen Einträge anderer Nutzer (Biete + Suche), nicht nur den zum eigenen Typ komplementären.

**Rationale:** Nutzer ohne eigenen Eintrag sollen ebenfalls sehen, was los ist. Ein Bieter kann auch sehen, wer noch sucht (und umgekehrt). Die vollständige Information gehört in das Widget.

## Risks / Trade-offs

- **Breaking API-Änderung:** Das `carpoolingHint`-Objekt im Dashboard-Response ändert sich. Da Frontend + Backend im selben Repo deployt werden, kein Risiko durch inkonsistente Clients.
- **N+1 vermieden:** Die `openEntries`-Query ist ein einzelner JOIN — kein separater Query pro Eintrag.
- **`myEntry` Sichtbarkeit:** `myEntry` bleibt im Modell und wird als kleine Statuszeile gerendert, damit Nutzer wissen ob sie bereits eingetragen sind.

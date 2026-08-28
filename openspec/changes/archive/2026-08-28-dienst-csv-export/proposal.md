## Why

Wer einen Heimspieltag organisiert, arbeitet heute gegen den Bildschirm: die Dienste
eines Zeitraums stehen in der Dienstbörse und im Spieltag-Detail-Modal, aber nirgends in
einer Form, die man ausdrucken, weiterreichen oder in Excel gegenprüfen kann. Genau das
ist der Bedarf — und zwar zusammen mit dem Kontext, der die Dienstlage überhaupt
erklärt: **wer den Spieltag ausrichtet** und **ob die Nachbarschaft eine Rolle spielt**
(mehrere Spiele am selben Tag, Heimspiel am Vor- oder Folgetag).

Dieser Kontext ist im System vorhanden, aber über drei Orte verstreut: der Ausrichter
hängt am Spieltag (`spieltag_ausrichter`, Auflösung in `settings.ResolveAusrichterForDay`),
die Nachbarschaftsregeln am Diensttyp (`duty_types.same_day_behavior`,
`adjacent_day_behavior`), die Tageskonstellation an den Spielen selbst. Beim Prüfen
„stimmt der Dienstplan für dieses Wochenende?" muss man ihn heute im Kopf
zusammensetzen.

## What Changes

- Neue Route `GET /api/duty-slots/export?from=&to=` liefert eine CSV (`;`-getrennt,
  UTF-8 mit BOM) mit einer Zeile je Dienst-Slot im Zeitraum.
- Jede Zeile trägt neben den Dienst-Zeiten (Beginn, Ende aus Beginn + `hours_value`,
  Dauer) den Termin-Kontext: Tages-Ausrichter (inkl. „für diesen Tag explizit gesetzt"),
  Spiele am Tag, Anwurfzeiten des Tages, Heimspiel am Vor-/Folgetag sowie die am
  Diensttyp eingestellte Regel für beide Nachbarschaftsfälle.
- **Ohne Belegung und ohne Namen** — keine `slots_filled`-Spalte, keine Zugewiesenen.
  Das Blatt trägt damit keine personenbezogenen Daten und darf an Ausrichter ohne
  TeamWERK-Zugang weitergegeben werden.
- Der Export **beschreibt** die Tageskonstellation, er **rechnet sie nicht nach**: die
  Entscheidung „entfällt/reduziert" bleibt allein in `internal/games/regen.go`
  (`applyBehavior`). Ausgegeben werden die Eingangsgrößen plus die konfigurierte Regel —
  kein zweiter, driftender Nachbau.
- Frontend: Menüpunkt „Dienste als CSV" im Aktionsmenü neben „+ Event" auf `/kalender`,
  dahinter ein kleiner Zeitraum-Dialog, vorbelegt mit dem angezeigten Monat.

## Capabilities

### New Capabilities

- `dienst-csv-export`: Dienst-Planungssicht als CSV über einen wählbaren Zeitraum

### Modified Capabilities

- `permissions`: `GET /api/duty-slots/export` im Gate
  `RequireClubFunction("vorstand","trainer","sportliche_leitung")`

## Impact

- `internal/duties/export_handler.go` (neu) — Handler, Spalten, Tages-Kontext
- `internal/app/router.go` — Route im Tier der Dienst-Slot-Pflege
- `internal/permissions/matrix_test.go`, `openspec/specs/permissions/spec.md` — Matrix-Eintrag
- `web/src/components/DutyExportModal.tsx` (neu), `web/src/pages/KalenderPage.tsx` — Menüpunkt + Dialog
- Tests: `internal/duties/export_handler_test.go`,
  `web/src/components/DutyExportModal.test.tsx`,
  `web/src/pages/__tests__/KalenderPage.permissions.test.tsx`

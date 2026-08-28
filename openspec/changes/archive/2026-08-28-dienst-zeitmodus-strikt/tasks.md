# Tasks

## 1. Validierung: unmögliche Spanne (Backend)

- [x] 1.1 `internal/duties/handler.go` — Helfer `dynamicSpanImpossible(anchor, offset, endAnchor, endOffset)` +
      Prüfung in `CreateType` und `UpdateType`, vor jedem Schreibvorgang, HTTP 400.
- [x] 1.2 `internal/games/handler.go` — dieselbe Prüfung je Item in `UpdateTemplate`,
      vor `BeginTx`, `{"error":"impossible_duration_span"}`.
- [x] 1.3 Tests: `TestCreateType_UnmoeglicheSpanneWirdAbgewiesen`,
      `TestUpdateType_UnmoeglicheSpanneWirdAbgewiesen`,
      `TestCreateType_UnmoeglicheSpanneNurImDynamischenModus`,
      `TestUpdateTemplate_UnmoeglicheSpanneWirdAbgewiesen`.

## 2. Regen: Rückfall entfernen, Ausfall melden

- [x] 2.1 `internal/games/regen.go` — `resolveSlotHours` liefert die reine Differenz
      (0 als Signal), `regenGameItems` überspringt das Item bei ≤ 0.
- [x] 2.2 `RegenSummary.InvalidSpan []InvalidSpanEntry` (`invalid_span`) + Eintrag beim Überspringen.
- [x] 2.3 Tests: `TestRegen_UnaufloesbareDynamischeDauerErzeugtKeinenSlot`,
      `TestRegen_UnaufloesbareDynamischeDauerStehtInDerZusammenfassung`;
      `TestRegen_DynamischeDauerFaelltAufAbsoluteZurueck` ersetzen.

## 3. Frontend: geteilte Regel + Zusammenfassung

- [x] 3.1 `web/src/lib/duration.ts` — `dynamicSpanImpossible()` (Spiegel der Server-Regel) + Vitest.
- [x] 3.2 `web/src/components/RegenSummaryCard.tsx` — `invalid_span` als Fehlerzeile.

## 4. Maske `/diensttypen`

- [x] 4.1 Modus-Beschriftungen „Startzeit + Dauer" / „Startzeit + Endzeit".
- [x] 4.2 Feldreihenfolge: Modus → Start-Anker + Start-Versatz → Dauer bzw. End-Anker + End-Versatz.
- [x] 4.3 Kein Dauer-Feld im dynamischen Modus; Speichern bei unmöglicher Spanne blockiert.
- [x] 4.4 Vitest: kein Dauer-Feld, blockiertes Speichern, Beschriftungen.

## 5. Maske `/dienstplan-vorlagen`

- [x] 5.1 Item-Maske ab „Anker" in dieselbe Form gebracht (Start-Anker/-Versatz oben,
      Dauer bzw. End-Felder darunter), Dauer aus der oberen Dreier-Zeile heraus.
- [x] 5.2 Speichern bei unmöglicher Spanne blockiert, Meldung je Eintrag.
- [x] 5.3 Vitest anpassen/ergänzen.

## 6. Termin-Dialog `/kalender`

- [x] 6.1 Hinweis „manuell gepflegt / nicht mehr automatisch regeneriert" in Anlege- und
      Bearbeiten-Dialog.
- [x] 6.2 Vitest.

## 7. Abschluss

- [x] 7.1 `docs/agent/06-gotchas.md` — Gotcha-Absatz zum Dauer-Modus aktualisieren/ergänzen.
- [ ] 7.2 `/verify-change` (Build/Test/Lint + Invarianten), dann archivieren.

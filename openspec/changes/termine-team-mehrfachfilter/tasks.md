## 1. Fokus endet bei aktiver Filteränderung

- [x] 1.1 `web/src/pages/TerminePage.tsx` — `updateFilter` löscht `focus`, wenn der Patch `team` oder `types` enthält und `focus` nicht selbst gesetzt wird; Kommentar nennt den Grund (Durchlass gilt dem Deep-Link, nicht der Sitzung)
- [x] 1.2 Prüfen, dass die Fokus-eigene `past`-Umschaltung (`triedPastExpansion`) davon unberührt bleibt

## 2. Team-Filter als Mehrfachauswahl

- [x] 2.1 `web/src/hooks/useDismissOnOutside.ts` — Schließ-Logik (mousedown/touchstart außerhalb, Scroll) aus `EventTypeFilter` extrahieren; `EventTypeFilter` auf den Hook umstellen (verhaltensgleich)
- [x] 2.2 `web/src/components/TeamFilter.tsx` — Dropdown mit Checkboxen, Compact-Modus wie `EventTypeFilter` (Icon + Zähler), Label sonst „Teams" / Kurzname / „n Teams"
- [x] 2.3 `TerminePage.tsx` — `team` als Menge parsen (CSV, Einzel-ID bleibt gültig), Filterprädikat auf „mindestens eine der gewählten Mannschaften", `updateFilter` schreibt CSV bzw. löscht den Parameter bei leerer/vollständiger Auswahl
- [x] 2.4 `<select>` in der Kopfzeile durch `TeamFilter` ersetzen; das `hidden sm:block` entfällt (Mobile bedienbar), Kommentar dazu aktualisieren
- [x] 2.5 `otherFiltersActive` / `resetFilters` auf die Menge umstellen

## 3. Tests

- [x] 3.1 `team=1,2` zeigt beide Mannschaften, dritte nicht; `team=1` bleibt gültig
- [x] 3.2 Abwählen einer Mannschaft schreibt die restlichen in die URL
- [x] 3.3 Abwählen aller / Anhaken aller entfernt `team` aus der URL und zeigt alles
- [x] 3.4 Team- bzw. Typ-Filteränderung entfernt `focus` aus der URL und blendet den zuvor fokussierten Termin aus
- [x] 3.5 Fokus aus der URL überlebt den ersten Render mit Filter (Bestandsverhalten)

## 4. Abschluss

- [x] 4.1 `pnpm -C web test` + `lint` + `build` grün
- [x] 4.2 `openspec validate termine-team-mehrfachfilter --strict`

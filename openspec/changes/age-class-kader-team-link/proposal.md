## Why

`teams.age_class` und `kader.age_class` sind freie Texte (z.B. `'B-Jugend'`), während `age_class_game_rules` Kurzform-Keys (`'A'`–`'D'`) verwendet — ein struktureller Bruch, der die Slot-Zeitberechnung mit einem Workaround am Laufen hält. Es gibt keine FK-Constraint, die verhindert, dass ungültige Altersklassen eingetragen werden, und die Admin-UI für Teams bietet kein Dropdown aus den tatsächlich konfigurierten Klassen.

## What Changes

- `age_class_game_rules` wird auf Langform-Keys umgestellt: `'A-Jugend'`, `'B-Jugend'`, `'C-Jugend'`, `'D-Jugend'`
- `teams.age_class` erhält einen FK-Constraint auf `age_class_game_rules.age_class` (nullable — erwachsene Teams haben keine Jugendklasse)
- Die Admin-UI für Teams bietet ein Dropdown mit den verfügbaren Altersklassen aus der DB
- Die Altersklassen-Regeln-UI entfernt das manuelle `-Jugend`-Suffix, weil der Key bereits vollständig ist
- Der Workaround-Code in `effectiveEventDuration` (Buchstaben-Extraktion per `[:1]`) entfällt
- `kader.age_class` bleibt freier Text (kein FK), weil Kader auch aus Datenimporten (Saison-Kopie, CSV) stammen und die Prüfung dort separat gehandhabt wird

## Capabilities

### New Capabilities
- `age-class-canonical-values`: Altersklassen-Werte als kanonische Liste — `age_class_game_rules` ist die Single Source of Truth; `teams.age_class` ist FK darauf; Admin-Dropdown für Teams

### Modified Capabilities
- `games`: `effectiveEventDuration` nutzt den direkten Wert aus `teams.age_class` ohne Transformation

## Impact

- **DB-Migration**: Umbenennung der `age_class_game_rules`-Keys (A → A-Jugend usw.) + FK auf `teams.age_class`; bestehende Teamdaten bleiben unverändert (sie verwenden bereits Langform)
- **Backend**: `internal/config/handler.go` — `validAgeClasses`-Map auf Langform; `internal/games/handler.go` — Workaround `[:1]` entfernen
- **Frontend**: `AdminAgeClassRulesPage.tsx` — kein manuelles `-Jugend`-Anhängen; `AdminDutyTypesPage`/Teams-Admin — Dropdown für `age_class`
- **Kein Breaking Change** an der API: `age_class`-Felder in JSON-Responses waren schon immer der Rohwert aus der DB

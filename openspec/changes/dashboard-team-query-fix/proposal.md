## Why

Die Dashboard-Funktion `teamQueryForUser(role string)` ermittelt Team-IDs für Spielplan, Fahrtgemeinschaft und Fahrzeug-Action rollenspezifisch — dabei fehlt für `trainer` der `family_links`-Pfad. Ein Trainer, dessen Teamzugang nur über Kinder (family_links) besteht, sieht auf dem Dashboard weder Spiele noch Fahrtgemeinschaften.

**Aktueller Code-Stand (nach role-model-refactor):** `role` wird nicht mehr direkt aus `claims.Role` gelesen, sondern über `effectivePersona(claims.ClubFunctions, claims.IsParent)` auf einen einzelnen String reduziert (Priorität: trainer > vorstand > spieler > elternteil). `teamQueryForUser` nimmt nach wie vor diesen einzelnen String — der `family_links`-Pfad fehlt für `trainer` weiterhin. Task 3.1 aus `role-model-refactor` (`teamQueryForUser` auf `(clubFunctions []string, isParent bool)` umstellen) ist als erledigt markiert, wurde aber nicht implementiert.

Das View `user_accessible_teams` (seit `001_baseline` in der DB) vereint alle Pfade korrekt und wird bereits im Carpooling-List-Handler verwendet.

## What Changes

- `teamQueryForUser(role string)` in `internal/dashboard/handler.go` wird entfernt
- `queryCarpoolingHint`, `queryNextGames` und `vehicleAction` verwenden stattdessen direkt `user_accessible_teams` als Subquery (analog zum Carpooling-List-Handler)
- Early-Return für `admin`/`vorstand` bleibt erhalten
- `memberDutyActions` prüfen, ob dasselbe Muster (`ds.team_id IN (...)`) ebenfalls betroffen ist und ggf. anpassen
- Task 3.1 aus `role-model-refactor` gilt damit als abgedeckt (durch den `user_accessible_teams`-Ansatz, nicht durch Signaturänderung)

## Capabilities

### New Capabilities

_(keine neuen Capabilities — rein interner Fix)_

### Modified Capabilities

- `dashboard-team-filter`: Die Logik zur Ermittlung der sichtbaren Teams auf dem Dashboard wird vereinheitlicht — alle Rollen verwenden `user_accessible_teams` statt rollenspezifischer Subqueries.

## Impact

- **Datei:** `internal/dashboard/handler.go`
- **Entfernt:** Methode `teamQueryForUser`
- **Geändert:** `queryCarpoolingHint`, `queryNextGames`, `vehicleAction` (und ggf. `memberDutyActions`)
- **Überschneidung:** Task 3.1 in `role-model-refactor/tasks.md` nach Abschluss als obsolet markieren
- **Kein API-Schema-Änderung**, kein Frontend-Change, keine neue Migration nötig (View existiert bereits)

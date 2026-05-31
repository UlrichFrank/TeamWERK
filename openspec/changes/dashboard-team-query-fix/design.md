## Context

Das Dashboard ermittelt team-spezifische Daten (Spiele, Fahrtgemeinschaft-Hint, Fahrzeug-Action, offene Dienste) über die Hilfsmethode `teamQueryForUser(role string)`. Diese gibt je nach Rolle eine SQL-Subquery zurück:

| Rolle | Pfad |
|-------|------|
| `spieler` | `team_memberships` via `members.user_id` |
| `elternteil` | `team_memberships` via `family_links` |
| `trainer` | `kader_trainers` via `members.user_id` |
| `admin`, `vorstand` | leer → kein Dashboard-Abschnitt |

**Aktueller Code-Stand:** Nach `role-model-refactor` kommt `role` nicht mehr direkt aus `claims.Role`, sondern aus `effectivePersona(claims.ClubFunctions, claims.IsParent)` — einem Prioritätsmapping (trainer > vorstand > spieler > elternteil). `teamQueryForUser` nimmt weiterhin diesen einzelnen String. Das bedeutet: ein Nutzer mit `clubFunctions=["trainer"]` und `isParent=true` bekommt `role="trainer"` → nur der `kader_trainers`-Pfad wird geprüft, `family_links` bleibt ignoriert.

Task 3.1 aus `role-model-refactor` sollte `teamQueryForUser` auf `(clubFunctions []string, isParent bool)` umstellen — ist als `[x]` markiert, aber nicht im Code vorhanden.

Das View `user_accessible_teams` (bereits in der DB, angelegt in `001_baseline.up.sql`) vereint alle drei Pfade korrekt und wird bereits im Carpooling-List-Handler (`internal/carpooling/handler.go`) genutzt.

## Goals / Non-Goals

**Goals:**
- `teamQueryForUser` ersetzen durch `user_accessible_teams` in allen betroffenen Dashboard-Queries
- `admin`/`vorstand` sehen weiterhin keine team-spezifischen Sections (Early-Return bleibt)
- Verhalten für `spieler` und `elternteil` bleibt identisch (nur die interne Subquery ändert sich)

**Non-Goals:**
- Keine Änderung am Frontend
- Keine neue DB-Migration
- Keine Änderung an `memberDutyActions` (nutzt `ds.team_id`, muss separat geprüft werden — evtl. eigener Change)

## Decisions

**`user_accessible_teams` statt rollenspezifische Subquery**

Alle drei Funktionen (`queryCarpoolingHint`, `queryNextGames`, `vehicleAction`) ersetzen das `fmt.Sprintf`-Pattern durch eine statische Query mit der Subquery:
```sql
SELECT team_id FROM user_accessible_teams WHERE user_id = ? AND season_id = ?
```
Parameter-Reihenfolge bleibt identisch (userID, seasonID für Subquery + seasonID für äußere Query).

Alternative verworfen: `teamQueryForUser` um einen `family_links`-Pfad für `trainer` erweitern — das würde die Logik weiter aufsplitten statt zu konsolidieren.

**`memberDutyActions` vorerst ausklammern**

Diese Funktion wird nur für `elternteil` und `spieler` aufgerufen (Switch in `buildActions`) — der `trainer`-Bug trifft sie nicht. Zudem filtert sie über `ds.team_id` (Duty Slots), nicht über `game_teams`. Separate Betrachtung empfohlen.

## Risks / Trade-offs

- [`user_accessible_teams` gibt Duplikate zurück (UNION ALL)] → `DISTINCT` oder `IN`-Operator absorbiert Duplikate, kein Problem
- [View existiert nur in Baseline-Migration, könnte auf älteren DBs fehlen] → View ist seit `001_baseline` vorhanden, kein Risiko auf Prod
- [Verhaltensänderung für Trainer mit kader_trainers-Einträgen] → keiner, da `user_accessible_teams` auch `kader_trainers` enthält

## Migration Plan

Kein Schema-Change. Reiner Code-Change, direkt deploybar via `make deploy`.

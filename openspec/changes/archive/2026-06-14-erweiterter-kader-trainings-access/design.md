## Context

`player_memberships` ist eine View über `kader_members` (Hauptkader). An drei Stellen in `internal/trainings/handler.go` wird diese View verwendet, obwohl `kader_extended_members` seit Migration 021 existiert. Bei Spielen (`internal/games/handler.go`) wurde die gleiche Lücke bereits durch eine UNION auf `kader_extended_members` geschlossen (Zeile 1580-1583). Trainings hinken nach.

## Goals / Non-Goals

**Goals:**
- Erweiterter Kader hat denselben Zugang zu Trainings wie Hauptkader (Sichtbarkeit, Anwesenheitsliste, Benachrichtigungen bei neuen Terminen)
- Implementierung analog zum bestehenden Muster in `games/handler.go`

**Non-Goals:**
- Retroaktive Benachrichtigungen wenn ein Spieler nachträglich zum Kader hinzugefügt wird
- Änderungen am Frontend
- Änderungen an anderen Domänen (games, duties, etc.)

## Decisions

**UNION statt View-Erweiterung**

`player_memberships` wird nicht geändert (andere Stellen im Code nutzen sie bewusst nur für Hauptkader). Stattdessen wird an den drei betroffenen Stellen direkt eine UNION auf `kader_extended_members` ergänzt — exakt wie in `games/handler.go`. Das hält die Änderung lokal und reversibel.

**Drei Änderungspunkte:**

1. `teamMembersAndParents()` (~Zeile 32): Benachrichtigungsempfänger
   ```sql
   UNION
   SELECT DISTINCT m.user_id FROM members m
   JOIN kader_extended_members kem ON kem.member_id = m.id
   JOIN kader k ON k.id = kem.kader_id
   JOIN seasons s ON s.id = k.season_id AND s.is_active = 1
   WHERE k.team_id = ? AND m.user_id IS NOT NULL
   ```

2. `ListTrainingSessions` Spieler-Condition (~Zeile 701): Sichtbarkeit
   ```sql
   ts.team_id IN (
     SELECT DISTINCT tm.team_id FROM player_memberships tm
     JOIN members m ON m.id = tm.member_id WHERE m.user_id = ?
     UNION
     SELECT DISTINCT k.team_id FROM kader_extended_members kem
     JOIN kader k ON k.id = kem.kader_id
     JOIN members m2 ON m2.id = kem.member_id WHERE m2.user_id = ?
   )
   ```

3. `GetAttendances` (~Zeile 1081): Anwesenheitsliste
   ```sql
   FROM members m
   WHERE (
     EXISTS (SELECT 1 FROM player_memberships pm WHERE pm.member_id = m.id AND pm.team_id = ? AND pm.season_id = ?)
     OR EXISTS (SELECT 1 FROM kader_extended_members kem JOIN kader k ON k.id = kem.kader_id WHERE kem.member_id = m.id AND k.team_id = ? AND k.season_id = ?)
   )
   ```

## Risks / Trade-offs

- [Risiko] Doppelte Einträge wenn ein Spieler im Haupt- UND Erw.-Kader steht → `DISTINCT` in den UNION-Queries schützt dagegen
- [Risiko] `teamMembersAndParents()` für Erw.-Kader-Spieler ohne `user_id` (kein Account) → `WHERE m.user_id IS NOT NULL` Filter verhindert NULL-Einträge in der Empfängerliste

## Context

Der `team_memberships`-View in `001_baseline.up.sql` (Zeile 399) kombiniert über UNION sowohl `kader_members` (Spieler) als auch `kader_trainers` (Trainer). Dieser View wird an 14 Stellen im Go-Backend genutzt — für sehr unterschiedliche Zwecke:

| Nutzungskontext | Datei | Gewünschte Menge |
|---|---|---|
| Teilnahmeliste Trainingseinheit | `trainings/handler.go:945` | **nur Spieler** |
| RSVP-opt-out-Zählung | `trainings/handler.go:621,722` | **nur Spieler** |
| Sichtbarkeit Termine für Spieler | `trainings/handler.go:601` | nur Spieler |
| Sichtbarkeit Termine für Eltern | `trainings/handler.go:598` | nur Spieler (Kinder) |
| Dienstslot-Zielgruppe | `duties/handler.go:286,298` | nur Spieler |
| Mitgliederliste | `members/handler.go:139,165` | nur Spieler |
| Scheduler-Erinnerungen | `scheduler/scheduler.go:148,172` | nur Spieler |
| Spielzugänglichkeit | `games/handler.go:1556,1563,1581` | alle (Spieler + Trainer) |

Der bestehende View war für Zugriffskontrolle konzipiert (wer darf ein Team sehen?). Für listenbildende Queries — wer nimmt teil, wer gehört zum Kader — ist er zu breit.

## Goals / Non-Goals

**Goals:**
- Zwei neue, semantisch klare Views: `player_memberships` (nur `kader_members`) und `trainer_memberships` (nur `kader_trainers`)
- Bug-Fix: Trainer erscheinen nicht mehr in der Trainings-Teilnahmeliste
- Alle bisherigen Nutzungen von `team_memberships` auf den passenden View umstellen
- Kein Breaking Change: `team_memberships` und `user_accessible_teams` bleiben unverändert erhalten

**Non-Goals:**
- Änderungen an der Berechtigungslogik (`hasTeamAccess`, `user_accessible_teams`)
- Umstrukturierung des `kader`-Datenmodells
- Frontend-Änderungen
- API-Änderungen

## Decisions

### 1. Neue Views statt View-Änderung

**Entscheidung:** `team_memberships` bleibt bestehen; zwei neue Views ergänzen ihn.

**Alternativen:**
- `team_memberships` auf Spieler reduzieren + neue View für Trainer → bricht alle Nutzungen in `games/`, die Trainer einschließen sollen
- Nur `GetAttendances` per Direktjoin gegen `kader_members` fixen (Option A) → schneller, aber technische Schuld bleibt in allen anderen Queries

**Rationale:** Explizite Views machen die Semantik für künftige Entwickler sofort sichtbar. Einmaliger Mehraufwand, dauerhafter Gewinn.

### 2. `team_memberships` für `games`-Handler beibehalten

**Entscheidung:** `games/handler.go` nutzt weiterhin `team_memberships` (Spieler + Trainer), da Trainer auch Spielzugänglichkeit benötigen.

**Rationale:** Die Queries dort bestimmen Sichtbarkeit von Spielen — Trainer sollen Spiele ihres Teams sehen. Das ist korrektes Verhalten.

### 3. Migration: nur neue Views anlegen, kein DROP

**Entscheidung:** Die Migration legt `player_memberships` und `trainer_memberships` via `CREATE VIEW` an. Kein `DROP VIEW team_memberships`.

**Rationale:** SQLite unterstützt kein `ALTER VIEW`; ein versehentliches Drop würde `user_accessible_teams` und evtl. unbekannte Stellen brechen. Die neuen Views sind additiv.

## Risks / Trade-offs

**[Risk] Übersehene Nutzungen von `team_memberships`** → Mitigation: `grep -rn "team_memberships"` vor Merge als Abnahmekriterium; alle bekannten Stellen (14) sind im Design dokumentiert.

**[Risk] `rsvp_opt_out`-Zählung zählt bisher auch Trainer mit** → Mitigation: Umstellen auf `player_memberships` korrigiert die Zählung; Zahl kann leicht sinken wo Trainer im Kader waren.

**[Trade-off] Zwei Views mehr im Schema** → akzeptabel; SQLite-Views haben keinen Laufzeit-Overhead.

## Migration Plan

1. Neue Migration `017_split_membership_views.up.sql`:
   ```sql
   CREATE VIEW player_memberships AS
   SELECT km.id, km.member_id, k.team_id, k.season_id
   FROM kader_members km
   JOIN kader k ON k.id = km.kader_id
   WHERE k.team_id IS NOT NULL;

   CREATE VIEW trainer_memberships AS
   SELECT kt.kader_id * 100000 + kt.member_id AS id, kt.member_id, k.team_id, k.season_id
   FROM kader_trainers kt
   JOIN kader k ON k.id = kt.kader_id
   WHERE k.team_id IS NOT NULL;
   ```
2. Down-Migration: `DROP VIEW IF EXISTS player_memberships; DROP VIEW IF EXISTS trainer_memberships;`
3. Go-Queries auf `player_memberships` umstellen (alle außer `games/handler.go`)
4. `make migrate-up` lokal testen, Build-Fehler beheben, deployen

## Open Questions

*(keine)*

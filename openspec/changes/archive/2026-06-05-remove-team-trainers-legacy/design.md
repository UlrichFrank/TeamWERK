## Context

Die `team_trainers`-Tabelle (`team_id, user_id`) ist eine direkte Verknüpfung ohne Saisonbezug. Sie entstand vor der Kader-Verwaltung und wurde nie vollständig entfernt. Das korrekte Modell ist `kader_trainers` (`kader_id, member_id`), das Trainer als *Vereinsmitglieder* einem saisonbezogenen Kader zuordnet.

Aktuell nutzen 5 Dateien `team_trainers`:

| Datei | Verwendung |
|---|---|
| `internal/trainings/handler.go` | `hasTeamAccess`, `ListSeries`-Filter, `ListSessions`-Filter |
| `internal/members/handler.go` | Mitgliederliste für Trainer einschränken |
| `internal/auth/handler.go` | Trainer-E-Mails bei Beitrittsantrag |
| `internal/scheduler/scheduler.go` | Duty-Reminder an Trainer |
| `internal/config/handler.go` | `AssignTrainer`-Handler (Insert) |

Außerdem enthält der Scheduler-Query `WHERE u.role = 'trainer'` — nach Migration 002 gibt es diese Rolle nicht mehr (`admin|standard`), d.h. Trainer-Reminder werden schon jetzt nie verschickt. Das wird im gleichen Zug behoben.

## Goals / Non-Goals

**Goals:**
- Alle Zugriffe auf `team_trainers` durch semantisch äquivalente `kader_trainers`-Queries ersetzen
- `team_trainers`-Tabelle per Migration droppen
- `AssignTrainer`-Handler und Route entfernen
- Broken Scheduler-Query (role='trainer') reparieren

**Non-Goals:**
- Kein UI-Change für die Trainer-Zuweisung (bleibt über Kader-Verwaltung)
- Keine Änderung der Zugriffsrechte — wer vorher Zugriff hatte, hat ihn danach weiterhin
- Keine Migration existierender `team_trainers`-Daten (Tabelle wird einfach gedroppt; Daten waren ohnehin nicht maßgeblich)

## Decisions

### SQL-Pattern für Trainer-Team-Zugriff

Einheitliches Subquery-Pattern überall:

```sql
-- Prüft ob user_id Trainer eines bestimmten Teams ist (in irgendeinem Kader/Saison)
SELECT COUNT(*) FROM kader_trainers kt
JOIN kader k ON k.id = kt.kader_id
JOIN members m ON m.id = kt.member_id
WHERE m.user_id = ? AND k.team_id = ?

-- Alle team_ids eines Trainers (für IN-Clauses):
SELECT DISTINCT k.team_id FROM kader k
JOIN kader_trainers kt ON kt.kader_id = k.id
JOIN members m ON m.id = kt.member_id
WHERE m.user_id = ?
```

**Entscheidung: kein Saison-Filter in `hasTeamAccess`.**  
Begründung: Trainer sollen auch Trainings aus vergangenen Saisons einsehen können. `ListTeamsForUser` filtert bereits auf die aktive Saison für die Dropdown-Befüllung — das ist ausreichend.

### Scheduler-Fix (role = 'trainer')

Der kaputte Check `u.role = 'trainer'` wird ersetzt durch einen Join auf `member_club_functions`:

```sql
SELECT DISTINCT u.id, u.email, u.first_name || ' ' || u.last_name
FROM users u
JOIN members m ON m.user_id = u.id
JOIN member_club_functions mcf ON mcf.member_id = m.id AND mcf.function = 'trainer'
JOIN kader_trainers kt ON kt.member_id = m.id
JOIN kader k ON k.id = kt.kader_id
WHERE k.team_id = ?
  AND u.duty_reminder_days IS NOT NULL
  AND <not-assigned-check>
```

### `AssignTrainer` komplett entfernen

Der Endpunkt `POST /api/admin/teams/{id}/assign-trainer` hat keine UI mehr und kollidiert mit dem Kader-basierten Modell. Entfernen statt deprecaten — die Route war nie dokumentiert und wird nicht genutzt.

## Risks / Trade-offs

**Kein Rollback der Daten möglich** → Die Migration droppt die Tabelle. Down-Migration kann sie leer wiederherstellen, aber alte Einträge sind weg. Akzeptabel, da `team_trainers` und `kader_trainers` nicht synchron waren und `team_trainers`-Einträge ohnehin unzuverlässig.

**Trainer ohne Kader-Eintrag verlieren Zugriff** → Jeder Trainer, der bisher nur über `team_trainers` (nicht über `kader_trainers`) eingetragen war, verliert den Zugriff auf Training-Verwaltung und Mitgliederlisten. Das ist korrekt — er wäre nie über das offizielle System registriert gewesen.

## Migration Plan

1. Code-Änderungen implementieren (alle 5 Handler-Dateien)
2. Migration `010_drop_team_trainers` anlegen
3. Lokal testen: als Trainer (via kader_trainers) Trainings anlegen und sehen
4. `make migrate-up` lokal + `make deploy` auf VPS

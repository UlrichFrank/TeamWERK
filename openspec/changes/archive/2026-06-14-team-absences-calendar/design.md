## Context

Die Abwesenheits-Infrastruktur existiert bereits vollständig (Migration 030, `member_absences`, `absences_public`-Spalte, `GET /api/absences/calendar`). Zwei Lücken blockieren die Nutzbarkeit:

1. **Bugfix:** `Member`-Struct in `internal/members/handler.go` enthält kein `AbsencesPublic`-Feld; `getMember()` selektiert die Spalte nicht → das Profil-Toggle liest immer `undefined`
2. **Feature:** `GET /api/absences/calendar` gibt fremde Abwesenheiten an alle aus (`OR m.absences_public = 1` ohne Rollen- oder Team-Prüfung) und der Kalender hat keinen Toggle zum Ein-/Ausblenden

Team-Zugehörigkeit im Backend: Die View `user_accessible_teams (user_id, team_id)` ist der einheitliche Zugangspunkt (verwendet in Auth, Chat, Trainings). Mitglieder → Teams: `team_memberships (member_id, team_id, season_id)`.

## Goals / Non-Goals

**Goals:**
- `absences_public` korrekt aus DB lesen und im Profil-Toggle anzeigen (Bugfix)
- Trainer/Sportvorstand/Vorstand/Admin können im Kalender Abwesenheiten ihres Teams einblenden
- Team-Abwesenheiten farblich von eigenen getrennt (brand-blue vs. brand-yellow)
- Personendetails nur per Tooltip und Click, kein Namens-Grid
- Session-basierter Filter-State (sessionStorage), Default AUS

**Non-Goals:**
- Keine Persistenz des Kalender-Filters in DB
- Keine eigene Detailseite für Abwesenheiten — bestehende Detailansicht (InfoModal) wird wiederverwendet
- Kein Export oder Report

## Decisions

### 1. Backend: `show_team=true` als Query-Parameter

`GET /api/absences/calendar?from=&to=&show_team=true[&team_id=X]`

- Ohne `show_team=true`: Verhalten unverändert (nur eigene + Kinder-Abwesenheiten)
- Mit `show_team=true` + unberechtigte Rolle: Parameter wird ignoriert (kein Fehler)
- Mit `show_team=true` + berechtigt: zusätzlich Abwesenheiten von Mitgliedern mit `absences_public = 1`, die in einem der eigenen Teams sind (aktive Saison, via `user_accessible_teams` + `team_memberships`)
- Mit `team_id=X`: Einschränkung auf dieses Team

**Team-Abfrage:**
```sql
AND m.id IN (
  SELECT tm.member_id FROM team_memberships tm
  JOIN seasons s ON s.id = tm.season_id AND s.is_active = 1
  WHERE tm.team_id IN (SELECT team_id FROM user_accessible_teams WHERE user_id = ?)
  [AND tm.team_id = ?]  -- nur wenn team_id übergeben
)
```

### 2. Backend: `is_own`-Flag in der Abwesenheits-Response

Die Abwesenheits-Response bekommt ein `is_own bool`-Feld:
- `true`: Abwesenheit des eingeloggten Users oder seiner Kinder
- `false`: Abwesenheit eines Teammitglieds

Das Frontend verwendet dieses Flag für die Farbgebung — keine clientseitige ID-Vergleichslogik nötig.

### 3. Frontend: Toggle nur sichtbar für berechtigte Rollen

```
user.role === 'admin'
|| user.role === 'trainer'
|| hasFunction(user, 'sportvorstand')
|| hasFunction(user, 'vorstand')
```

Toggle-Zustand in `sessionStorage` unter dem Key `kalender_show_team_absences`. Beim Laden der Seite wird der gespeicherte Wert gelesen (Default `false`).

### 4. Bestehenden Team-Filter mitschicken

Wenn `filterTeamId !== null`, wird `team_id={filterTeamId}` an `GET /api/absences/calendar` angehängt. So greift der vorhandene Team-Dropdown auch für Team-Abwesenheiten.

### 5. Darstellung

- **Eigene** (`is_own: true`): `bg-brand-yellow/20 border-brand-yellow/60` (unverändert)
- **Team** (`is_own: false`): `bg-brand-blue/20 border-brand-blue/60`
- **Tooltip** (`title`-Attribut): `${absence.member_name}: ${type} ${start}–${end}`
- **Click**: öffnet vorhandenes `InfoModal` mit `type: 'absence'` (bereits implementiert)

## Risks / Trade-offs

- **Aktive Saison erforderlich:** Ohne aktive Saison (`seasons.is_active = 1`) werden keine Team-Mitgliedschaften gefunden → keine Team-Abwesenheiten angezeigt. Vertretbar, da der Kalender ohnehin saisonabhängig ist.
- **`user_accessible_teams` ist eine View:** Bei komplexen Vereinsstrukturen (Mehrfachtrainer) kann der JOIN teurer werden. Bei aktuellem Datenvolumen (< 200 Mitglieder) kein Problem.
- **Keine Echtzeitaktualisierung:** SSE-Event `absences` löst `loadAbsences()` aus — das berücksichtigt bereits `show_team`-State, wenn dieser beim Reload mitübergeben wird.

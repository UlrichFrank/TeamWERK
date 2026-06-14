## Context

Die Tabelle `kader_extended_members` existiert seit Migration 021. Der Admin-Flow (Hinzufügen/Entfernen) ist fertig implementiert. Was fehlt: abgesetzte Spieler haben keinen Zugang zum Team und sehen keine Spiele, weil alle relevanten Zugangs-Views und Handler-Queries ausschließlich `kader_members` kennen.

Drei Stellen steuern heute den Zugang:

| Stelle | Datei | Problem |
|---|---|---|
| `user_accessible_teams` View | Migration 018 | kein `kader_extended_members`-Arm |
| `GetRoster` | `internal/teams/handler.go:109` | nur `kader_members` |
| `ListMyGames` Teamfilter | `internal/games/handler.go:1575` | nur `team_memberships` |

`GET /api/games/{id}/participants` ist bereits korrekt (UNION mit `is_extended`-Flag).

## Goals / Non-Goals

**Goals:**
- Abgesetzte Spieler sehen das Team in der Teamauswahl und können die Teamseite öffnen
- Abgesetzte Spieler sehen Spiele ihres erweiterten Teams in `/termine`
- Roster-API liefert `extended_players` als separates Feld
- Auto-Confirm (opt-out) gilt weiterhin nur für reguläre Kader-Mitglieder

**Non-Goals:**
- Trainings-Sichtbarkeit für abgesetzte Spieler (bestehende Spec explizit ausgeschlossen)
- Änderung der `player_memberships`-View (Training-Invariante bleibt erhalten)
- Dienstbörse / Duty-Assignments für abgesetzte Spieler

## Decisions

### 1. View-Update statt Handler-Queries patchen

`user_accessible_teams` wird in einer neuen Migration (022) um einen UNION-Arm für `kader_extended_members` erweitert. Das ist der sauberste Weg: alle Handler, die auf dieser View basieren (ListMyTeams, GetRoster-Zugriffscheck), profitieren automatisch.

**Alternative verworfen:** Pro-Handler-Queries anpassen — würde mehrere Stellen unkoordiniert verändern und die View inkonsistent mit der Realität lassen.

### 2. `extended_players` als eigenes Feld in GetRoster

`GET /api/teams/{id}/roster` bekommt ein neues Feld `extended_players: []PlayerEntry` (gleiche Struktur wie `players`). Das ist kein Breaking Change, da neue Felder additive JSON-Erweiterungen sind.

**Alternative verworfen:** Einzelne `players`-Liste mit `is_extended`-Flag — würde Frontend-Sortierlogik erfordern und mischt zwei semantisch verschiedene Gruppen.

### 3. Auto-Confirm: `in_regular_kader`-Flag im ListMyGames-Query

Statt die `team_memberships`-View zu ändern, wird ein zusätzlicher `EXISTS`-Subquery als neuer SELECT-Column in die `ListMyGames`-Query eingefügt. Er prüft ob der aktuelle Member im regulären Kader eines der Spiel-Teams ist. Go-seitig wird der auto-confirm nur gesetzt wenn `inRegularKader = true`.

```sql
EXISTS(
  SELECT 1 FROM game_teams gt_r
  JOIN kader k_r ON k_r.team_id = gt_r.team_id AND k_r.season_id = g.season_id
  JOIN kader_members km_r ON km_r.kader_id = k_r.id AND km_r.member_id = ?
  WHERE gt_r.game_id = g.id
) AS in_regular_kader
```

Diese Lösung ist präzise: wenn der User in Team A (regulär) und Team B (erweitert) ist und ein Spiel beide Teams hat, greift opt-out (er ist reguläres Mitglied von Team A).

**Alternative verworfen:** `team_memberships`-View erweitern — würde `confirmed_count` bei opt-out-Spielen verfälschen, da dieser Count ebenfalls die View nutzt.

### 4. Frontend: eigener Abschnitt statt Badge

`TermineDetailPage` sortiert extended members unter einen eigenen Heading „Erweiterter Kader" in der Teilnahmetabelle. Das bestehende „Erw."-Badge entfällt zugunsten der visuellen Trennung. `MeinTeamPage` bekommt analog einen „Erweiterter Kader"-Block unter der Spielertabelle.

## Risks / Trade-offs

- **View-Änderung ist für alle Handler sichtbar** → Erweiterte Mitglieder sehen künftig auch alle anderen View-Konsumenten des Teams. Geprüft: `user_accessible_teams` wird nur für Zugangs-Checks und `ListMyTeams` verwendet — beides ist gewünscht.
- **`attachChildrenRSVPToGames`** (Eltern-RSVP-Liste, `games/handler.go:1990`) nutzt ebenfalls `kader_members` direkt. Kinder im erweiterten Kader erscheinen dort nicht — das ist korrekt, da Eltern nur für reguläre Kader-Kinder proxy-RSVP machen.

## Migration Plan

1. **DB Migration 022**: View `user_accessible_teams` mit neuem UNION-Arm deployen (non-destructive, additive)
2. **Backend** deployen (GetRoster + ListMyGames — additive Felder)
3. **Frontend** deployen (neue Sektionen)
4. Rollback: Migration 022 down (View zurücksetzen) — kein Datenverlust

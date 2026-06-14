## Why

Spieler im erweiterten Kader eines Teams (abgesetzt aus einem anderen Team) sind unsichtbar — sie erscheinen weder im Team-Tab auf der Mein-Team-Seite noch in der Spielliste. Sie können daher keine Spielzusage abgeben, obwohl der Trainer sie einplant.

## What Changes

- `GET /api/teams/{id}/roster` gibt eine neue Sektion `extended_players` zurück mit allen Mitgliedern aus `kader_extended_members` des Teams
- `user_accessible_teams` View (DB Migration 022) wird um einen `kader_extended_members`-Arm erweitert, sodass abgesetzte Spieler Team-Zugang und Spielsichtbarkeit erhalten
- `GET /api/games/my` berücksichtigt erweiterte Kader-Mitgliedschaften im Team-Filter
- `GET /api/games/my` wendet die opt-out-Auto-Zusage nur auf reguläre Kader-Mitglieder an — erweiterte Kader-Mitglieder müssen stets explizit zusagen
- `MeinTeamPage`: neuer Abschnitt „Erweiterter Kader" unterhalb der regulären Spieler im Team-Tab
- `TermineDetailPage`: erweiterte Kader-Mitglieder werden visuell als eigene Gruppe unterhalb der regulären Teilnehmer in der Teilnahmetabelle angezeigt

## Capabilities

### New Capabilities

- `erweiterter-kader-sichtbarkeit`: Sichtbarkeit von abgesetzten Spielern auf MeinTeamPage und in der Spielliste; Zugangssteuerung über `user_accessible_teams`

### Modified Capabilities

- `erweiterter-kader`: Ergänzung — abgesetzte Spieler erscheinen in `GET /api/teams/{id}/roster` unter `extended_players` und erhalten Spiel-Sichtbarkeit für ihr erweitertes Team
- `game-rsvp`: Ergänzung — Auto-Confirm (opt-out) gilt nicht für erweiterte Kader-Mitglieder; sie müssen explizit zusagen
- `roster-section-tabs`: Ergänzung — Team-Tab zeigt einen zusätzlichen Abschnitt „Erweiterter Kader" wenn `extended_players` vorhanden

## Impact

- **DB:** neue Migration 022 (View-Update `user_accessible_teams`)
- **Backend:** `internal/teams/handler.go` (GetRoster), `internal/games/handler.go` (ListMyGames)
- **Frontend:** `web/src/pages/MeinTeamPage.tsx`, `web/src/pages/TermineDetailPage.tsx`
- **Keine neuen Abhängigkeiten**, kein Breaking Change an bestehenden API-Feldern

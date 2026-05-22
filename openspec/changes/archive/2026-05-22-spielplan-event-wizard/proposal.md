## Why

Der Spielplan unterstützt heute nur das Anlegen von Heimspielen (admin-only, kein Typ, kein Multi-Team). Trainer können keine Events anlegen, alle anderen Rollen sehen den Spielplan nicht. Events wie Turniere oder Weihnachtsfeiern, die mehrere Mannschaften gleichzeitig betreffen, sind nicht abbildbar.

## What Changes

- **Neues Datenmodell**: `games.team_id` (NOT NULL) wird durch eine Junction-Tabelle `game_teams (game_id, team_id)` ersetzt — ein Event kann 1..n Mannschaften zugeordnet sein (**BREAKING** für alle Game-Queries)
- **Wizard "Event anlegen"**: 4-Schritt-Wizard ersetzt den einfachen "Heimspiel anlegen"-Dialog — Typ → Details → Vorlage → Bestätigen
- **Drei Event-Typen**: Heimspiel, Auswärtsspiel, Sonstiges Event (generisch) — je Typ wird passende Dienstplan-Vorlage gefiltert und explizit gewählt
- **Berechtigungen Lesen**: Spielplan-API und Nav-Eintrag für alle eingeloggten User (bisher: nur admin + trainer)
- **Berechtigungen Schreiben**: admin + vorstand + trainer dürfen Events anlegen/bearbeiten/löschen (bisher: nur admin)
- **Trainer-Scope**: Bei Heimspiel/Auswärtsspiel nur eigene Mannschaft(en) wählbar; bei Sonstigem Event alle Mannschaften, Multi-Select möglich
- **Slot-Generierung**: Pro gewähltem Team ein Satz Slots aus der gewählten Vorlage
- **Bug-Fix**: Frontend ruft veralteten Endpoint `/api/admin/game-template/preview` auf statt `/api/admin/duty-templates/{id}/preview`

## Capabilities

### New Capabilities

- `event-wizard`: 4-Schritt-Wizard zum Anlegen von Events (Typ → Details → Vorlage → Dienste), inkl. Multi-Team-Auswahl für generische Events und Trainer-Scoping

### Modified Capabilities

- `games`: Spielplan-Lesezugriff für alle eingeloggten User; Schreibzugriff für admin+vorstand+trainer; `team_id` durch `game_teams`-Junction ersetzt; Multi-Team-Support für generische Events

## Impact

- **DB**: Migration: `game_teams (game_id, team_id)` anlegen, bestehende `games.team_id`-Daten migrieren, `games.team_id` entfernen
- **Backend**: `internal/games/handler.go` — alle Queries auf `game_teams` JOIN umstellen; `CreateGame` akzeptiert `team_ids []int`; Slot-Generierung pro Team; Berechtigungen anpassen (`RequireRole` für Lesen entfernen, für Schreiben auf admin+vorstand+trainer setzen); Trainer-Check bei heim/auswärts
- **Frontend**: `SpielplanPage.tsx` — Wizard-Dialog mit 4 Schritten, Team-Multi-Select für generisch, Vorlage-Dropdown; `AppShell.tsx` — Nav-Eintrag ohne Rollenbeschränkung; Preview-URL-Bug fix
- **Keine neuen Abhängigkeiten**

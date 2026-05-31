## Why

`users.role` vermischt zwei unabhängige Konzepte: den Systemzugriff (wer darf was in der Plattform) und die Vereinsfunktion (welche Rolle hat jemand im Verein). Das führt zu einer redundanten Zweig-Struktur neben `members.club_function` und macht es unmöglich, dass ein Mitglied mehrere Vereinsfunktionen gleichzeitig hat (z.B. Spieler und Trainer).

## What Changes

- **BREAKING** `users.role` reduziert auf `'admin' | 'standard'` (war: admin, vorstand, trainer, elternteil, spieler)
- **BREAKING** JWT-Struktur erweitert um `club_functions: []string` und `is_parent: bool`; alle Sessions werden beim Deploy invalidiert
- Neue Junction-Tabelle `member_club_functions(member_id, function)` ersetzt `members.club_function TEXT`; Mehrfachfunktionen pro Mitglied werden damit möglich (z.B. Spieler + Trainer)
- `invitation_tokens.target_role` reduziert auf `'admin' | 'standard'`; Vereinsfunktion wird unabhängig über die Mitglieds-Verknüpfung gesetzt
- Dienstpflicht-Priorität bei Mehrfachfunktionen: Trainer > Spieler > Elternteil
- `auth.RequireRole`-Middleware bleibt für `admin`-Guards; neue Hilfsmethode `claims.HasFunction(f string)` für Vereinsfunktions-Checks
- `roleRank`-Map im Einladungs-Flow entfällt; einzige Regel: nur Admins können Admins einladen

## Capabilities

### New Capabilities
- `vereinsfunktion`: Multi-valued Vereinsfunktion für Mitglieder (Junction-Tabelle, JWT-Propagation, Dienstpflicht-Priorität)

### Modified Capabilities
- `auth`: Rollenmodell in JWT, Middleware-Guards, Einladungs-Flow und Zugriffsprüfungen ändern sich grundlegend
- `members`: `club_function`-Spalte wird durch Junction-Tabelle ersetzt; API und Formulare unterstützen Mehrfachauswahl

## Impact

- **Datenbank**: Migration ändert `users`, `members`, `invitation_tokens`; neue Tabelle `member_club_functions`
- **Backend**: `internal/auth/` (tokens, middleware, handler), `internal/members/`, `internal/dashboard/`, `internal/games/`, `internal/duties/`
- **Frontend**: `AuthContext`, `AppShell`, `App.tsx` (RoleRoute), `AdminUsersPage`, `MemberStammdatenTab`, `MembersPage`, `DashboardPage`
- **Sessions**: Alle aktiven JWT-Tokens werden beim Deploy ungültig (Breaking JWT-Änderung)

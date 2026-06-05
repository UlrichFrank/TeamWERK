## Why

Die Vereinsstruktur von Team Stuttgart erfordert zwei weitere Funktionsrollen: **Kassierer** als eigenständige Vorstandsfunktion (mit später folgenden eigenen API-Zugängen) und **Sportliche Leitung** als vereinsweite Trainerfunktion ohne Teambeschränkung. Beide Rollen existieren im Verein, sind aber im System bisher nicht abbildbar.

## What Changes

- Neuer Wert `kassierer` in `member_club_functions.function` CHECK-Constraint — vorerst nur als Label, keine neuen Routen
- Neuer Wert `sportliche_leitung` in `member_club_functions.function` CHECK-Constraint
- `sportliche_leitung` erhält dieselben API-Zugänge wie `trainer` (Kader, Spielplan, Dienste), jedoch **ohne** Teambeschränkung — sieht immer alle aktiven Kader-Teams
- `sportliche_leitung` erhält **keinen** Zugang zu Mitglieder-Seiten (diese bleiben `admin`/`vorstand`)
- Frontend: beide Funktionen in allen Dropdowns und Label-Maps ergänzt

## Capabilities

### New Capabilities

*(keine neuen eigenständigen Capabilities — die Änderungen erweitern bestehende)*

### Modified Capabilities

- `vereinsfunktion`: Zwei neue gültige Funktionswerte (`kassierer`, `sportliche_leitung`) und neue Zugangskontroll-Regeln für `sportliche_leitung` (trainer-äquivalenter Zugang + kein Team-Filter)

## Impact

- **DB-Migration**: CHECK-Constraint in `member_club_functions` erweitern
- **Backend**: `auth/tokens.go` (neue Hilfsmethode), `cmd/teamwerk/main.go` (3 Routen), `internal/games/handler.go` (2 Stellen), `internal/members/handler.go` (Team-Filter)
- **Frontend**: `lib/constants.ts`, `ProfileMemberTab.tsx`, `App.tsx`, `AppShell.tsx`
- **Breaking changes**: keine — bestehende Nutzer und Daten unverändert

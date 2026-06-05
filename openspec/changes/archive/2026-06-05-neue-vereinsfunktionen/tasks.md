## 1. Datenbank-Migration

- [x] 1.1 Nächste freie Migrations-Nummer ermitteln (`ls internal/db/migrations/`)
- [x] 1.2 `0NN_neue_vereinsfunktionen.up.sql` erstellen: `member_club_functions` neu anlegen mit CHECK-Constraint, der `kassierer` und `sportliche_leitung` enthält (PRAGMA foreign_keys OFF → Tabelle umbenennen → neu erstellen → Daten kopieren → alte Tabelle löschen → PRAGMA foreign_keys ON)
- [x] 1.3 `0NN_neue_vereinsfunktionen.down.sql` erstellen: Zeilen mit `kassierer`/`sportliche_leitung` löschen, dann Tabelle auf alten CHECK-Constraint zurücksetzen

## 2. Backend — Auth

- [x] 2.1 In `internal/auth/tokens.go` Methode `IsTrainerLike() bool` hinzufügen: `return c.HasFunction("trainer") || c.HasFunction("sportliche_leitung")`

## 3. Backend — Router

- [x] 3.1 In `cmd/teamwerk/main.go` Zeile ~199: `RequireClubFunction("trainer")` → `RequireClubFunction("trainer", "sportliche_leitung")`
- [x] 3.2 In `cmd/teamwerk/main.go` Zeile ~224: `RequireClubFunction("vorstand", "trainer")` → `RequireClubFunction("vorstand", "trainer", "sportliche_leitung")`
- [x] 3.3 In `cmd/teamwerk/main.go` Zeile ~289: `RequireClubFunction("vorstand", "trainer")` → `RequireClubFunction("vorstand", "trainer", "sportliche_leitung")`

## 4. Backend — Team-Filter

- [x] 4.1 In `internal/games/handler.go` `ListTeamsForUser()` (~Zeile 689): `if claims.HasFunction("trainer")` → `if claims.IsTrainerLike() && !claims.HasFunction("sportliche_leitung")` (sportliche_leitung bekommt den Alle-Teams-Branch)
- [x] 4.2 In `internal/games/handler.go` Spielplan-Filter (~Zeile 1512): `if claims.HasFunction("trainer")` → `if claims.IsTrainerLike() && !claims.HasFunction("sportliche_leitung")`
- [x] 4.3 In `internal/members/handler.go` trainer-Team-Filter prüfen und analog anpassen (sportliche_leitung darf alle Teams sehen)

## 5. Frontend — Konstanten und Labels

- [x] 5.1 In `web/src/lib/constants.ts` `CLUB_FUNCTION_OPTIONS` um `{ value: 'kassierer', label: 'Kassierer' }` und `{ value: 'sportliche_leitung', label: 'Sportliche Leitung' }` erweitern
- [x] 5.2 In `web/src/lib/constants.ts` `AUDIENCE_OPTIONS` um `{ value: 'kassierer', label: 'Kassierer' }` und `{ value: 'sportliche_leitung', label: 'Sportliche Leitung' }` erweitern
- [x] 5.3 In `web/src/components/profile/ProfileMemberTab.tsx` Label-Map `CLUB_FUNCTION_LABELS` um `kassierer: 'Kassierer'` und `sportliche_leitung: 'Sportliche Leitung'` erweitern

## 6. Frontend — Routing und Navigation

- [x] 6.1 In `web/src/App.tsx` Kader-Route (`admin/kader`) um `'sportliche_leitung'` in `roles`-Array erweitern
- [x] 6.2 In `web/src/components/AppShell.tsx` Kader-Nav-Eintrag um `'sportliche_leitung'` in `roles`-Array erweitern

## 7. Verifikation

- [x] 7.1 `go build ./...` erfolgreich
- [x] 7.2 `pnpm --prefix web build` erfolgreich (TypeScript ohne Fehler)
- [x] 7.3 Migration lokal testen: `make migrate-up` und `make migrate-down` ohne Fehler
- [x] 7.4 Manuell prüfen: Test-User mit `sportliche_leitung` anlegen, Kader-Seite öffnet, alle Teams sichtbar
- [x] 7.5 Manuell prüfen: Test-User mit `kassierer` anlegen, Funktion im Profil sichtbar

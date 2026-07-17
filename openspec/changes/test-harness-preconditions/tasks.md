## 1. prodserver-Verdrahtung

- [x] 1.1 `internal/testutil/prodserver/prodserver.go`: `MatchReports` (`matchreports.NewHandler`), `Settings`+`SettingsStore` (`settings.NewHandler` + `settings.NewStoreForTest`), `Stammvereine` (`stammvereine.NewHandler`) verdrahten — exakt wie `cmd/teamwerk/main.go`
- [x] 1.2 Nil-Guard für `/api/stammvereine` in `router.go` (analog `if h.MatchReports != nil`)
- [x] 1.3 Commit: `test(harness): prodserver verdrahtet MatchReports/Settings/Stammvereine`

## 2. Zentrale Fixtures

- [x] 2.1 `CreateFolder`, `SetFolderPermission`, `CreateFile` in `fixtures.go`
- [x] 2.2 `PostMultipart` in `server.go`
- [x] 2.3 `CreateAbsence`, `RecordTrainingAttendance`, `RecordGameAttendance`
- [x] 2.4 `SetMemberBankEnvelope`, `SetClubSepaEnvelope`
- [x] 2.5 `CreateMemberWithFields` (Options-Struct `MemberOpts`) — additiv, `CreateMember` bleibt
- [x] 2.6 Commit: `test(harness): zentrale Fixtures für files/absences/attendance/beitragslauf`

## 3. Authz-Drift-Detektor

- [x] 3.1 `internal/arch/authz_test.go`: scope-bewusster AST-Walker über `BuildRouter` (Stack für `r.Group`/`r.Route`-Scopes), Alias-Auflösung `auth.Role*`
- [x] 3.2 Abgleich gegen die Erwartungs-Maps in `internal/permissions/matrix_test.go`; verwaiste Einträge failen
- [x] 3.3 Begründete Allowlist-Struktur für bewusste Ausnahmen
- [x] 3.4 Commit: `test(arch): Authz-Drift-Detektor (Router-Gates ↔ Persona-Matrix)`

## 4. Verifikation

- [x] 4.1 `go test ./...` grün — 41 Pakete, keine Fehler; gofmt + golangci-lint sauber
- [x] 4.2 Drift-Lücken geschlossen: Die neu verdrahteten Handler machten **14** zuvor unsichtbare Routen sichtbar (maintenance-mode GET/POST, maintenance-status, alle match-reports-Routen). Klassifiziert in `matrix_test.go`: Autor-Tier → `exPressTeam` (nur admin, da keine press_team-Persona), Freigeber → `exMatchReportPublisher` (admin+vorstand, keine medien-Persona), Handler-entscheidet-Routen → `exMatchReportMixed` (httpAnyOK), maintenance-mode → `exAdmin`, maintenance-status → `exPublic`. **Offen für Welle 1:** press_team-/medien-Persona ergänzen, um die Positiv-Pfade (press_team DARF) echt zu prüfen.
- [x] 4.3 `openspec validate test-harness-preconditions` grün
- [ ] 4.4 Archivieren (nach Review durch den Menschen)

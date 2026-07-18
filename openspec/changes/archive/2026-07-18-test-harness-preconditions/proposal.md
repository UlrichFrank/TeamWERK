## Why

Welle 0 der `test-coverage-roadmap` — der Enabler mit dem höchsten Hebel pro Aufwand. Drei Blindspots blockieren mechanisch die nachfolgenden Test-Wellen und lassen einen bereits existierenden Autorisierungs-Gate ins Leere laufen:

1. **`internal/testutil/prodserver/prodserver.go` verdrahtet `h.MatchReports`, `h.Settings`(+Store) und `h.Stammvereine` nie.** In `router.go` stehen deren Routen hinter `if h.X != nil {…}` — im Test-Router bleiben die Felder nil, die Routen werden nie registriert. `internal/permissions/matrix_test.go` (das über `BuildRouter` läuft) sieht die komplette Spielbericht-Autorisierung (RequireRole PressTeam/Admin, RequireClubFunction medien/vorstand) daher nie. Der Gate ist blind auf einer ganzen Domäne.
2. **`/api/stammvereine` hat keinen Nil-Guard.** Bei nicht verdrahtetem Handler → Nil-Panic → `health.Recoverer` → HTTP 500. Der `httpAllowed`-Check (`≠401/403`) wertet 500 still als „bestanden" — ein maskierter Fehlpfad.
3. **Zentrale Test-Fixtures fehlen.** `internal/files` ist ohne einen Multipart-Helper (`PostMultipart`) und `CreateFile`/`CreateFolder`/`SetFolderPermission` gar nicht route-testbar (9/12 Routen). `absences`/`attendance`/`beitragslauf` duplizieren Insert-Logik lokal, die zentral gehört (`CreateAbsence`, `Record*Attendance`, `Set*Envelope`, `CreateMemberWithFields`).

Ohne diese Vorbedingungen sind Welle 1 (PII-Route-Authz) und Welle 2 (Finance-Audit) nicht sauber umsetzbar.

## What Changes

- **`prodserver`-Verdrahtung vervollständigen**: `MatchReports`, `Settings`+`SettingsStore`, `Stammvereine` exakt wie in `cmd/teamwerk/main.go` bauen. Nil-Guard für `/api/stammvereine` (kein maskierter 500 mehr).
- **Zentrale Fixtures** nach `internal/testutil` (additiv, ersetzen keine bestehenden Signaturen):
  - `CreateFolder`, `SetFolderPermission`, `CreateFile` (files)
  - `PostMultipart` (Server-Helper in `server.go`; optional `Put`/`Delete`-Convenience)
  - `CreateAbsence`, `RecordTrainingAttendance`, `RecordGameAttendance`
  - `SetMemberBankEnvelope`, `SetClubSepaEnvelope`
  - `CreateMemberWithFields` (Options-Struct — **neben** `CreateMember`, nicht statt)
- **Authz-Drift-Detektor** in `internal/arch` analog `broadcast_test.go`: parst `BuildRouter`, sammelt pro Route das aktive `RequireRole`/`RequireClubFunction`-Set (scope-bewusster Walker über `r.Group`/`r.Route`/`if`-Blöcke, Alias-Auflösung für `auth.Role*`-Konstanten), und failt, wenn eine gated Route in keiner Erwartungs-Zeile der `permissions`-Matrix steht (oder ein Matrix-Eintrag verwaist ist).

## Impact

- **Kein Laufzeit-/Produktionsverhalten betroffen** — Änderungen liegen ausschließlich in `internal/testutil/**` und `internal/arch/**` (Testcode). Ausnahme: der Nil-Guard in `router.go` (defensive, ändert nur den nil-Handler-Fall, der in Produktion nie eintritt, weil `main.go` alle Handler verdrahtet).
- **Sofort-Effekt**: `matrix_test.go` deckt nach der Verdrahtung matchreports/settings/stammvereine mit ab; der Drift-Detektor meldet konkrete Lücken, die in Welle 1 geschlossen werden.
- Betroffen: `internal/testutil/prodserver/prodserver.go`, `internal/testutil/fixtures.go`, `internal/testutil/server.go`, `internal/arch/authz_test.go` (neu), `internal/app/router.go` (nur Nil-Guard).

## Test-Anforderungen

| Route/Invariante | Testname | erwartet |
|---|---|---|
| Drift-Detektor findet gated Route ohne Matrix-Eintrag | `TestArch_AuthzGatesMatchMatrix` | Fail mit Routennamen |
| Verwaister Matrix-Eintrag (Route entfernt) | `TestArch_AuthzMatrix_NoOrphans` | Fail mit Eintragsnamen |
| matchreports-Autor-Tier über Prod-Router | `TestPermissionMatrix_MatchReports_NonPressTeamForbidden` | 403 (nicht 500) |
| stammvereine-Erfolgspfad über Prod-Router | `TestStammvereine_ViaProdRouter_NoNilPanic` | 200/201 (nicht 500) |

Garantierte Invariante: Jede hinter `RequireRole`/`RequireClubFunction` gemountete Route ist entweder in der Persona-Matrix erfasst oder in einer begründeten Allowlist — und wird über den **echten** `BuildRouter` mit verdrahteten Handlern geprüft, nicht über einen Mini-Router.

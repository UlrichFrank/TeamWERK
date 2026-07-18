# test-harness Specification

## Purpose
TBD - created by archiving change test-harness-preconditions. Update Purpose after archive.
## Requirements
### Requirement: Test-Produktions-Router verdrahtet alle Handler
`internal/testutil/prodserver` SHALL jeden in `cmd/teamwerk/main.go` verdrahteten Handler ebenfalls verdrahten, sodass `app.BuildRouter` im Test dieselben Routen registriert wie in Produktion. Kein Handler-Feld darf im Test-Router nil bleiben, wenn die zugehörige Route hinter einem Autorisierungs-Gate gemountet ist.

#### Scenario: Neuer Handler wird in main.go verdrahtet
- **WHEN** ein Handler (z.B. `MatchReports`) in `main.go` gebaut und in `router.go` hinter `RequireRole`/`RequireClubFunction` gemountet wird, `prodserver` ihn aber nicht setzt
- **THEN** der Drift-Detektor SHALL fehlschlagen und die unverdrahtete Domäne benennen — die gated Routen dürfen nicht still übersprungen werden

#### Scenario: Nil-Handler darf nicht als bestanden durchrutschen
- **WHEN** eine Route ohne Nil-Guard registriert ist und der Handler nil ist
- **THEN** die Route SHALL einen Nil-Guard erhalten, sodass kein Nil-Panic → HTTP 500 entsteht, das ein `httpAllowed`-Check (`≠401/403`) fälschlich als bestanden wertet

### Requirement: Zentrale Fixtures für Cross-Domain-Tabellen
Test-Fixtures für Tabellen, die von mehr als einem Package benötigt werden (`file_folders`, `folder_permissions`, `files`, `member_absences`, `training_attendances`, `game_attendances`, Bank-/SEPA-Envelope), SHALL in `internal/testutil` zentralisiert sein statt paket-lokal dupliziert. Neue Fixtures SHALL additiv sein und bestehende Signaturen (z.B. `CreateMember`) nicht ersetzen.

#### Scenario: files-Route braucht Multipart-Upload
- **WHEN** ein Test `POST /api/folders/{id}/files` (Upload) ansteuern will
- **THEN** ein `PostMultipart`-Helper in `internal/testutil` SHALL vorhanden sein, sodass der Upload-Handler route-seitig testbar ist (nicht nur die reine ACL-Funktion)

#### Scenario: erweiterte Member-Felder für Beitragslogik
- **WHEN** ein Test die Halbierungslogik (`join_date`/`exit_date`/`home_club_id`) benötigt
- **THEN** `CreateMemberWithFields` (Options-Struct) SHALL diese Felder setzen können, während `CreateMember` mit einfacher Signatur für bestehende Tests erhalten bleibt

### Requirement: Autorisierungs-Gates werden mechanisch gegen die Persona-Matrix abgeglichen
Ein Arch-Test in `internal/arch` SHALL `BuildRouter` parsen und für jede hinter `RequireRole`/`RequireClubFunction` gemountete Route prüfen, dass sie in den Persona-Erwartungs-Maps (`internal/permissions/matrix_test.go`) erfasst ist. Bewusste Ausnahmen SHALL in einer begründeten Allowlist stehen; verwaiste Allowlist-/Matrix-Einträge SHALL den Test fehlschlagen lassen.

#### Scenario: Neue gated Route ohne Matrix-Eintrag
- **WHEN** eine neue Route mit `RequireClubFunction(...)` gemountet wird, ohne dass eine Persona-Erwartung dafür existiert
- **THEN** der Arch-Test SHALL fehlschlagen und die Route beim Namen nennen

#### Scenario: Gate-Argument ändert sich
- **WHEN** die Rollen-/Funktionsliste eines Gates in `router.go` geändert wird, die hand-gepflegte Erwartungs-Map aber nicht
- **THEN** der Abgleich SHALL die Divergenz erkennen, statt sie erst im nächsten manuellen Review aufzudecken


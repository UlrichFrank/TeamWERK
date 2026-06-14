## Why

Das Berechtigungsmodell wurde umgebaut: `users.role` kennt nur noch `admin` und `standard`. Die fünf historischen Rollen (`trainer`, `vorstand`, `sportliche_leitung`, `elternteil`, `spieler`) sind heute **Vereinsfunktionen** in `member_club_functions` und werden via `claims.HasFunction(...)` bzw. `auth.RequireClubFunction(...)` geprüft.

Der Audit zeigt jedoch, dass an mehreren Stellen noch der alte Sprachgebrauch lebt — als SQL-Filter (`WHERE u.role = 'spieler'`), als Claim-Vergleich (`claims.Role == "trainer"`), als Frontend-Check (`user.role === 'vorstand'`), in Test-Tokens und in der Doku. Die betroffenen Queries liefern **nie** ein Ergebnis, weil keine User mehr mit `role='spieler'` etc. existieren — d.h. die Dienstbörse-Erinnerungen für Spieler und Eltern laufen still ins Leere. Zusätzlich existiert die Phantom-Vereinsfunktion `sportvorstand`, die nie ins Schema aufgenommen wurde, aber an zwei Stellen geprüft wird.

## What Changes

- **fix(scheduler):** SQL-Queries in `internal/scheduler/scheduler.go` für `target_role = 'spieler'` und `target_role = 'elternteil'` schreiben gegen `member_club_functions` (Spieler) bzw. `family_links` + `member_club_functions(spieler)` (Eltern), nicht gegen `users.role`.
- **fix(duties):** Empfänger-Auflösung in `internal/duties/handler.go` analog umstellen.
- **fix(absences):** `claims.Role == "trainer"` durch `claims.HasFunction("trainer")` ersetzen. `claims.HasFunction("sportvorstand")` zu `claims.HasFunction("vorstand")` korrigieren (Phantom-Funktion eliminieren).
- **fix(frontend):** In `DutyPage`, `KalenderPage`, `MembersPage` jeden Vergleich `user.role === '<alte rolle>'` durch `hasFunction(user, '<funktion>')` ersetzen. Phantom-Check `hasFunction(user, 'sportvorstand')` aus `KalenderPage.tsx` entfernen.
- **fix(frontend):** `RoleRoute`/`AppShell`-Logik: NavItems und Routen-Guards trennen klar System-Rolle (`admin`/`standard`) und Vereinsfunktion. Vorhandene polymorphe Helfer-Funktion wird umbenannt und dokumentiert.
- **chore(db):** Migration `042_duty_types_target_role_check.up.sql` aktualisiert den CHECK-Constraint auf `duty_types.target_role` von `('spieler','elternteil','trainer','admin','vorstand')` auf valide Werte `('spieler','elternteil','trainer','vorstand','sportliche_leitung','vorstand_beisitzer','kassierer')`. `'admin'` als target_role wird entfernt (semantisch sinnlos: Admin ist keine Zielgruppe für Dienste).
- **test(auth):** `internal/auth/handler_test.go:385` prüft jetzt explizit, dass `PUT /api/admin/users/{id}/role` mit `{"role":"trainer"}` 400 zurückgibt.
- **test(trainings):** `internal/trainings/handler_test.go:361` setzt Token mit `role="standard"` und `isParent=true` statt `role="elternteil"`.
- **docs(claude-md):** Rollen-Tabelle in `CLAUDE.md` aufgeräumt: zwei System-Rollen + Liste der Vereinsfunktionen, mit klarer Abgrenzung wann welcher Mechanismus greift.

## Capabilities

### New Capabilities
*(keine — alle Änderungen verfeinern bestehende Capabilities)*

### Modified Capabilities
- `auth`: Klärt die Invariante, dass `users.role` nur `admin` und `standard` annimmt, und dokumentiert die Trennung zwischen System-Rolle und Vereinsfunktion. Ergänzt Scenario zur Ablehnung ungültiger Rollen-Werte durch `PUT /api/admin/users/{id}/role`.
- `vereinsfunktion`: Ergänzt Scenario, das die Phantom-Funktion `sportvorstand` als ungültig kennzeichnet (sie wurde nie ins Schema aufgenommen), und stellt klar, dass `elternteil` keine Vereinsfunktion ist sondern aus `family_links` abgeleitet wird.
- `duties`: Aktualisiert den `target_role`-Wertebereich auf die valide Vereinsfunktion-Menge (plus `elternteil` als spezieller Familien-Marker).

## Impact

**Code:**
- `internal/scheduler/scheduler.go` (SQL-Queries Spieler/Eltern)
- `internal/duties/handler.go` (Empfänger-Auflösung)
- `internal/absences/handler.go` (Claims-Checks)
- `internal/auth/handler_test.go` und `internal/trainings/handler_test.go` (Test-Fixes)
- `web/src/pages/DutyPage.tsx`, `web/src/pages/KalenderPage.tsx`, `web/src/pages/MembersPage.tsx` (Frontend-Checks)
- `web/src/App.tsx`, `web/src/components/AppShell.tsx` (Klärung Role vs. ClubFunction)

**Datenbank:**
- Neue Migration `042_duty_types_target_role_check.{up,down}.sql` (CHECK-Constraint via `CREATE TABLE … _new; INSERT…SELECT; DROP; RENAME` — SQLite-Pattern für CHECK-Änderungen). Daten-Migration: Bestehende `duty_types.target_role='admin'` (falls vorhanden) wird auf `'vorstand'` gemappt.

**Doku:**
- `CLAUDE.md` Rollen-Tabelle aktualisiert.

**Keine breaking changes** in HTTP-APIs — die Routen-Pfade und Response-Schemata bleiben unverändert; nur die *interne Auswertung* der Berechtigung wird konsistent.

**Risiko:** Wenn in der Produktions-DB `duty_types`-Datensätze mit `target_role='admin'` existieren, müssen sie vor dem Migration-Run inspiziert werden. Die Migration mappt sie automatisch auf `vorstand`, aber das sollte einmalig validiert werden.

## Test-Anforderungen

- **Route `PUT /api/admin/users/{id}/role` — Invariante:** Akzeptiert nur `admin` und `standard`. Test: `TestUpdateUserRole_RejectsLegacyRole` setzt `{"role":"trainer"}` und erwartet 400.
- **Scheduler `eligibleUsers` — Invariante:** Für `target_role='spieler'` werden User mit `member_club_functions.function='spieler'` zurückgegeben, nicht User mit `users.role='spieler'` (existieren nicht mehr). Test: `TestEligibleUsers_SpielerViaClubFunction` legt einen Member mit Funktion `spieler` an, setzt ein Slot mit `target_role='spieler'`, erwartet den User in der Empfängerliste.
- **Scheduler `eligibleUsers` — Invariante:** Für `target_role='elternteil'` werden User über `family_links` zu Kindern mit Vereinsfunktion `spieler` aufgelöst. Test: `TestEligibleUsers_ElternteilViaFamilyLinks`.
- **Absences Read-Access — Invariante:** Trainer eines Teams (Vereinsfunktion `trainer` + `kader_trainers`-Eintrag) darf Abwesenheiten des Teams sehen. Test: `TestAbsences_TrainerLikeAccess`.
- **Duty-Type Target-Role — Invariante:** `INSERT INTO duty_types (..., target_role='admin') …` schlägt nach Migration 042 mit CHECK-Constraint-Fehler fehl.

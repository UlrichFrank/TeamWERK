## 1. Datenbank-Migration

- [x] 1.1 Neue Migration `internal/db/migrations/042_duty_types_target_role_check.up.sql` anlegen — CHECK-Constraint auf `duty_types.target_role` über `CREATE TABLE … _new`-Pattern aktualisieren auf valide Werte `('spieler','elternteil','trainer','vorstand','sportliche_leitung','vorstand_beisitzer','kassierer')`. Daten-Map `'admin' → 'vorstand'` im `INSERT…SELECT`. Indizes von `duty_types` (falls vorhanden) neu anlegen. Commit: `chore(db): Migration 042 – duty_types.target_role CHECK ohne Legacy-Werte`
- [x] 1.2 Dazu `internal/db/migrations/042_duty_types_target_role_check.down.sql` schreiben, der den alten CHECK (`'spieler','elternteil','trainer','admin','vorstand'`) wiederherstellt und `'sportliche_leitung'`/`'vorstand_beisitzer'`/`'kassierer'` zurück auf `'vorstand'` mappt. Commit: Teil von 1.1.
- [x] 1.3 Pre-flight-Hinweis in der Migration-Datei (SQL-Kommentar oben) dokumentieren: „Bestehende Rows mit target_role='admin' werden zu 'vorstand'. Vor Deploy einmalig prüfen: SELECT id, name FROM duty_types WHERE target_role='admin'." Commit: Teil von 1.1.

## 2. Backend SQL-Queries fixen

- [x] 2.1 `internal/duties/handler.go:46–50` umschreiben: statt `WHERE u.role IN ('spieler','elternteil','trainer')` joine über `members` und `member_club_functions` (Funktion `spieler` oder `trainer`) sowie `family_links` (für Eltern). Die Funktion gibt User-IDs zurück, die für ein Team-Reminder in Frage kommen. Commit: `fix(duties): Empfänger-Auflösung über member_club_functions statt users.role`
- [x] 2.2 `internal/scheduler/scheduler.go:368–387` (`case "spieler"`): SQL umschreiben — `WHERE u.role = 'spieler'` entfernen, stattdessen `JOIN members m ON m.user_id = u.id JOIN member_club_functions mcf ON mcf.member_id = m.id AND mcf.function = 'spieler'`. Bei `teamID.Valid` zusätzlich `JOIN kader_members km ON km.member_id = m.id JOIN kader k ON k.id = km.kader_id AND k.team_id = ? AND k.season_id = (SELECT id FROM seasons WHERE is_active=1)`. Commit: `fix(scheduler): Spieler-Empfänger über member_club_functions und Kader`
- [x] 2.3 `internal/scheduler/scheduler.go:389–409` (`case "elternteil"`): SQL umschreiben — `WHERE u.role = 'elternteil'` entfernen, stattdessen `JOIN family_links fl ON fl.parent_user_id = u.id JOIN members m ON m.id = fl.member_id JOIN member_club_functions mcf ON mcf.member_id = m.id AND mcf.function = 'spieler'` (Eltern sind Eltern von Spielern). Team-Filter analog 2.2. Commit: `fix(scheduler): Eltern-Empfänger über family_links und Kind-Vereinsfunktion`
- [x] 2.4 `internal/scheduler/scheduler.go:433–439` (`default`-Fall): Statt `WHERE u.role = ?` joine `member_club_functions mcf ON mcf.function = ?`. Commit: Teil von 2.2/2.3 oder eigenen Commit `fix(scheduler): Fallback-Empfänger-Query über Vereinsfunktion statt SystemRole`.

## 3. Backend Claims-Checks fixen

- [x] 3.1 `internal/absences/handler.go:544–545`: `claims.Role == "trainer"` entfernen (kann nie zutreffen). `claims.HasFunction("sportvorstand")` durch `claims.HasFunction("vorstand")` ersetzen — die Bedingung wird zu: `claims.Role == "admin" || claims.HasFunction("vorstand") || claims.IsTrainerLike()`. Commit: `fix(absences): Phantom-Vereinsfunktion sportvorstand entfernt, trainer-Check auf HasFunction umgestellt`
- [x] 3.2 Grep über `internal/` nach `claims.Role == "trainer"`, `claims.Role == "vorstand"`, `claims.Role == "spieler"`, `claims.Role == "elternteil"`, `claims.Role == "sportliche_leitung"` und `claims.HasFunction("sportvorstand")` — sicherstellen, dass keine weiteren Treffer existieren. Commit: nur falls weitere Stellen gefunden, dann eigener Fix-Commit pro Datei.

## 4. Frontend Claims-Checks fixen

- [x] 4.1 `web/src/pages/DutyPage.tsx:34`: `const isAdminOrTrainer = user?.role === 'admin' || user?.role === 'trainer'` → `const isAdminOrTrainer = user?.role === 'admin' || hasFunction(user, 'trainer') || hasFunction(user, 'sportliche_leitung')`. Import `hasFunction` falls noch nicht vorhanden. Commit: `fix(duty-page): trainer-Check über hasFunction statt user.role`
- [x] 4.2 `web/src/pages/KalenderPage.tsx:119`: `user.role === 'trainer'` durch `hasFunction(user, 'trainer')` ersetzen. Commit: `fix(kalender): trainer-Check über hasFunction statt user.role`
- [x] 4.3 `web/src/pages/KalenderPage.tsx:120`: `hasFunction(user, 'sportvorstand')` entfernen — `vorstand` deckt den Fall ab. Commit: Teil von 4.2 oder `fix(kalender): Phantom-Vereinsfunktion sportvorstand entfernt`.
- [x] 4.4 `web/src/pages/KalenderPage.tsx:683`: `user.role === 'spieler' || user.role === 'elternteil'` entfernen — `hasFunction(user, 'spieler') || user.isParent` ist die korrekte Bedingung (steht schon dahinter). Commit: `fix(kalender): canCreateAbsence über Vereinsfunktion und isParent statt user.role`
- [x] 4.5 `web/src/pages/MembersPage.tsx:110`: `user?.role === 'vorstand'` durch `hasFunction(user, 'vorstand')` ersetzen. Commit: `fix(members-page): vorstand-Check über hasFunction statt user.role`
- [x] 4.6 Grep über `web/src/` nach `user?.role === 'trainer'`, `user.role === 'trainer'`, `user.role === 'vorstand'`, `user.role === 'spieler'`, `user.role === 'elternteil'`, `user.role === 'sportliche_leitung'` und `hasFunction(user, 'sportvorstand')` — sicherstellen, dass keine weiteren Treffer existieren. Commit: nur bei Funden.

## 5. Frontend Nav/Route-Guards dokumentieren

- [x] 5.1 `web/src/App.tsx:47` (`RoleRoute`): JSDoc über `RoleRoute` ergänzen, der erklärt, dass `roles`-Strings sowohl System-Rollen (`admin`, `standard`) als auch Vereinsfunktionen sein können — die Komponente prüft `r === 'admin' ? user.role === 'admin' : hasFunction(user, r)`. Kein Verhaltens-Change. Commit: `docs(app): RoleRoute-Polymorphismus dokumentiert`
- [x] 5.2 `web/src/components/AppShell.tsx:147-148`: JSDoc-Kommentar über `NavItem.roles` und `excludeRoles` ergänzen — analog 5.1. Helper `matchesRequirement(user, r): boolean` extrahieren statt inline-Lambda. Commit: `refactor(app-shell): NavItem-Polymorphismus über matchesRequirement-Helper`

## 6. Tests anpassen

- [x] 6.1 `internal/auth/handler_test.go:385`: Test `TestUpdateUserRole_RejectsLegacyRole` — schickt `{"role":"trainer"}` und erwartet HTTP 400 (status ist heute schon falsch geprüft, sicherstellen dass die Erwartung 400 ist und der Body `invalid role` enthält). Zusätzlich Sub-Tests für `vorstand`, `spieler`, `elternteil`, `sportliche_leitung`. Commit: `test(auth): Legacy-Rollen werden von UpdateUserRole abgelehnt`
- [x] 6.2 `internal/trainings/handler_test.go:361`: Token-Fixture korrigieren — statt `IssueAccessToken(secret, userID, email, "elternteil", ...)` setze `IssueAccessToken(secret, userID, email, "standard", []string{}, true /* isParent */)`. Wenn Test danach failt, Logik im Handler prüfen — die Abhängigkeit von `claims.Role == "elternteil"` muss in `claims.IsParent` umgeschrieben werden. Commit: `test(trainings): Eltern-Token verwendet role=standard + isParent statt role=elternteil`
- [x] 6.3 Neuer Test `internal/scheduler/scheduler_test.go::TestEligibleUsers_SpielerViaClubFunction` — legt User+Member mit Vereinsfunktion `spieler` an, Slot mit `target_role='spieler'`, erwartet User in der Empfängerliste. Commit: `test(scheduler): Spieler-Auflösung über member_club_functions`
- [x] 6.4 Neuer Test `internal/scheduler/scheduler_test.go::TestEligibleUsers_ElternteilViaFamilyLinks` — legt Eltern-User mit family_link zu Member-mit-spieler-Funktion an, Slot mit `target_role='elternteil'`, erwartet Eltern in Empfängerliste. Commit: `test(scheduler): Eltern-Auflösung über family_links`
- [x] 6.5 Neuer Test `internal/duties/handler_test.go::TestDutyType_TargetRole_RejectsAdmin` — versucht `INSERT INTO duty_types (..., target_role='admin')`, erwartet CHECK-Fehler (nach Migration 042). Commit: `test(duties): duty_types CHECK rejected admin als target_role`

## 7. Aktive Specs aufräumen

- [x] 7.1 `openspec/specs/team-absences-calendar/spec.md:4`: Text „Club-Funktion `sportvorstand`/`vorstand`" ändern zu „Club-Funktion `vorstand`/`sportliche_leitung`". (Archivierter Change unter `openspec/changes/archive/2026-06-14-team-absences-calendar/` bleibt unangetastet — Archiv ist eingefroren.) Commit: `docs(specs): team-absences-calendar – sportvorstand entfernt`
- [x] 7.2 Grep über `openspec/specs/**/*.md` nach `sportvorstand` und nach Rollen-Begriffen wie ``Rolle `trainer`'' im Sinne von SystemRole. Stellen, die in aktiven Specs den alten Sprachgebrauch nutzen, gegen den neuen begrifflichen Standard anpassen (System-Rolle `admin`/`standard` vs. Vereinsfunktion). Archive (`openspec/changes/archive/`) **nicht** anfassen. Commit: pro Datei einen `docs(specs): …`-Commit oder gebündelt `docs(specs): aktive Specs auf SystemRole/Vereinsfunktion-Vokabular vereinheitlicht`.

## 8. CLAUDE.md aktualisieren

- [x] 8.1 `CLAUDE.md` Abschnitt „Rollen" (Zeile ~131): Tabelle ersetzen durch:
  - Tabelle „System-Rollen": `admin` (Vollzugriff, umgeht ClubFunction-Checks), `standard` (alle anderen).
  - Tabelle „Vereinsfunktionen": `spieler`, `trainer`, `vorstand`, `vorstand_beisitzer`, `kassierer`, `sportliche_leitung` mit Kurzbeschreibung pro Funktion.
  - Hinweis: `elternteil` ist **keine** Vereinsfunktion sondern wird via `family_links` aufgelöst und im JWT als `is_parent: true` mitgeführt.
  - Hinweis: `sportvorstand` existiert nicht — falls fachlich gewünscht, kombiniere `vorstand` + `sportliche_leitung`.
  Commit: `docs(claude-md): Rollen-Tabelle auf SystemRole + Vereinsfunktion neu strukturiert`

## 9. Verifikation & Smoke-Test

- [x] 9.1 `make build && make migrate-up` lokal — Migration 042 erfolgreich, Build grün.
- [x] 9.2 `go test ./...` lokal — alle Tests grün (insbesondere die neuen 6.1, 6.3, 6.4, 6.5).
- [ ] 9.3 Lokaler Smoke-Test: User mit Vereinsfunktion `spieler` anlegen → Duty-Slot mit `target_role='spieler'` erstellen → `teamwerk scheduler:run` triggern → Reminder-Mail-Log/Push-Log prüfen, dass User adressiert wurde.
- [x] 9.4 Frontend `cd web && pnpm build` — keine TypeScript-Fehler durch fehlende `hasFunction`-Imports.
- [ ] 9.5 Browser-Test: Login als User mit `vorstand`-Funktion → MembersPage öffnen → Schreib-Buttons (Neu, Edit) sichtbar (Test für 4.5).
- [ ] 9.6 Browser-Test: Login als User mit `trainer`-Funktion → Kalender öffnen → Schreib-Funktionen sichtbar (Test für 4.2).
- [ ] 9.7 Browser-Test: Login als reiner Eltern-User (`isParent=true`, keine Funktion) → Kalender → „Abwesenheit anlegen" sichtbar (Test für 4.4).

## 10. Abschluss

- [ ] 10.1 OpenSpec-Proposal als applied markieren bzw. nach Archiv verschieben gemäß `openspec/AGENTS.md`-Workflow. Commit: `chore(openspec): cleanup-legacy-roles archiviert`

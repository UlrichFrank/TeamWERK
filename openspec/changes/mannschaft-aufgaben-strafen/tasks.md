## 1. Datenmodell & Migration

- [ ] 1.1 Nächste freie Migrationsnummer ermitteln (`ls internal/db/migrations/`) und `0NN_mannschaft_aufgaben_strafen.up.sql` + `.down.sql` anlegen
- [ ] 1.2 Table `responsibility_types(id, kader_id → kader ON DELETE CASCADE, label)` + UNIQUE(kader_id, label)
- [ ] 1.3 Table `penalty_types(id, kader_id → kader ON DELETE CASCADE, reason, default_amount_cent INTEGER)`
- [ ] 1.4 Table `kader_strafenwarte(kader_id → kader ON DELETE CASCADE, member_id → members ON DELETE CASCADE, PRIMARY KEY(kader_id, member_id))`
- [ ] 1.5 Table `member_responsibilities(id, kader_id → kader ON DELETE CASCADE, member_id → members ON DELETE CASCADE, label)` — label als Snapshot
- [ ] 1.6 Table `team_penalties(id, kader_id → kader ON DELETE CASCADE, member_id → members ON DELETE CASCADE, amount_cent INTEGER, reason, created_by_member_id → members, created_at DATETIME)` — reason/amount_cent als Snapshot, kein Status
- [ ] 1.7 `.down.sql` droppt alle fünf Tables in FK-sicherer Reihenfolge
- [ ] 1.8 `make migrate-up` lokal ausführen; Migration idempotent verifizieren; Architektur-Test (neues Package klassifizieren, falls eigenes Package) beachten

## 2. Backend — Team→Kader-Auflösung & Gates (policy)

- [ ] 2.1 Helper in `internal/policy/rules.go`: `IsTrainerOfKader(userID, kaderID)` bzw. team-basiert, im Stil des bestehenden Trainer-Scopings
- [ ] 2.2 Helper `IsStrafenwartOfKader(userID, kaderID)` (Lookup `kader_strafenwarte` via `members.user_id`)
- [ ] 2.3 Helper `CanReadPenalties(userID, teamID)` — true bei Spieler/Trainer/Erweitertem Kader der aktiven Saison, **false** bei Eltern/Außenstehenden
- [ ] 2.4 Team→aktive-Saison-Kader-Auflösung an bestehende `GetRoster`-Logik ausrichten (mehrere Kader pro Team via `team_number` gleich behandeln)
- [ ] 2.5 Unit-Tests für die drei Gates (positive + negative Fälle, insb. Eltern → false, Fremd-Team → false)

## 3. Backend — Aufgaben-Handler & Routen

- [ ] 3.1 Handler für Aufgaben-Catalog-CRUD (`responsibility_types`) — Gate Trainer des Kaders, `h.hub.Broadcast("responsibilities")`
- [ ] 3.2 Handler für Aufgaben-Zuweisung/Entfernen (`member_responsibilities`) — Snapshot-Label, Gate Trainer, Broadcast
- [ ] 3.3 `GetRoster` erweitern: Aufgaben-Labels je Spieler additiv in die Roster-Response aufnehmen
- [ ] 3.4 Routen in `internal/app/router.go` (`BuildRouter`) unter passendem Tier registrieren (`r.PathValue`)
- [ ] 3.5 Tests: `TestResponsibilityCatalog_TrainerCreates_200`, `_NonTrainer_403`, `TestResponsibilityAssign_Trainer_200`, `TestRoster_IncludesResponsibilities`, `TestResponsibility_CatalogEditKeepsSnapshot`

## 4. Backend — Strafen-Handler & Routen

- [ ] 4.1 Handler `GET /api/teams/{id}/penalties` — Read-Gate `CanReadPenalties`, Liste + Kassenstand-Summe pro Spieler; **nicht** auf der Roster-Response
- [ ] 4.2 Handler `POST /api/teams/{id}/penalties` — Gate Strafenwart des Kaders, Betrag editierbar (Snapshot), Broadcast `penalties`
- [ ] 4.3 Handler `DELETE /api/teams/{id}/penalties/{pid}` — Storno (hard delete), Gate Strafenwart, Broadcast
- [ ] 4.4 Handler `DELETE /api/teams/{id}/penalties?member={mid}` — Zurücksetzen je Spieler (hard delete), Gate Strafenwart, Broadcast
- [ ] 4.5 Handler Strafen-Catalog-CRUD (`penalty_types`) + Strafenwart-Ernennung (`kader_strafenwarte`) — Gate Trainer, Broadcast
- [ ] 4.6 Routen in `BuildRouter` registrieren
- [ ] 4.7 Tests: `TestPenalties_Player_200`, `_ExtendedMember_200`, `_Parent_403`, `_Outsider_403`, `TestRoster_ExcludesPenalties`
- [ ] 4.8 Tests: `TestPenaltyCreate_Strafenwart_200`, `_NonStrafenwart_403`, `_ForeignTeamStrafenwart_403`, `TestPenaltyStorno_Strafenwart_200`, `TestPenaltyReset_PerMember_200`
- [ ] 4.9 Tests: `TestStrafenwartAppoint_Trainer_200`/`_NonTrainer_403`, `TestPenalty_CatalogEditKeepsSnapshot`, `TestClubFunctions_NoStrafenwartValue`

## 5. Test-Fixtures

- [ ] 5.1 Fixtures in `internal/testutil/` ergänzen soweit nötig (z.B. `AppointStrafenwart`, `CreatePenalty`, `AssignResponsibility`) — bestehende `CreateKader`/`CreateMember`/`CreateTeam` nutzen
- [ ] 5.2 Sicherstellen, dass Fixtures Eltern (`family_links`) und Erweiterten Kader (`kader_extended_members`) für die Negativ-/Positiv-Tests abbilden

## 6. Frontend — MeinTeamPage

- [ ] 6.1 Aufgaben als Chips neben dem Spielernamen im Team-Tab (`RosterSection`), aus der erweiterten Roster-Response
- [ ] 6.2 Strafen als eigener Bereich/Tab, nur sichtbar/geladen für Berechtigte (`GET /teams/{id}/penalties`; 403 sauber behandeln = Bereich ausblenden), Liste + Kassenstand-Summe pro Spieler (Cent → €-Anzeige)
- [ ] 6.3 Trainer-Verwaltung: Aufgaben-Catalog pflegen, Aufgaben zuweisen, Strafen-Catalog pflegen, Strafenwart ernennen (Dropdown + Freitext, brand-Tokens, lucide-Icons, `aria-label` bei Icon-Buttons)
- [ ] 6.4 Strafenwart-Aktionen: Strafe vergeben (Betrag editierbar), stornieren, je Spieler zurücksetzen (Bestätigung vor Reset)
- [ ] 6.5 `useLiveUpdates` um `responsibilities` und `penalties` erweitern → betroffene Ansicht neu laden

## 7. Verifikation

- [ ] 7.1 `make test` (inkl. Architektur- + Broadcast-Gate) grün
- [ ] 7.2 `make lint` grün (brand-Tokens statt Raw-Tailwind, keine Emojis/Unicode-Icons)
- [ ] 7.3 `pnpm -C web build && pnpm -C web test && pnpm -C web lint` grün
- [ ] 7.4 `/verify-change` durchlaufen (Route→Tests, Mutation→Broadcast/useLiveUpdates, Migrationsnummer, `openspec validate`)
- [ ] 7.5 Ein Commit pro Task-Gruppe (Conventional Commits, Scope `teams`/`db`/`policy`); abschließend Proposal archivieren

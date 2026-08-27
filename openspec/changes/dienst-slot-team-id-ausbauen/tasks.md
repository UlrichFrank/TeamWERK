## 1. Leseseite tolerant machen (muss vor allem anderen kommen)

- [x] 1.1 `internal/duties/handler.go` (Board): Bedingung `ds.team_id IS NULL AND ds.game_id IN (…)` → `ds.game_id IN (…)`; den `ds.team_id IN (meine Teams)`-Zweig auf Slots **ohne** `game_id` einschränken
- [x] 1.2 Gleiche Umstellung im `eltern`-Audience-Zweig (`pm_a.team_id = ds.team_id` nur noch bei `ds.game_id IS NULL`)
- [x] 1.3 `internal/dashboard/handler.go` — identischer Doppelzweig
- [x] 1.4 `internal/duties/handler.go` `slotTeamScope` + `internal/scheduler/scheduler.go` `slotTeamScope`: `game_id` hat Vorrang vor `team_id` (heute umgekehrt)
- [x] 1.5 `internal/hub/audience.go` `TeamIDsForDutySlot`: dieselbe Vorrang-Umkehr
- [x] 1.6 Go-Test: Bestands-Slot mit `game_id` **und** `team_id=A` an einem Termin mit A+B ist für B sichtbar und erreicht B bei Benachrichtigung/Erinnerung

## 2. Schreibseite: kein `team_id` mehr an spielgebundenen Slots

- [x] 2.1 `regen.go` `makeCustomKey`: `HasTeam=false`, sobald der Slot ein `game_id` trägt (Aufrufer in `snapshotCustomSlots`, `loadNewAutoSlotsKeyed`, `snapshotDeletedSlots` mitziehen)
- [x] 2.2 `regen.go` `regenGameItems`: Team-Loop → `itemAppliesToAnyTeam`-Prädikat, ein Insert mit `team_id = NULL`; `createdCount` entsprechend (nicht mehr `n × Teams`)
- [x] 2.3 `regen.go` `buildRotationPlan`/Insert: kein `team_id`; Zuteilungen mehrerer Teams am selben Anker-Spiel zu einem Slot mit summierten Kuchen verschmelzen
- [x] 2.4 `duties.CreateSlot`: `team_id` bei gesetztem `game_id` ignorieren (kein 400)
- [x] 2.5 `SpieltagDetailModal.tsx`: `team_id` nicht mehr senden, Helfer `slotTeamIdForGame` samt Test entfernen
- [x] 2.6 Go-Test: Regen und `CreateSlot` schreiben bei gesetztem `game_id` `team_id = NULL` (Schreibseite direkt geprüft, nicht nur über die Sichtbarkeit)
- [x] 2.7 Go-Test: Zusage überlebt den Übergang alt (`team_id=A`) → neu (`NULL`) ohne „Dienst entfernt"-Benachrichtigung
- [x] 2.8 Go-Test: Heimspiel mit zwei Teams erzeugt einen Slot je Vorlagen-Item (statt zwei) und einen zusammengefassten Rotations-Slot

## 3. Migration

- [x] 3.1 `internal/db/migrations/0NN_duty_slots_team_id_nur_ohne_spiel.up.sql`: `UPDATE duty_slots SET team_id = NULL WHERE game_id IS NOT NULL`
- [x] 3.2 `.down.sql` leer anlegen, mit Kommentar warum kein Rückweg nötig ist (alter Code unterstützt den Zielzustand)
- [x] 3.3 Nächste freie Migrationsnummer prüfen (`ls internal/db/migrations | tail`) — nie ≤ aktueller DB-Version

## 4. Verifikation

- [x] 4.1 `make test`, `pnpm -C web test`, `golangci-lint run`, `openspec validate --strict`
- [x] 4.2 Gegen eine frische Kopie der Prod-DB: vor der Migration Sichtbarkeit für ein Mitglied jedes Teams eines Mehr-Team-Termins prüfen, dann migrieren, dann erneut prüfen (identisches Ergebnis)
- [x] 4.3 Auf der Kopie einen Regen-Lauf über einen Termin mit Zusage fahren und bestätigen, dass `duty_assignments` erhalten bleibt
- [ ] 4.4 Deploy A, danach `make backup` und `make migrate-remote-up`

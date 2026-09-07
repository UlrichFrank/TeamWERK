## 1. Empfängerauflösung an einer Stelle

- [x] 1.1 `internal/notify/audience.go` anlegen: `TeamAudience(db *sql.DB, teamIDs ...int) []int` — eine Query mit `team_id IN (…)` und vier `UNION`-Zweigen (Stammkader-Mitglied, dessen Elternteil, erweitertes Mitglied, dessen Elternteil), `DISTINCT`, Bezug auf die aktive Saison; leere Team-Liste liefert ohne Query eine leere Menge
- [x] 1.2 Query-Fehler über `slog.Error` protokollieren und leere Menge liefern (kein stiller `nil`-Rückgabepfad wie in den Bestandskopien, siehe design.md Decision 3)
- [x] 1.3 Einheitentests: doppelte Zugehörigkeit (beide Kader-Listen, mehrere Mannschaften) erzeugt genau einen Empfänger; Mitglied ohne `user_id` fällt raus; Kader einer inaktiven Saison zählt nicht; leere Team-Liste

## 2. Aufrufstellen umstellen

- [x] 2.1 `internal/games/handler.go`: `teamMembersAndParents` löschen, die drei Aufrufstellen (Anlage, Änderung inkl. entfernter Teams, Absage) auf `notify.TeamAudience` ziehen
- [x] 2.2 `internal/trainings/handler.go`: `teamMembersAndParents` löschen (inkl. des unvollständigen `kader_extended_members`-Zweigs), die vier Aufrufstellen umstellen
- [x] 2.3 `internal/scheduler/scheduler.go`: `teamMembersAndParents` löschen, Spiel- und Trainings-Erinnerungen umstellen; die Reihenfolge „Empfänger auflösen → `claimUnsent` → senden" beibehalten, damit die Idempotenz je Nutzer und Slot unverändert greift
- [x] 2.4 `internal/scheduler/event_notes_push.go`: `teamMembersAndParentsMulti` ersatzlos löschen, Termin-Hinweise auf `notify.TeamAudience(db, teamIDs...)` ziehen
- [x] 2.5 Prüfen, dass keine Aufrufstelle die Menge zusätzlich filtert oder vorsortiert — `notify.Send` übernimmt Event-Log, Push-Präferenz und E-Mail unverändert

## 3. Verhalten absichern

- [x] 3.1 `internal/games`: `TestCreateGame_ErwKaderUndElternWerdenBenachrichtigt`, `TestUpdateGame_ErwKaderWirdBenachrichtigt`, `TestDeleteGame_ErwKaderWirdBenachrichtigt`
- [x] 3.2 `internal/games`: `TestDeleteGame_SilentUnterdruecktAuchErwKader` — `silent` unterdrückt die Meldung für **alle**, der Broadcast läuft trotzdem
- [x] 3.3 `internal/trainings`: `TestUpdateSession_ElternDesErwKadersWerdenBenachrichtigt`, `TestDeleteSeries_ErwKaderUndElternWerdenBenachrichtigt`
- [x] 3.4 `internal/scheduler`: `TestGameReminder_ErwKaderErhaeltReminder`, `TestTrainingReminder_ErwKaderErhaeltReminder` (inkl. Duplikatschutz über `notification_log`), `TestEventNotePush_ErwKaderErhaeltHinweis`
- [x] 3.5 Bestandstests der vier Meldungswege durchsehen: keine Assertion darf auf der alten, engeren Empfängerzahl festhängen

## 4. Rückfall verhindern

- [x] 4.1 `internal/arch`: `TestKeineEigeneEmpfaengerAufloesung` — scannt `internal/` nach Funktionsrümpfen, die `player_memberships` und `family_links` verbinden, und lässt sie außerhalb von `internal/notify` fehlschlagen
- [x] 4.2 Allowlist mit Begründung füllen: die Dienst-Pfade in `internal/duties` und die Dienst-Erinnerung im Scheduler fragen nach Dienstpflicht, nicht nach Terminbetroffenheit (design.md Decision 6); verwaiste Einträge lassen den Test fehlschlagen
- [x] 4.3 Gotcha in `docs/agent/06-gotchas.md` ergänzen: Empfängermenge einer Terminmeldung kommt aus `notify.TeamAudience`, Dienste bleiben auf `player_memberships`

## 5. Abschluss

- [x] 5.1 `make test` + `golangci-lint` grün; `openspec validate terminmeldungen-erweiterter-kader --strict`
- [x] 5.2 Ankündigung an die Trainer aktualisieren: der erweiterte Kader und dessen Eltern bekommen ab dem Deploy dieselben Termin- und Erinnerungsmeldungen; Bitte um Durchsicht der erweiterten Kader (Karteileichen erzeugen jetzt echte Meldungen); Dienste bleiben unverändert beim Stammkader

## 6. Nachtrag: Trainer und Auslöser

- [x] 6.1 Fünfter `UNION`-Zweig in `notify.TeamAudience`: Trainer des Kaders (`kader_trainers`) — **ohne** Eltern-Zweig, sie stehen in ihrer Funktion in der Menge, nicht als Kind
- [x] 6.2 Festhalten, dass der Auslöser einer Änderung **nicht** herausgefiltert wird (Bestätigung des eigenen Versands) — Spec-Szenario + Test statt stillschweigendem Nebeneffekt
- [x] 6.3 Tests: `TestTeamAudience_FuenfGruppen`, `TestTeamAudience_TrainerOhneElternZweig`, `TestTeamAudience_TrainerUndSpielerInEinerPerson`, `TestCreateGame_AusloeserBekommtDieEigeneMeldung`; Trainer in die Fixtures von games/trainings/scheduler aufgenommen
- [x] 6.4 Spec, Design (Decision 4b + 7), Proposal und Gotcha auf die erweiterte Menge gezogen

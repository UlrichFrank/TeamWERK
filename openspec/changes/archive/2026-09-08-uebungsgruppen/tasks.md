## 1. Schema

- [x] 1.1 Migration `057_uebungsgruppen` — `kader`-Rebuild: `age_class` und `gender`
      nullable, neue Spalten `kind TEXT NOT NULL DEFAULT 'team' CHECK (kind IN ('team','practice'))`
      und `name TEXT`. `PRAGMA legacy_alter_table=ON` (vier Views verweisen auf `kader`),
      alle Indizes neu anlegen (`kader_unique`, `idx_kader_season`)
- [x] 1.2 CHECK-Constraint aus `design.md — Entscheidung 6` ergänzen (Alles-oder-nichts je
      Variante, inkl. `team_id IS NULL` bei `practice`). Den bestehenden
      `CHECK (gender IN ('m','f','mixed'))` **unverändert** übernehmen — `NULL IN (…)`
      erfüllt ihn
- [x] 1.3 `CREATE UNIQUE INDEX idx_kader_practice_name ON kader(season_id, name) WHERE kind='practice'`
- [x] 1.4 `057_uebungsgruppen.down.sql` — Rebuild zurück auf NOT NULL; scheitert bewusst,
      wenn `kind='practice'`-Zeilen existieren (kein stiller Datenverlust)
- [x] 1.5 Migration `058_trainings_kader_owner` — Rebuild von `training_sessions` und
      `training_series`: `kader_id INTEGER NOT NULL REFERENCES kader(id) ON DELETE RESTRICT`
      mit Backfill über `(team_id, season_id)`, `team_id` auf nullable. Ein Datensatz ohne
      passenden Kader lässt die Migration am NOT NULL abbrechen — gewollt.
      **RESTRICT, nicht CASCADE:** `teams` werden nie gelöscht, Kader schon — ein CASCADE
      nähme beim Aufräumen eines leeren Altkaders die Trainingshistorie mit
- [x] 1.6 `058_trainings_kader_owner.down.sql`; prüfen, dass `training_responses`,
      `training_attendances` und `member_series_unavailabilities` die Rebuilds unbeschadet
      überstehen (IDs bleiben erhalten, `foreign_keys=OFF` beim Up)

## 2. Übungsgruppen-Package

- [x] 2.1 `internal/practicegroups/handler.go` — `NewHandler(db, hub)`, CRUD für
      `GET|POST /api/practice-groups`, `GET|PUT|DELETE /api/practice-groups/{id}`,
      `GET /api/practice-groups/{id}/member-suggestions`
- [x] 2.2 Anlage schreibt `kind='practice'` **ohne** `ensureTeam` — kein `teams`-Zwilling.
      Aktive Saison ist Pflicht (400 sonst), Namensdublette → 409
- [x] 2.3 Jede Mutation ruft `h.hub.Broadcast("practice-groups")` (Broadcast-Gate,
      `internal/arch/broadcast_test.go`)
- [x] 2.4 Routen in `internal/app/router.go` im Vorstand-Tier registrieren, Handler in
      `cmd/teamwerk/main.go` verdrahten
- [x] 2.5 `internal/arch/arch_test.go` — neues Package `practicegroups` als Domain
      klassifizieren
- [x] 2.6 Gate in `internal/kader/handler.go`: `PUT /api/kader/{id}` lehnt
      `extended_members_add`/`_remove` bei `kind='practice'` mit HTTP 409 ab
- [x] 2.7 Trainings-Guard in `DeleteKader` **und** `DeletePracticeGroup`: 409 mit
      `training_count`, solange Termine oder Serien am Kader hängen — dieselbe Form wie die
      bestehende Mitglieder-Guard (`handler.go:737`)
- [x] 2.8 `internal/kader/copy.go` — `CopyFromSeason` filtert `kind='practice'` aus der
      Quellmenge. Ohne diesen Filter kollabieren alle Übungsgruppen auf den `sourceMap`-
      Schlüssel `"|"` und `ensureTeam(NULL, NULL, 1)` legt ein Müll-Team an.
      *Nur die Sicherung — ein Kopierpfad für Übungsgruppen ist entschieden ausgeschlossen
      (`design.md — Entscheidung 7`).*

## 3. Trainings auf `kader_id` umstellen

- [x] 3.1 `internal/trainings/handler.go` — alle Vorkommen von
      `k.team_id = ts.team_id AND k.season_id = ts.season_id` durch `k.id = ts.kader_id`
      ersetzen (63 Fundstellen). Der Block wird dabei kürzer, nicht länger
- [x] 3.2 `player_memberships`-Zugriffe im selben Paket durch direktes `kader_members` mit
      `kader_id` ersetzen — die View filtert `WHERE k.team_id IS NOT NULL` und würde
      Übungsgruppen ausschließen
- [x] 3.3 Schreibpfade (`POST`/`PUT` Session und Serie) setzen `kader_id`; `team_id` wird
      aus `kader.team_id` abgeleitet und ist für Übungsgruppen NULL
- [x] 3.4 `internal/trainings/unavailabilities.go` auf `kader_id` umstellen
- [x] 3.5 `internal/hub/audience.go` — `trainingTeams` (Zeile 80) löst die SSE-Zielmenge
      über `kader_id` auf. Ohne diesen Schritt bleiben Live-Updates für Übungsgruppen
      **stumm**, ohne Fehlermeldung
- [x] 3.6 `internal/scheduler` — 24-h-/3-h-Erinnerungen und Termin-Hinweise erreichen
      Übungsgruppen (Empfänger über `kader_id`)
- [x] 3.7 Verifizieren, dass `internal/attendance`, `absences`, `calendar`, `dashboard`,
      `videos` **unverändert** bleiben und über `team_id` weiter auflösen — das ist der
      Ausschlussmechanismus, kein Versäumnis

## 4. Chat

- [x] 4.1 `internal/chat/team_groups.go` — `ListTeamGroups` liefert Übungsgruppen mit
      `groupType='practice'`; Sichtbarkeit über Mitgliedschaft/Trainer/Eltern statt über
      `user_accessible_teams`
- [x] 4.2 Neue Route `GET /api/chat/practice-groups/{id}/{kind}/members` samt Auflösung;
      `spieler` **ohne** Union mit `kader_extended_members`
- [x] 4.3 404 bei `kind='team'`-Kader über die Übungsgruppen-Route

## 5. Frontend

- [x] 5.1 `web/src/pages/UebungsgruppenPage.tsx` — Maske analog `AdminKaderPage`, aber
      ohne Altersklasse, Geschlecht, Jahrgang und erweiterten Kader; Name als Pflichtfeld
- [x] 5.2 Route `/uebungsgruppen` in `App.tsx`, Nav-Eintrag in `AppShell.tsx`
      (Vorstand/Admin), `useLiveUpdates` auf `practice-groups`
- [x] 5.3 Trainings-Anlage kann eine Übungsgruppe als Ziel wählen
- [x] 5.4 `AdminKaderPage.tsx:561` und `:725` — `optgroup label="Trainingsgruppen"` →
      `"Sonderkader"`. Reine Beschriftung, keine Datenänderung
- [x] 5.5 Nur `brand-*`-Tokens, `lucide-react`-Icons, Klassen-Strings aus
      `lib/buttonStyles.ts` importieren

## 6. Tests & Abschluss

- [ ] 6.1 **Vor dem Deploy** auf Prod zählen:
      `SELECT COUNT(*) FROM training_sessions ts WHERE NOT EXISTS (SELECT 1 FROM kader k WHERE k.team_id=ts.team_id AND k.season_id=ts.season_id);`
      (analog `training_series`). Ergebnis muss 0 sein, sonst bricht Migration `058` ab
- [x] 6.2 Alle Tests aus `proposal.md — Test-Anforderungen`. Der ursprünglich
      gelistete `TestRsvpUebungsgruppe_FremderAbgelehnt` ist entfallen: die
      fehlende Kaderprüfung der Selbst-RSVP ist variantenunabhängiges
      Bestandsverhalten und wird im Folge-Change `rsvp-kader-gate` geschlossen
      (`design.md — Bekannter Rest`)
- [x] 6.3 **Bestehende Trainings-Tests müssen grün bleiben** — sie sind der
      Regressionsschutz für Task 3, insbesondere die RSVP-Sichtbarkeit für Spieler,
      Eltern, Trainer und erweiterten Kader der Mannschaftsvariante
- [x] 6.4 `TestDeleteKader_MitTrainingsAbgelehnt`, `TestDeletePracticeGroup_MitTrainingsAbgelehnt`,
      `TestDeletePracticeGroup_OhneTrainingsErfolg`,
      `TestCopyFromSeason_UeberspringtUebungsgruppen`
- [x] 6.5 `TestPracticeGroup_KeineTeamRoute` — hält die strukturelle Aussage fest, dass
      Übungsgruppen unter `/api/teams/{id}/…` nicht adressierbar sind
- [x] 6.6 `make test`, `golangci-lint`, `pnpm -C web build/test/lint`,
      `openspec validate uebungsgruppen --strict`
- [x] 6.7 Gotcha in `docs/agent/06-gotchas.md`: „Übungsgruppen" — `kader_id` ist der
      Besitzer eines Trainings, `training_sessions.team_id IS NULL` ist die Zusage und
      **kein** Datenfehler; die Abwesenheit der `teams`-Zeile ist das Gate. Bei der
      Gelegenheit den Satz im Ordner-Rechte-Gotcha präzisieren: „Kader ohne `team_id`
      (Trainingsgruppen) sind darüber nicht adressierbar" stimmt für Förderkader/
      Perspektivkader **nicht** (`createTrainingGroupKader` ruft `ensureTeam`, sie haben
      ein `team_id`) — er beschreibt ab jetzt genau die Übungsgruppen
- [x] 6.8 `docs/agent/04-api-db.md` — Auth-Tier-Tabelle um `/api/practice-groups` ergänzen

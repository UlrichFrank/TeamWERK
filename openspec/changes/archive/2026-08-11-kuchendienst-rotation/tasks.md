## 1. Datenbank

- [x] 1.1 Migration `045_bewirtungsrotation.up.sql`: `ALTER TABLE game_template_items ADD COLUMN rotation_max_per_team INTEGER;` + `INSERT OR IGNORE INTO system_settings (key, value) VALUES ('bewirtung_verhaeltnis', '1');` (nächste freie Nummer vor dem Schreiben mit `ls internal/db/migrations/ | sort -V | tail -1` erneut prüfen)
- [x] 1.2 Migration `045_bewirtungsrotation.down.sql`: `ALTER TABLE game_template_items DROP COLUMN rotation_max_per_team;` + `DELETE FROM system_settings WHERE key='bewirtung_verhaeltnis';`
- [x] 1.3 `make migrate-up` lokal verifiziert (up/down/up-Zyklus ohne Fehler, gegen eine isolierte Kopie — nicht gegen die echte `teamwerk.db`, siehe Hinweis unten). **Bekannte Repo-Einschränkung** (nicht Teil dieses Change): `teamwerk migrate down` bzw. `make migrate-down` ist ein No-Op — `runMigrate()`/`db.Migrate()` in `cmd/teamwerk/main.go`/`internal/db/db.go` ruft ausschließlich `m.Up()` auf, es gibt keinen echten „down"-Pfad im CLI. Die `.down.sql`-Datei wurde deshalb direkt per `sqlite3 < 045_bewirtungsrotation.down.sql` gegen eine Testkopie verifiziert (Spalte weg, Row weg) statt über `make migrate-down`.

## 2. Backend: Bewirtung-Einstellung

- [x] 2.1 `internal/settings/bewirtung.go`: `GetBewirtungVerhaeltnis`/`SetBewirtungVerhaeltnis` als Direkt-Read/Write gegen `system_settings.bewirtung_verhaeltnis` — kein `Store`/`atomic`/Poll-Loop (design.md Decision 6). `GetBewirtungVerhaeltnis` nimmt ein kleines `RowQuerier`-Interface (`QueryRowContext`) statt konkret `*sql.DB`, damit die Regen-Engine (Gruppe 4) es innerhalb ihrer eigenen `tx` lesen kann, ohne die Logik zu duplizieren.
- [x] 2.2 Route `GET /api/settings/bewirtung` → `{"verhaeltnis": number}`, Authenticated-Tier
- [x] 2.3 Route `PUT /api/settings/bewirtung` → Body `{"verhaeltnis": number}`, gated auf Vereinsfunktion `vorstand` (Admin-Bypass automatisch über bestehende Middleware); validiert `verhaeltnis > 0` (400 sonst); persistiert als String; ruft `h.hub.Broadcast("settings-changed")`
- [x] 2.4 Routen in `internal/app/router.go` eingetragen (GET im Authenticated-Block, PUT im bestehenden `vorstand`-Block)
- [x] 2.5 Tests in `internal/settings/bewirtung_test.go`: Happy-Path (inkl. Admin-Bypass, Broadcast-Assertion), Fehlerfall 403 (Nicht-Vorstand, Wert unverändert), Fehlerfall 400 (negativ/nicht-numerisch, Wert unverändert), GET ohne Auth → 401

## 3. Backend: Rotations-Cap auf Vorlagen-Items

- [x] 3.1 `templateItemRow` (`internal/games/regen.go`) und `templateItem`/JSON-Struct (`internal/games/handler.go`) um `RotationMaxPerTeam sql.NullInt64` bzw. `*int json:"rotation_max_per_team,omitempty"` erweitert; Queries um `gti.rotation_max_per_team` ergänzt
- [x] 3.2 `PUT /api/duty-templates/{id}` (tatsächliche Route — **kein** `/admin`-Präfix, abweichend von der ursprünglichen Task-Notiz): `rotation_max_per_team` lesen/schreiben, analog zum `team_ids`-Pattern
- [x] 3.3 Validierung: Item mit gesetztem `rotation_max_per_team` UND `same_day_behavior != 'normal'` oder `adjacent_day_behavior != 'normal'` → `400 rotation_requires_normal_behavior`, nichts persistiert (design.md Decision 4). Zusätzlich: `rotation_max_per_team <= 0` → `400 invalid_rotation_max_per_team`
- [x] 3.4 Tests in `internal/games/rotation_template_test.go`: Happy-Path (Cap gesetzt + per GET zurückgelesen), Fehlerfall `same_day_behavior≠normal` → 400, Fehlerfall `adjacent_day_behavior≠normal` → 400, Fehlerfall 403 (Nicht-Vorstand, Wert bleibt NULL)

## 4. Backend: Regen-Engine — Tagesweite Rotation

- [x] 4.1 `buildRotationPlan(ctx, tx, dayGames)` in `internal/games/regen.go`: filtert `event_type='heim'`, lädt pro Spiel Template-Items + Team, gruppiert rotations-aktivierte Items nach `duty_type_id` (`rotationGroup`), baut Team-Warteschlange (distinct, erstes Auftreten), berechnet `kuchenBedarf = min(homeGameCount, ceil(homeGameCount * verhältnis))` pro Gruppe, weist die ersten `kuchenBedarf` Heimspiele (chronologisch) zu (greedy, Cap pro Team), Rest bleibt `team_id=NULL`. Rückgabetyp `rotationPlan = map[dutyTypeID]map[gameID]rotationAssignment{HasSlot, TeamID}` (design.md Decision 1–3)
- [x] 4.2 `regenSingleDay` ruft `buildRotationPlan` einmal pro Tag auf und reicht `rotationTypes` (pro Spiel aus `items` abgeleitet) an `regenGameItems` durch
- [x] 4.3 `regenGameItems`: neuer Zweig für Items mit `RotationMaxPerTeam.Valid` — ein `insertOne`-Aufruf pro Spiel, Team aus dem Rotationsplan (ggf. `sql.NullInt64{}`); Spiele ohne Plan-Eintrag werden übersprungen (kein Slot, kein `Skipped`-Eintrag)
- [x] 4.4 `RegenSummary.Unassigned []UnassignedEntry{Date, DutyType, GameID}` ergänzt; `capSummary`/`mergeSummary` mitgezogen
- [x] 4.5 `makeCustomKey(dutyTypeID, eventTime, teamID, rotationTypes)` als **eine** zentrale Stelle für den Key-Aufbau eingeführt, genutzt von `snapshotCustomSlots`, `loadNewAutoSlotsKeyed`, `restoreAssignments` und der Konfliktprüfung — lässt `TeamID`/`HasTeam` weg, wenn `rotationTypes[dutyTypeID]` wahr ist (design.md Decision 5). Per Mutationsprobe verifiziert (Ausnahme testweise entfernt → Restore-Test schlägt fehl, danach zurückgesetzt)
- [x] 4.6 Unit-Tests `internal/games/rotation_regen_test.go` (eigene Datei statt `regen_test.go`, um die bestehende Datei nicht unnötig aufzublähen):
  - Warteschlangen-Aufbau: Team mit mehreren Spielen erscheint einmal, an Position des ersten Spiels
  - Greedy-Zuteilung 5 Spiele / 3 Teams / Cap 2 → `A,A,B,B,C`
  - Verhältnis `0.5`: nur die ersten N Spiele bekommen einen Slot
  - Verhältnis `2` bei 3 Spielen: höchstens 3 Slots, nicht 6
  - Cap-Überlauf: überzählige Slots mit `team_id=NULL`, `RegenSummary.Unassigned` gefüllt, kein Team über Cap
  - Reset pro Spieltag: zwei aufeinanderfolgende Tage mit gleicher Konstellation starten beide bei Position 1
  - Restore: Zusage auf Rotations-Slot überlebt Regen trotz verschobener Team-Zuordnung (Team wechselt, `event_time` bleibt) — `(duty_type_id, event_time)`-Match
  - Restore: Nicht-Rotations-Item behält weiterhin `(duty_type_id, event_time, team_id)`-Match (Regressionstest gegen bestehendes Verhalten)

## 5. Frontend: Einstellungen-Tab „Bewirtung"

- [x] 5.1 `AdminSettingsPage.tsx`: `TABS`-Array um `{id: 'bewirtung', label: 'Bewirtung', cap: 'manage_duty_types'}` erweitert (keine exakt passende Capability existierte; `manage_duty_types` gated dieselbe `IsVorstandLike`-Personengruppe wie die `PUT`-Route serverseitig)
- [x] 5.2 `BewirtungTab`-Komponente: lädt `GET /api/settings/bewirtung`, Zahlen-Input mit Komma-/Punkt-Parsing, speichert via `PUT`, Hinweistext zur Deckelung, reagiert live auf `settings-changed` (SSE)
- [x] 5.3 `AdminDutyTemplateDetailPage.tsx`: `RotationCapField`-Komponente neben `TeamScopeField`, leer=deaktiviert, zeigt `rotation_requires_normal_behavior` als lesbare deutsche Fehlermeldung an
- [x] 5.4 `settings-changed`-Live-Reload im `BewirtungTab` ergänzt
- [x] 5.5 Vitest: `AdminSettingsPage.bewirtung.test.tsx` (5 Tests), `AdminDutyTemplateDetailPage.rotationCap.test.tsx` (5 Tests) — alle grün

## 6. Abschluss

- [x] 6.1 `docs/agent/06-gotchas.md`: Gotcha-Absatz „Bewirtungs-/Kuchendienst-Rotation" ergänzt (Team-Warteschlange, Bedarfsformel, Unassigned-Fallback, Restore-Matching-Ausnahme, `same_day_behavior='normal'`-Voraussetzung)
- [x] 6.2 Verifiziert: `go build ./...`, `go vet ./...`, `go test ./...` (alle Packages inkl. `internal/games`, `internal/settings`, `internal/arch`), `golangci-lint run ./...` (0 issues), `pnpm -C web build`, `pnpm -C web test` (757/757), `pnpm -C web lint` (0 Fehler, 16 vorbestehende Warnungen außerhalb der geänderten Dateien), `openspec validate --changes kuchendienst-rotation` — alles grün
- [ ] 6.3 Change archivieren (`openspec-archive-change`) nach Review/Merge — **noch offen**, bewusst nicht automatisch ausgeführt (Review/Merge steht noch aus)

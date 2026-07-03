## 1. Datenbank-Migration

- [ ] 1.1 `internal/db/migrations/018_rsvp_defaults_per_role.up.sql`: für `training_sessions`, `training_series`, `games` je Tabelle neu erzeugen (SQLite `CREATE TABLE …_new` + `INSERT INTO … SELECT …` + `DROP` + `RENAME`) mit den zwei neuen Spalten `rsvp_default_players TEXT NOT NULL DEFAULT 'none' CHECK (rsvp_default_players IN ('confirmed','declined','none'))` und `rsvp_default_extended TEXT NOT NULL DEFAULT 'none' CHECK (rsvp_default_extended IN ('confirmed','declined','none'))`. Backfill: `rsvp_default_players = CASE WHEN rsvp_opt_out = 1 THEN 'confirmed' ELSE 'none' END`, `rsvp_default_extended = 'none'`. Ohne `rsvp_opt_out` in der neuen Tabelle.
- [ ] 1.2 `internal/db/migrations/018_rsvp_defaults_per_role.down.sql`: umgekehrter Weg — `rsvp_opt_out INTEGER NOT NULL DEFAULT 0` zurück, `rsvp_opt_out = CASE WHEN rsvp_default_players = 'confirmed' THEN 1 ELSE 0 END` (dokumentierter Datenverlust bei `declined`-Werten, wird in `.down.sql`-Kommentar erwähnt).
- [ ] 1.3 Lokaler Migrations-Roundtrip: `make migrate-up`, spot-check per `sqlite3` (`.schema training_sessions`), `make migrate-down`, erneut `up`.

## 2. Backend — Trainings

- [ ] 2.1 `internal/trainings/handler.go`: `sessionItem` / `seriesItem` / Request-Structs um `RsvpDefaultPlayers` und `RsvpDefaultExtended` (JSON `rsvp_default_players`, `rsvp_default_extended`) erweitern; `RsvpOptOut`-Feld entfernen.
- [ ] 2.2 `insertSessions`: neue Spalten aus der Serie in die Session-Insert-Liste aufnehmen.
- [ ] 2.3 `PUT /api/training-series/{id}` und `PUT /api/training-sessions/{id}`: Payload akzeptiert nur die neuen Felder; alte `rsvp_opt_out`-Referenz aus dem SET-Builder entfernen.
- [ ] 2.4 Payload-Validierung: `rsvp_default_players='declined'` oder `rsvp_default_extended='declined'` in Kombination mit `rsvp_require_reason=1` → HTTP 400 `{"error":"invalid_rsvp_settings"}`.
- [ ] 2.5 `GetAttendances`-Query: Default-Zweig für Zeilen ohne Response ersetzen. Für Trainer-Zeilen bleibt `confirmed` hart. Für Spieler/Erweiterten Zeilen: `COALESCE(tr.status, CASE WHEN is_extended=1 THEN ts.rsvp_default_extended ELSE ts.rsvp_default_players END)` — wenn Ergebnis `'none'` ist, bleibt `rsvp_status=NULL`.
- [ ] 2.6 Result-Loop: neben `RSVPStatus` ein `RSVPIsDefault bool` (JSON: `rsvp_is_default,omitempty`) markieren, wenn die Antwort virtuell aus dem Default kam (für UI-Unterscheidung dezent/kursiv).
- [ ] 2.7 Header-Zähler-Query (`GetSession` und aggregierte Session-Listen): `LEFT JOIN` mit `COALESCE(status, default_for_role)` — Trainer weiterhin per `NOT IN (SELECT member_id FROM kader_trainers …)` ausschließen; Default `'none'` zählt nirgendwo mit.

## 3. Backend — Games

- [ ] 3.1 `internal/games/handler.go`: `gameItem` + Request-Structs analog um die zwei Felder erweitern, `RsvpOptOut` entfernen.
- [ ] 3.2 `POST /api/games` (Insert) und `PUT /api/games/{id}` (Update SET-Builder): auf neue Spalten umstellen; alte Referenz raus.
- [ ] 3.3 Payload-Validierung analog zu 2.4.
- [ ] 3.4 `GetAttendances`- / `GetParticipants`-Query: Default-Zweig identisch zu Trainings-Semantik; `COALESCE` mit Rollen-abhängigem Default; `IsDefault`-Flag im JSON.
- [ ] 3.5 `GET /api/games/my` und `GET /api/games/{id}` `my_rsvp`-Feld: liefert nach Priorität Response > `rsvp_default_players` (wenn im Stammkader) > `rsvp_default_extended` (wenn nur im Erweiterten Kader) > `null`.
- [ ] 3.6 Header-Zähler-Query (`ListGames`, `GetGame`, `ListMyGames`): Default-Werte einbeziehen, Trainer ausschließen.

## 4. Backend-Tests

- [ ] 4.1 `internal/trainings/rsvp_defaults_test.go` — Happy-Path: Session mit `rsvp_default_players='declined'` → Spieler ohne Response bekommt `rsvp_status='declined'`, `rsvp_is_default=true`.
- [ ] 4.2 Erweiterter Kader unabhängig: `players='confirmed'`, `extended='none'` → Stammkader auto-confirmed, Erweiterter Kader `null`.
- [ ] 4.3 Serie → Session-Copy: Serie mit `players='declined'` anlegen, generierte Session erbt die Werte.
- [ ] 4.4 Konfliktsperre: `PUT` mit `players='declined' + rsvp_require_reason=1` → HTTP 400, keine DB-Änderung.
- [ ] 4.5 Header-Zähler: Session mit `players='confirmed'`, 3 Kader-Spieler, 0 Responses → `confirmed_count=3`, `declined_count=0`.
- [ ] 4.6 Header-Zähler: Session mit `extended='declined'`, 2 Erweiterte, 0 Responses → `declined_count=2`.
- [ ] 4.7 Trainer bleibt hart-confirmed: Session mit `players='declined'` → Trainer (aus `kader_trainers`) hat weiterhin `rsvp_status='confirmed'` in der Zeile, wird aber nicht im Zähler mitgezählt.
- [ ] 4.8 Bestandstests `handler_test.go`, `erw_kader_eltern_test.go`, `trainer_rsvp_test.go` an neue Feldnamen anpassen.
- [ ] 4.9 Symmetrische Test-Suite `internal/games/rsvp_defaults_test.go` für Games (Happy-Path, Konfliktsperre, Header-Zähler, `GET /api/games/my`).

## 5. Frontend — Edit-Modals

- [ ] 5.1 `web/src/components/TrainingEditModal.tsx`: Checkbox „Alle Spieler standardmäßig zugesagt (Opt-Out)" ersetzen durch zwei Radio-Gruppen. Titel „RSVP-Voreinstellung" darüber. Gruppen: „Kader-Spieler" und „Erweiterter Kader", je drei Radios („Standardmäßig zugesagt" / „Standardmäßig abgesagt" / „Keine automatische Rückmeldung"). State `rsvpOptOut`/`setRsvpOptOut` entfernen, ersetzen durch `rsvpDefaultPlayers` / `rsvpDefaultExtended`.
- [ ] 5.2 `web/src/components/TrainingEditModal.tsx`: Konfliktsperre — wenn `rsvpRequireReason=true`, sind die `declined`-Radios `disabled`; wenn eine der beiden Voreinstellungen `declined` ist, ist die `rsvpRequireReason`-Checkbox `disabled` mit `title`-Tooltip „Nicht mit ‚Standardmäßig abgesagt' kombinierbar".
- [ ] 5.3 `web/src/components/GameEditModal.tsx`: analog zu 5.1 und 5.2.
- [ ] 5.4 `web/src/pages/AdminTrainingsPage.tsx`: Serie-Bulk-Formular analog umbauen.
- [ ] 5.5 Alle Frontend-`rsvp_opt_out`-Referenzen (`web/src/lib/api.ts`, Types, Test-Fixtures) entfernen; Payload-Shape auf neue Felder.

## 6. Frontend — Detail-Seiten & Kalender

- [ ] 6.1 `web/src/pages/TermineDetailPage.tsx`: `AttendanceItem`/`ParticipantItem`-Typen um `rsvp_is_default?: boolean` erweitern. `ParticipantRow`: bei `rsvp_is_default` die Antwort-Anzeige mit `text-brand-text-subtle italic` rendern statt `text-brand-text`.
- [ ] 6.2 `web/src/pages/TerminePage.tsx`: falls Session-Karten die alte `rsvp_opt_out`-Info nutzen, auf neue Felder umstellen; ansonsten Typ-Alignment.
- [ ] 6.3 `web/src/pages/KalenderPage.tsx`: dito Typ-Alignment.
- [ ] 6.4 `web/src/pages/__tests__/*.tsx` (u. a. `TermineDetailPage.permissions.test.tsx`, `TerminePage.permissions.test.tsx`, `TermineDetailPage.crossteam.test.tsx`, `SpieltagDetailPage.permissions.test.tsx`, `TerminePage.todayDivider.test.tsx`, `TermineDetailPage.eventNote.test.tsx`, `TrainingEditModal.*.test.tsx`, `GameEditModal.*.test.tsx`): Fixtures/Mocks auf neue Felder umstellen.

## 7. Frontend-Tests (neu)

- [ ] 7.1 `web/src/components/__tests__/TrainingEditModal.rsvpDefaults.test.tsx`: Radio-Gruppen rendern, Auswahl wird gespeichert; `declined` deaktiviert Reason-Checkbox; gesetzte Reason-Checkbox deaktiviert `declined`.
- [ ] 7.2 `web/src/components/__tests__/GameEditModal.rsvpDefaults.test.tsx`: analog.
- [ ] 7.3 `web/src/pages/__tests__/TermineDetailPage.rsvpDefault.test.tsx`: virtuelle Default-Zeile wird dezent gerendert (`.italic`-Klasse präsent), aktive Antwort nicht.

## 8. Verifikation

- [ ] 8.1 `go build ./...` grün
- [ ] 8.2 `go test ./...` grün (Architektur-Test inkl.)
- [ ] 8.3 `pnpm -C web build` grün
- [ ] 8.4 `pnpm -C web test` grün
- [ ] 8.5 `pnpm -C web lint` grün
- [ ] 8.6 `openspec validate rsvp-defaults-per-rolle --strict` grün
- [ ] 8.7 Manuelle UI-Verifikation nach Deploy: Serie mit „Standardmäßig abgesagt" für Erweiterten Kader anlegen; Detail-Seite zeigt Erweiterte als „abgesagt (kursiv)"; Header-Zähler stimmt; Konflikt-Sperre in beiden Richtungen greift.

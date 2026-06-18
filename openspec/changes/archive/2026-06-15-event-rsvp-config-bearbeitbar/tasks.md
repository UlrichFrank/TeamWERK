## 1. Backend — `UpdateGame`

- [x] 1.1 `internal/games/handler.go`: Request-Struct in `UpdateGame` (Zeile ~731-740) um `RsvpOptOut *int` und `RsvpRequireReason *int` mit `json:"...,omitempty"` erweitern.
- [x] 1.2 UPDATE-Statement (Zeile ~784-789) so erweitern, dass die beiden Felder dynamisch ins SET aufgenommen werden, wenn der Pointer ≠ nil ist (Partial-Update). Pattern: dynamisch zusammengebautes SQL analog `UpdateMember`.
- [x] 1.3 Sicherstellen, dass der bestehende `hub.Broadcast("games")`-Aufruf erhalten bleibt.
- [x] 1.4 Test `TestUpdateGame_RsvpFlagsPersisted`: PUT mit `rsvp_opt_out=1` und `rsvp_require_reason=0` → DB-Spalten sind danach gesetzt; Response enthält neuen Wert (sofern Response Body sie ausgibt; sonst Follow-GET).
- [x] 1.5 Test `TestUpdateGame_RsvpFlagsPartialUpdate`: PUT ohne die beiden Felder → DB-Wert bleibt unverändert (kein impliziter Reset).
- [x] 1.6 Test `TestUpdateGame_RsvpFlags_PlayerForbidden`: Spieler-Token im Auth-Header → 403, DB-Spalten unverändert.

## 2. Backend — `UpdateSession`

- [x] 2.1 `internal/trainings/handler.go`: Request-Struct in `UpdateSession` (Zeile ~617-619) um `RsvpOptOut *int` und `RsvpRequireReason *int` mit `omitempty` erweitern.
- [x] 2.2 UPDATE-Statement entsprechend erweitern (Partial-Update wie 1.2).
- [x] 2.3 Sicherstellen, dass `hub.Broadcast("trainings")` erhalten bleibt.
- [x] 2.4 Test `TestUpdateSession_RsvpFlagsPersisted`, `TestUpdateSession_RsvpFlagsPartialUpdate`, `TestUpdateSession_RsvpFlags_PlayerForbidden` — analog zu Tasks 1.4–1.6.

## 3. Backend — `UpdateSeries` (Partial-Update-Anpassung)

- [x] 3.1 Verifiziert: `UpdateSeries` akzeptierte die Felder bereits, aber als plain `int` → impliziter Reset bei fehlendem Feld. Umgestellt auf `*int` mit Fallback auf aktuellen DB-Wert (Partial-Update-Semantik konform zu Spec).
- [x] 3.2 UpdateSeries-Tests separat ausgelassen, da der `testServer` in `internal/trainings/handler_test.go` `UpdateSeries` nicht mountet. Partial-Update-Semantik ist über `TestUpdateSession_RsvpFlagsPartialUpdate` exemplarisch abgedeckt; identische Logik in UpdateSeries.

## 4. Frontend — `GameEditModal`

- [x] 4.1 `web/src/components/GameEditModal.tsx`: TypeScript-Typ des Form-State um `rsvp_opt_out: boolean` und `rsvp_require_reason: boolean` erweitern.
- [x] 4.2 Form-Initialisierung: bei „Bearbeiten" mit aktuellen Game-Werten vorbelegen. (Anlegen-Pfad läuft über `KalenderPage`, dort schon implementiert.)
- [x] 4.3 Render: Sektion „RSVP-Einstellungen" am Ende des Formulars mit zwei Checkboxen, Labels „Alle Spieler standardmäßig zugesagt (Opt-Out)" und „Begründung bei Absage erforderlich".
- [x] 4.4 PUT-Payload: `rsvp_opt_out` und `rsvp_require_reason` als `0`/`1` mitsenden (boolean → int konvertieren).
- [x] 4.5 Zusätzlich: Backend `ListGames` und `GetGame` um die beiden Felder erweitert, damit Frontend sie sieht. KalenderPage `Game`-Interface ebenfalls erweitert.
- [ ] 4.6 Optional: Beim Toggle von `rsvp_opt_out` auf einen schon angelegten Termin ein Info-Banner — bewusst nicht umgesetzt (Open-Question 1).

## 5. Frontend — `AdminTrainingsPage` + `TrainingEditModal`

- [x] 5.1 `web/src/pages/AdminTrainingsPage.tsx`: `disabled={!isNewSeries}` an beiden Series-Checkboxen entfernt; Hinweis-Text „nur für neue Termine" entfällt.
- [x] 5.2 `SessionModal` um `rsvp_opt_out`/`rsvp_require_reason` erweitert; `openNewSession`/`openEditSession` setzen Vorbelegung.
- [x] 5.3 PUT-Payload für `/api/training-series/{id}` unverändert — bereits korrekt.
- [x] 5.4 PUT-Payload für `/api/training-sessions/{id}` um die zwei Felder erweitert.
- [x] 5.5 `web/src/components/TrainingEditModal.tsx`: Read-only-Checkboxen durch editierbare ersetzt; useEffect sorgt für korrekte Vorbelegung beim Scope-Wechsel.

## 6. Frontend — Detail-Badge

- [x] 6.1 `web/src/pages/TermineDetailPage.tsx`: `RsvpConfigBadges`-Komponente eingeführt; rendert „Opt-Out aktiv" (brand-info) und „Begründung Pflicht" (brand-yellow) als Pills, nur bei Flag=1.
- [x] 6.2 Sowohl Trainings-Header (nach `session.note`) als auch Spiel-Header (nach `MapsLink`) zeigen die Badges.
- [x] 6.3 Badges sind READ-only ohne Click-Handler.

## 7. Spec-Konsistenz

- [x] 7.1 Spec-Delta in `openspec/changes/event-rsvp-config-bearbeitbar/specs/rsvp-event-config/spec.md` ersetzt das „Flag eingefroren"-Scenario; wird beim Archivieren in den Haupt-Spec übernommen.

## 8. Lokaler Smoke-Test (nicht-CI)

- [ ] 8.1 Spiel anlegen ohne RSVP-Flags zu setzen → DB-Default geprüft via `SELECT id, rsvp_opt_out, rsvp_require_reason FROM games WHERE …`. *(Empfohlen vor Deploy, hier nicht ausgeführt.)*
- [ ] 8.2 Im Modal `rsvp_opt_out=1` setzen, speichern → Badge erscheint im Detail. *(Empfohlen vor Deploy.)*
- [ ] 8.3 Modal erneut öffnen → Checkbox ist vorbelegt mit aktuellem Wert. *(Empfohlen vor Deploy.)*
- [x] 8.4 Spieler-403-Schutz durch Backend-Tests `TestUpdateGame_RsvpFlags_PlayerForbidden` und `TestUpdateSession_RsvpFlags_PlayerForbidden` abgedeckt.

## 9. Commit & Archiv

- [ ] 9.1 Pro Task-Gruppe ein Conventional-Commit (`feat(games):`, `feat(trainings):`, `feat(termine):` usw.). *(Auf Anweisung des Users committen.)*
- [ ] 9.2 Abschluss-Commit: Proposal archivieren. *(Nach Smoke-Test im Live-System.)*

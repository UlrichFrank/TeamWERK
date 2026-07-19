## 1. Datenbank

- [ ] 1.1 Migration `internal/db/migrations/0NN_member_series_unavailabilities.up.sql` mit nächster freier Nummer anlegen: Tabelle `member_series_unavailabilities` (Schema aus design.md) + Indizes `idx_msu_series`, `idx_msu_member`
- [ ] 1.2 Passende `.down.sql` (DROP INDEX + DROP TABLE)
- [ ] 1.3 `make migrate-up` lokal, danach `make migrate-down`/`up` gegentesten (Reversibilität)

## 2. Backend — Domain-Logik & Ableitung

- [ ] 2.1 In `internal/trainings` einen Helper `seriesUnavailabilityApplies(memberID, seriesID, date)` bzw. eine Batch-Variante (`unavailableMemberIDsForSession(sessionID)`) implementieren, der die Ableitung aus `serien-abmeldung`-Spec (member+series+Datumsfenster, NULL offen) als reinen Lookup umsetzt
- [ ] 2.2 Unit-Test der Ableitung: innerhalb/außerhalb Fenster, NULL-Grenzen (permanent/ab-Beginn), Einzeltermin `series_id IS NULL` nie betroffen, überlappende Einträge harmlos

## 3. Backend — CRUD-Routen (Trainer-Tier)

- [ ] 3.1 Handler `ListSeriesUnavailabilities` (`GET /api/training-series/{id}/unavailabilities`): `hasTeamAccess(series.team_id)`; liefert member_id, member_name, start_date, end_date, reason, created_at
- [ ] 3.2 Handler `CreateSeriesUnavailability` (`POST .../unavailabilities`): Body `{member_id, start_date?, end_date?, reason?}`; `hasTeamAccess`; Insert via `LastInsertId()` (kein RETURNING); HTTP 201; `h.hub.Broadcast("training-unavailability-changed")`
- [ ] 3.3 Handler `DeleteSeriesUnavailability` (`DELETE .../unavailabilities/{uid}`): `hasTeamAccess`; 404 wenn `{uid}` nicht zur Serie gehört; Broadcast
- [ ] 3.4 Routen in `internal/app/router.go` im bestehenden `RequireClubFunction("trainer","sportliche_leitung")`-Sub-Router registrieren
- [ ] 3.5 Falls nötig `hub *hub.EventHub` sicherstellen (Trainings-Handler hat es bereits)

## 4. Backend — RSVP-Sperre

- [ ] 4.1 In `Respond` (`internal/trainings/handler.go:1359`) nach dem Absence-Lock die Serien-Abmelde-Ableitung prüfen und bei Treffer HTTP 403 zurückgeben (kein Insert/Upsert in `training_responses`)

## 5. Backend — Attendance-Ausschluss

- [ ] 5.1 In `SaveAttendances` (`internal/trainings/handler.go:1665`) abgemeldete Mitglieder aus dem Persist überspringen (analog trainer-only-Skip); restliche Erfassung unbeeinflusst

## 6. Backend — Statistik

- [ ] 6.1 In `loadCounts` (`internal/attendance/handler.go:117`) LEFT JOIN auf `member_series_unavailabilities` über `(member_id, ts.series_id)` + Datumsfenster; die drei Trainings-Buckets um `AND msu.id IS NULL` erweitern (Ausschluss dominiert excused)
- [ ] 6.2 In `GetMemberStats` (`handler.go:380`) betroffene Trainings-Sessions in der `events`-Liste mit `category: "unavailable"` + `reason` ausweisen (kein Zähler-Beitrag)
- [ ] 6.3 Team-Aggregat: sicherstellen, dass die Team-Quote als Ø der Pro-Spieler-Quoten berechnet wird (unterschiedliche Nenner je Spieler)

## 7. Backend — Session-Listing

- [ ] 7.1 In `ListSessions`/`GetSession` pro Mitglied `unavailable: {reason, permanent} | null` mitliefern (`permanent = end_date IS NULL`); Mitglied bleibt in der Liste sichtbar

## 8. Backend — Tests

- [ ] 8.1 Route-Tests für CRUD (siehe `## Test-Anforderungen`): Happy-Path + Fehlerfälle (403 fremdes Team, 403 Spieler, 404 falsche uid)
- [ ] 8.2 RSVP-403-Test (Spieler + Eltern), Attendance-Skip-Test, Statistik-Ausschluss-Test (inkl. Dominanz über excused), Session-Listing-`unavailable`-Feld-Test
- [ ] 8.3 Broadcast-Gate: sicherstellen, dass die neuen Mutations-Routen `training-unavailability-changed` broadcasten (sonst schlägt `internal/arch/broadcast_test.go` fehl)

## 9. Frontend — API & Serien-Bearbeitung

- [ ] 9.1 API-Calls in `web/src/lib/` bzw. der Trainings-Seite (`GET/POST/DELETE .../unavailabilities`)
- [ ] 9.2 Serien-Bearbeitung: Abschnitt „Dauerhaft abgemeldete Spieler" (Liste + „Spieler abmelden"-Modal mit Spielerauswahl, optional Zeitraum + Grund; brand-Tokens, lucide-Icons, Touch-Targets ≥ 44px)
- [ ] 9.3 `useLiveUpdates((e) => { if (e === 'training-unavailability-changed') reload() })`

## 10. Frontend — Termine & Anwesenheit

- [ ] 10.1 In `/termine` (Session-Detail) je Spieler Badge „dauerhaft abgemeldet" + Grund rendern (`<CalendarX>`/`<Ban>`), An-/Abwesenheits-Toggle für abgemeldete Spieler sperren; Trainer bekommt Lösch-/Abmelden-Aktion (Prefill Serie=aktuell, start=heute)
- [ ] 10.2 Anwesenheits-Sichten: Kategorie `unavailable` in Termin-Liste rendern; Fußnote „* dauerhaft abgemeldete Spieler zählen für ihre Termine nicht mit"
- [ ] 10.3 Spieler sieht eigenen Status (read-only, keine Aktion)

## 11. Abschluss

- [ ] 11.1 `/verify-change` (Build/Test/Lint + Invarianten: Route→Tests, Mutation→Broadcast+useLiveUpdates, brand-Tokens, lucide-Icons, Migrationsnummer, `openspec validate`)
- [ ] 11.2 Benutzerhandbuch (`docs/anleitung`) um die Serien-Abmeldung ergänzen, falls einschlägig

## Test-Anforderungen

Garantierte Invariante: Ein für eine Serie abgemeldeter Spieler kann für betroffene Sessions weder RSVP abgeben noch eine Anwesenheit erfassen lassen, und diese Sessions zählen in keiner Statistik-Säule (Ausschluss > entschuldigt).

| Route | Testname | Erwarteter Status |
|---|---|---|
| `POST /api/training-series/{id}/unavailabilities` | Trainer legt Abmeldung für eigenes Team an | 201 |
| `POST /api/training-series/{id}/unavailabilities` | Trainer eines fremden Teams abgewiesen | 403 |
| `POST /api/training-series/{id}/unavailabilities` | Spieler/Elternteil abgewiesen | 403 |
| `GET /api/training-series/{id}/unavailabilities` | Trainer listet Abmeldungen | 200 |
| `GET /api/training-series/{id}/unavailabilities` | Fremder Trainer abgewiesen | 403 |
| `DELETE /api/training-series/{id}/unavailabilities/{uid}` | Trainer löscht Abmeldung | 200/204 |
| `DELETE /api/training-series/{id}/unavailabilities/{uid}` | uid gehört nicht zur Serie | 404 |
| `POST /api/training-sessions/{id}/respond` | Abgemeldeter Spieler antwortet | 403 |
| `POST /api/training-sessions/{id}/respond` | Elternteil für abgemeldetes Kind | 403 |
| `POST /api/training-sessions/{id}/attendances` | Abgemeldeter Spieler wird übersprungen, Rest gespeichert | 200 (keine Attendance-Zeile für den Abgemeldeten) |
| `GET /api/members/{id}/attendance-stats` | Betroffene Session als `unavailable`, kein Zähler-Beitrag | 200 |
| `GET /api/teams/{id}/attendance-stats` | Abgemeldete Sessions nicht im Nenner | 200 |
| `GET /api/training-sessions/{id}` | Mitglied liefert `unavailable`-Feld, bleibt sichtbar | 200 |

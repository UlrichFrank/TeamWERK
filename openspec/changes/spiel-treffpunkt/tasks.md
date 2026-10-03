# Tasks

## 1. Datenmodell und Umrechnung

- [x] 1.1 Migration `073_game_meeting_point.up.sql`/`.down.sql` anlegen: `games.meet_offset_minutes INTEGER NULL CHECK (0..720)`, `games.meet_place TEXT NOT NULL DEFAULT '' CHECK (length<=100)`, Tabelle `pending_game_meeting_push` (design.md Entscheidung 5); verifizieren mit `go test ./internal/db/...` (Migration up/down läuft auf frischer DB)
- [ ] 1.2 `timez.MeetTime(date, startHHMM, offset)` in `internal/timez` implementieren (schneidet `date[:10]`, liefert Vortag korrekt) mit Tests `TestMeetTime_GleicherTag`, `TestMeetTime_Vortag`, `TestMeetTime_KeinAbstand`; verifizieren mit `go test ./internal/timez/...`

## 2. Schreibroute

- [ ] 2.1 `canEditGameNote` in `canEditGameInfo` umbenennen und in `UpdateGameNote` weiterverwenden; verifizieren mit grünen bestehenden `event-notes`-Tests in `internal/games`
- [ ] 2.2 Handler `UpdateGameMeeting` (`PUT /api/games/{id}/meeting`) in `internal/games` implementieren: Reihenfolge ID → Body → 404 → 403 → Validierung, Umrechnung gegen den gespeicherten Anwurf, `UPDATE games` und Upsert in `pending_game_meeting_push` (alte Werte nur beim Insert) in einer Transaktion, `broadcastGame(…, "games")`, Antwort mit `meet_time`/`meet_date`/`meet_place`; Route in `internal/app/router.go` neben `/note` eintragen. Verifizieren mit den Tests aus proposal.md Test-Anforderungen (`TestUpdateGameMeeting_*`, inkl. Prüfung „Spalten unverändert" bei 400/403)
- [ ] 2.3 Fixture für `PUT /api/games/{id}/meeting` in der Objektrechte-Matrix (`internal/permissions/object_matrix_test.go` / `object_fixtures_test.go`) ergänzen; verifizieren mit `go test ./internal/permissions/... ./internal/arch/...` (Tier-Matrix, Objektrechte, Broadcast-Gate grün)

## 3. Lesepfade Backend

- [ ] 3.1 `meet_time`, `meet_date`, `meet_place` in alle Spiel-Antworten aus `internal/games` aufnehmen (`ListGames`, `GetGame`, `ListMyGames`) über `timez.MeetTime`; verifizieren mit `TestMeetingFolgtVerlegtemAnwurf` (Treffzeit 13:30 bei Anwurf 15:00, nach `PUT /api/games/{id}` auf 17:00 → `15:30`)
- [ ] 3.2 Dashboard-Terminzeilen (`internal/dashboard`) um `meetTime` ergänzen; verifizieren mit einem Test in `internal/dashboard/handler_test.go`, der das Feld für ein Spiel mit Treffzeit prüft und für eines ohne `null`
- [ ] 3.3 Spielkopf der Mitfahrgelegenheiten (`internal/carpooling`) um `meet_place` ergänzen; verifizieren mit einem Test in `internal/carpooling`, der den Ort im Listen-Response findet
- [ ] 3.4 iCal-Feed (`internal/calendar/handler.go`): „Treffen: HH:MM Uhr[, Ort]" als erster Absatz der DESCRIPTION, Vortag-Datum bei abweichendem `meet_date`, `DTSTART` unverändert; verifizieren mit `TestCalendar_DescriptionNenntTreffzeit`

## 4. Push

- [ ] 4.1 Scheduler-Job für `pending_game_meeting_push` (`internal/scheduler/game_meeting_push.go`, im minütlichen Lauf neben `sendEventNoteReminders` aufrufen): Netto-Vergleich gegen `prev_*`, Text aus aktuellem Stand, `notify.Send(…, "games", …)` an `notify.TeamAudience`, Spieltag-Prüfung gegen `timez.Berlin()`, Zeile immer löschen; verifizieren mit `TestMeetingPush_*` aus proposal.md (Datei-DB wegen Goroutinen, siehe 07-testing)
- [ ] 4.2 Spielerinnerung 24 h/3 h (`internal/scheduler/scheduler.go`) um „ · Treffen HH:MM Uhr[, Ort]" erweitern; verifizieren mit `TestGameReminder_NenntTreffzeit` und einem Test ohne Treffzeit (kein Treffzeit-Teil)

## 5. Frontend

- [ ] 5.1 Fehlercodes `meet_after_start`, `meet_offset_out_of_range`, `meet_place_too_long`, `meet_place_without_time` in `web/src/lib/errors.ts` übersetzen; Felder `meet_time`/`meet_date`/`meet_place` in den Spiel-Typen ergänzen; verifizieren mit `pnpm -C web build` (Typecheck)
- [ ] 5.2 Anzeige-Komponente `MeetingPointLine` (lucide `Flag`, brand-Tokens, Vortag-Hinweis) mit Vitest-Test (mit/ohne Ort, Vortag, ohne Treffzeit rendert nichts); verifizieren mit `pnpm -C web test MeetingPointLine`
- [ ] 5.3 `MeetingPointEditor` (Uhrzeitfeld, Ort `maxLength=100`, Speichern, Entfernen, Hinweis „verschiebt sich mit dem Anwurf", Klassen aus `buttonStyles.ts`) mit Vitest-Test (sendet korrekten Body, zeigt übersetzten Fehler, Entfernen sendet leere Werte); verifizieren mit `pnpm -C web test MeetingPointEditor`
- [ ] 5.4 Anzeige und Editor in `EventInfoModal` (neben `EventNoteEditor`, nur bei `can_edit`) und `TermineDetailPage` (unter dem Anwurf) einbauen; Anzeige in `TerminePage`-Spielkarte und `DashboardPage`-Untertitel („(Treffen HH:MM)"); verifizieren mit je einem Vitest-Test für Detailseite (Editor nur für Berechtigte) und Terminliste (Zeile erscheint/fehlt)
- [ ] 5.5 `MitfahrgelegenheitenPage`: `FormModal` befüllt `treffpunkt` bei neuem Eintrag mit `meet_place` vor, bestehende Einträge unverändert; verifizieren mit Vitest-Test für beide Fälle
- [ ] 5.6 Gates prüfen: `pnpm -C web lint` und `pnpm -C web test` (Design-Token-, Typografie-, Button-Gate grün)

## 6. Dokumentation und Integration

- [ ] 6.1 Gotcha-Absatz „Spiel-Treffzeit" in `docs/agent/06-gotchas.md`: Speicherung als Abstand, `timez.MeetTime` als einzige Umrechnung, Netto-Vergleich der Debounce-Push; verifizieren durch Lesen im Diff
- [ ] 6.2 Integration: `/verify-change` ausführen (Build, `go test ./...`, golangci-lint, `pnpm -C web build/test/lint`, `openspec validate spiel-treffpunkt --strict`) und manuell im laufenden System prüfen: Treffzeit setzen, Anwurf im `GameEditModal` verlegen, Treffzeit in Liste/Detail/Modal/Dashboard verschoben, iCal-Feed zeigt den Absatz

## 1. Backend — Konflikt-Sperre entfernen

- [x] 1.1 `internal/trainings/handler.go`: `rsvpSettingsConflict` und `writeInvalidRsvpSettings` entfernen; die zugehörigen 400-Aufrufe in `CreateSeries`, `UpdateSeries`, `CreateSession`, `UpdateSession` streichen (Werte werden ohne Konfliktprüfung gespeichert; `validRsvpDefault`-Prüfung bleibt).
- [x] 1.2 `internal/games/handler.go`: `rsvpSettingsConflict`/`writeInvalidRsvpSettings` und die 400-Aufrufe in `CreateGame` und `UpdateGame` entfernen (`validRsvpDefault` bleibt).

## 2. Backend — Trainer-`my_rsvp`-Default in den Listen

- [x] 2.1 `internal/trainings/handler.go` `ListSessions`: `my_rsvp`-`CASE` um einen Trainer-Zweig erweitern — ist der User Trainer des Team-Kaders (`kader_trainers` für `ts.team_id`/`ts.season_id`) und existiert keine explizite Response, `default_rsvp='confirmed'`. Priorität: Response > Stammkader > Erweitert > Trainer > NULL. Trainer-Default darf **nicht** in die `confirmed_count`-Aggregation einfließen (unverändert).
- [x] 2.2 `internal/games/handler.go` `ListMyGames`: analoger Trainer-Zweig für `my_rsvp` (Trainer eines beteiligten Teams via `kader_trainers`), Header-Zähler unverändert.

## 3. Frontend — Editor-Kopplung lösen

- [x] 3.1 `web/src/components/RsvpDefaultsEditor.tsx`: die gegenseitige `disabled`-Logik + `title`-Tooltip zwischen `declined`-Radios und Reason-Checkbox entfernen; `anyDeclined`-Sperre streichen. Beide Kontrollen immer bedienbar.

## 4. Frontend — Trainer-Buttons auf /termine

- [x] 4.1 `web/src/pages/TerminePage.tsx`: die pauschale `!isTrainer`-Bedingung an den RSVP-Button-Blöcken (Trainings + Spiele) durch ein Teilnahme-Signal ersetzen — Buttons rendern, wenn `my_rsvp` nicht-null ist (bzw. für den Nicht-Eltern-Zweig entsprechend). Für den Eltern-Zweig unverändert.
- [x] 4.2 `web/src/pages/TerminePage.tsx`: „Zusagen"-Toggle-Logik so anpassen, dass sie für Trainer nicht an `rsvp_default_players` gekoppelt ist (Trainer → einfaches Setzen auf `confirmed`); Spieler-Verhalten unverändert.

## 5. Tests — Backend

- [x] 5.1 `internal/trainings/rsvp_defaults_test.go`: Konflikt-Test (`TestRsvpDefault_ConflictRejected`) entfernen bzw. in „Kombination wird akzeptiert" (200/204, Werte gespeichert) umschreiben.
- [x] 5.2 `internal/games/rsvp_defaults_test.go`: Konflikt-Tests (`Create`/`Update`ConflictRejected) analog umschreiben auf „akzeptiert".
- [x] 5.3 `internal/trainings/rsvp_defaults_test.go`: neuer Test — Trainer des Teams ohne Response → `GET /api/training-sessions` liefert `my_rsvp='confirmed'`; Vorstand/fremd → `my_rsvp=null`.
- [x] 5.4 `internal/games/rsvp_defaults_test.go`: neuer Test — Trainer eines beteiligten Teams ohne Response → `GET /api/games/my` liefert `my_rsvp='confirmed'`; Header-Zähler bleibt unverändert.

## 6. Tests — Frontend

- [x] 6.1 `web/src/components/__tests__/TrainingEditModal.rsvpDefaults.test.tsx` + `GameEditModal.rsvpDefaults.test.tsx`: Konflikt-Tests entfernen; stattdessen prüfen, dass gesetzte Reason-Checkbox die `declined`-Radios **aktiv** lässt und umgekehrt.
- [x] 6.2 `web/src/pages/__tests__/`: Test für `TerminePage` — Trainer eines Team-Termins (`my_rsvp='confirmed'`) sieht RSVP-Buttons; Nicht-Teilnehmer (`my_rsvp=null`) sieht keine.

## 7. Verifikation

- [x] 7.1 `go build ./...` grün
- [x] 7.2 `go test ./...` grün
- [x] 7.3 `pnpm -C web build` grün
- [x] 7.4 `pnpm -C web test` grün
- [x] 7.5 `pnpm -C web lint` grün
- [x] 7.6 `openspec validate termine-rsvp-nachbesserung --strict` grün
- [ ] 7.7 Manuelle UI-Verifikation: Termin mit `declined` + „Begründung erforderlich" anlegen (kein Fehler); als Trainer auf `/termine` zu-/absagen bei eigenem Team; als Vorstand keine Buttons auf fremdem Team-Termin.

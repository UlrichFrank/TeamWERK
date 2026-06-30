## 1. Training: children_rsvp um erw. Kader erweitern

- [x] 1.1 `attachChildrenRSVPToSessions` (`internal/trainings/handler.go`) um UNION-Zweig auf `kader_extended_members` erweitern (mit `NOT EXISTS (kader_members …)`-Guard gegen Doppelung) und `0/1 AS is_extended` mitführen
- [x] 1.2 Auto-Confirm bei `rsvp_opt_out=1` nur für `is_extended == 0` anwenden (Mirror von `GetAttendances`)
- [x] 1.3 Test (Happy): Elternteil mit Kind nur im erw. Kader → `children_rsvp` enthält das Kind (`internal/trainings/erw_kader_eltern_test.go`)
- [x] 1.4 Test (Regel): erw.-Kader-Kind bei `rsvp_opt_out=1` ohne Response → `rsvp: null`; Stammkader-Kind → `confirmed`
- [x] 1.5 Test (Dedup): Kind in Stamm- **und** erw. Kader desselben Teams → genau ein `children_rsvp`-Eintrag

## 2. Spiel: children_rsvp um erw. Kader erweitern

- [x] 2.1 `attachChildrenRSVPToGames` (`internal/games/handler.go`) analog um UNION-Zweig auf `kader_extended_members` (+ Dedup-Guard, `is_extended`-Flag) erweitern
- [x] 2.2 Auto-Confirm bei `rsvp_opt_out=1` nur für `is_extended == 0`
- [x] 2.3 Test (Happy): Elternteil mit erw.-Kader-Kind → `GET /api/games/my` liefert `children_rsvp` mit dem Kind (`internal/games/erw_kader_eltern_test.go`)
- [x] 2.4 Test (Regel): erw.-Kader-Kind ohne Response bei `rsvp_opt_out=1` → `rsvp: null`

## 3. Eltern können für erw.-Kader-Kind antworten (Verifikation des Pfads)

- [x] 3.1 Test (Happy): Elternteil `POST /api/training-sessions/{id}/respond` mit erw.-Kader-Kind-`member_id` → 204, `training_responses`-Eintrag vorhanden
- [x] 3.2 Test (Happy): Elternteil `POST /api/games/{id}/respond` mit erw.-Kader-Kind-`member_id` → 204, `game_responses`-Eintrag vorhanden
- [~] 3.3 Test (Fehlerfall): Elternteil sendet `member_id` eines nicht verknüpften Kindes → 403 — **NICHT umgesetzt / außerhalb Scope.** Befund: `users.role` ist nur `admin|standard`; die `Respond`-Switch-Zweige `case "spieler"/"elternteil"` sind toter Code, ein Standard-Parent landet im `default`-Zweig **ohne** `parentHasChild`-Prüfung. Der 403-Pfad ist also gar nicht erreichbar — eine **bestehende** Authz-Lücke (jeder eingeloggte User kann für jede `member_id` antworten), unabhängig von diesem Fix. Separater Change empfohlen (betrifft `eltern-rsvp`-Spec „403 für fremdes Kind").
- [x] 3.4 Test (Sichtbarkeit): erw.-Kader-Mitglieder erscheinen in `GET /api/training-sessions/{id}/attendances` mit `is_extended: true` — bereits durch Capability `erweiterter-kader-trainings-access` (Bestandstests) abgedeckt; voller Suite-Lauf grün.

## 4. Teamfilter: erw.-Kader-Teams für Eltern

- [x] 4.1 Eltern/Spieler-Zweig von `ListTeamsForUser` (`internal/games/handler.go`) von `team_memberships` auf `user_accessible_teams` (aktive Saison) umstellen
- [x] 4.2 Test (Happy): Elternteil eines erw.-Kader-Kindes → `GET /api/teams` enthält das Team
- [x] 4.3 Test (Negativ): Elternteil ohne Kader-Bezug → Team nicht enthalten
- [x] 4.4 Test (Regression): Stammkader-Eltern + Trainer/Vorstand/sportliche_leitung → Teamliste unverändert (bestehende `/api/teams`-Tests, voller Suite-Lauf grün)

## 5. Verifikation & Abschluss

- [x] 5.1 `go test ./...` (1014 grün, inkl. Architektur-Test) + `go vet` + `golangci-lint` + `gofmt` sauber
- [ ] 5.2 Manuell gegen reproduzierbaren Fall prüfen: Elternteil eines erw.-Kader-Kindes sieht auf `/termine` Zu-/Absagen-Buttons, Status auf Detailseite, Team im Filter (am laufenden System nach Deploy)
- [x] 5.3 `openspec validate erw-kader-eltern-rueckmeldung --strict` grün; Proposal nach Abnahme archivieren (Archivierung steht noch aus)

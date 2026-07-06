## 1. Backend — Listen-Endpoints (`my_reason`, `children_rsvp[].reason`)

- [x] 1.1 `internal/games/handler.go`: `gameListItem` um `MyReason *string \`json:"my_reason,omitempty"\`` erweitern; `childRSVP` um `Reason *string \`json:"reason,omitempty"\``.
- [x] 1.2 `ListMyGames`-SQL (Zeile ~1943): zusätzliche Subquery `SELECT reason FROM game_responses WHERE game_id=g.id AND member_id=?` mitziehen, als `sql.NullString` scannen, `MyReason` nur setzen wenn valid + nicht leer. **Nicht** aus `defaultRSVP` ableiten — Reason gehört nur zu expliziten Antworten.
- [x] 1.3 `attachChildrenRSVPToGames`-SQL (Zeile ~2551 + ~2564): `gr.reason` in beide UNION-Zweige aufnehmen, als 6. Scan-Spalte in der Rows-Loop. Nur setzen, wenn Kind explizit geantwortet hat (also `rsvp.Valid == true`).
- [x] 1.4 `internal/trainings/handler.go`: `sessionListItem` um `MyReason`, `childRSVP` um `Reason` erweitern (analog 1.1).
- [x] 1.5 `ListSessions`-SQL (Zeile ~1011): zusätzliche Subquery `SELECT reason FROM training_responses WHERE training_id=ts.id AND member_id=?` mitziehen; Scan + Setz-Logik analog.
- [x] 1.6 Children-Aggregations-Query in Trainings (Nähe Zeile ~1090): `tr.reason` mitziehen, analog zu 1.3.

## 2. Backend — Attendance-Endpoints (`canSeeReason` konsistent gaten)

- [x] 2.1 `internal/trainings/handler.go` `GetAttendances` (Zeile ~1406): am Anfang `memberID, _ := h.memberIDForUser(...)` und `childMemberIDs`-Map (nur wenn `claims.IsParent`) analog zu `GetSession` vorab laden.
- [x] 2.2 Reason-Gate in derselben Funktion (Zeile 1541) auf `isTrainerLike || (memberID > 0 && item.MemberID == memberID) || childMemberIDs[item.MemberID]` erweitern.
- [x] 2.3 `internal/games/handler.go` `GetAttendances` (Zeile ~2793): `memberID` und `childMemberIDs` vorab laden (heute gar nicht vorhanden — der Endpoint hat null Reason-Gating).
- [x] 2.4 Reason-Gate in derselben Funktion (Zeile 2938) einführen: `if canSeeReason && reason.Valid && reason.String != "" { item.Reason = &reason.String }`. Regressionstest speziell für den bisherigen Leak (siehe 3.4).

## 3. Backend — Tests

- [x] 3.1 `internal/games/reason_visibility_test.go`: `TestListMyGames_MyReason_Populated_When_RespondedWithReason`, `TestListMyGames_MyReason_Absent_When_DefaultRsvp`, `TestListMyGames_ChildrenReason_ForParent`, `TestListMyGames_ChildrenReason_OmittedWhenEmpty`.
- [x] 3.2 `internal/trainings/reason_visibility_test.go`: `TestListSessions_MyReason_Populated_When_RespondedWithReason`, `TestListSessions_MyReason_Absent_When_DefaultRsvp`, `TestListSessions_ChildrenReason_ForParent`.
- [x] 3.3 `internal/trainings/reason_visibility_test.go`: `TestGetTrainingAttendances_Reason_Trainer_SeesAll`, `TestGetTrainingAttendances_Reason_Member_SeesOwn`, `TestGetTrainingAttendances_Reason_Parent_SeesChild`, `TestGetTrainingAttendances_Reason_Foreigner_Hidden` (via Extended-Kader-Nutzer, weil `user_accessible_teams` eine View ist).
- [x] 3.4 `internal/games/reason_visibility_test.go`: `TestGetGameAttendances_Reason_Trainer_SeesAll`, `TestGetGameAttendances_Reason_SportlicheLeitung_HidesForeignReason` (Regression gegen historischen Leak). Member- und Parent-Zugriff auf `/attendances` sind faktisch gesperrt (`canRecordGameAttendance` = admin/sL/team-Trainer), weshalb die Own/Parent-Fälle über `/participants` gedeckt werden — siehe existierende `TestListGameResponses`-Tests.

## 4. Frontend — TerminePage.tsx

- [x] 4.1 Interfaces `Session` und `Game` in `web/src/pages/TerminePage.tsx` erweitern: `my_reason?: string`; `children_rsvp[i]` um optionales `reason?: string`.
- [x] 4.2 Karten-Rendering (Trainingssession, ~Zeile 465–530): wenn `s.my_reason` gesetzt und `s.my_rsvp in ('declined','maybe')`, darunter eine dezente Zeile mit `MessageCircle`-Icon + Text.
- [x] 4.3 Karten-Rendering (Spiel, ~Zeile 580–640): analog für `g.my_reason`.
- [x] 4.4 Kind-Zeilen-Rendering: pro `child in children_rsvp`, wenn `child.reason` gesetzt und `child.rsvp in ('declined','maybe')`, ebenfalls dezente Zeile darunter.

## 5. Frontend — Tests

- [x] 5.1 `web/src/pages/TerminePage.test.tsx`: 3 Tests (my_reason gerendert, Feld fehlt → keine Zeile, Kind-Reason im Payload für Elternteil).

## 6. Manuelle Verifikation + Abschluss

- [ ] 6.1 Lokal `go run ./cmd/teamwerk` + `pnpm dev`, mit Test-User auf `/termine` absagen mit Grund → Grund erscheint sofort auf Karte. (offen für Nutzer-seitige Verifikation)
- [ ] 6.2 Als Trainer-User `/termine/training/{id}` — alle Reasons sichtbar; als Player-User dieselbe Seite — nur eigene Reason; als Elternteil-User — Kind-Reason zusätzlich zur eigenen. (offen für Nutzer-seitige Verifikation)
- [x] 6.3 Build/Test/Lint durchlaufen: `go test ./...` (1234 grün), `pnpm build` (ok), `pnpm test` (510/510), `golangci-lint run ./...` (no issues), `pnpm lint` (keine neuen Findings in geänderten Dateien).
- [ ] 6.4 Commit(s) im OpenSpec-Konventions-Format (siehe unten).
- [ ] 6.5 Archive-Change ausführen (Move nach `openspec/changes/archive/`), wenn PR gemerged.

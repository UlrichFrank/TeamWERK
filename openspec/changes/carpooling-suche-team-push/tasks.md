## 1. Backend: Empfänger-Auflösung

- [x] 1.1 In `internal/carpooling/handler.go` private Helfer `qualifyingTeamsForNextGame(ctx, gameID, seasonID) ([]int, error)` ergänzen — liefert die Untermenge der `game_teams`-Teams, für die das Spiel das nächste anstehende ist.
- [x] 1.2 Private Helfer `kaderRecipients(ctx, teamIDs []int, seasonID int, excludeUserID int) ([]int, error)` — Eltern via `family_links` über `kader_members ∪ kader_extended_members` sowie Trainer via `kader_trainers` → `members.user_id`. DISTINCT, `excludeUserID` raus.
- [x] 1.3 Beide Helfer mit Fokus auf einen einzelnen DB-Roundtrip pro Aufruf (`IN`-Klausel statt N×Query).

## 2. Backend: Trigger im Handler

- [x] 2.1 In `Upsert` nach dem bestehenden `go h.notifyOpposite(...)` einen zweiten Goroutine-Block einsetzen, der bei `body.Typ == "suche" && isNewEntry` läuft.
- [x] 2.2 Im neuen Block: `seasonID` aus dem Spiel laden, `qualifyingTeamsForNextGame` aufrufen, bei leerem Ergebnis still beenden.
- [x] 2.3 `kaderRecipients(...)` aufrufen, bei leerem Ergebnis still beenden.
- [x] 2.4 `notify.Send(h.db, h.cfg, recipients, "carpooling", "Mitfahrgelegenheit", body, "/mitfahrgelegenheiten")` mit identischem Body-Format wie `notifyOpposite` (`"%s sucht eine Mitfahrgelegenheit zu %s, %s"`).

## 3. Tests

- [x] 3.1 `TestCarpooling_SucheInsert_NextGame_TeamPushFanOut` — Happy Path: Trainer + zwei Eltern (je 1× regulärer, 1× erweiterter Kader) → 3 Empfänger, Steller raus.
- [x] 3.2 `TestCarpooling_SucheInsert_NotNextGame_NoTeamPush` — zwei Spiele, Suche zum späteren → kein Team-Push.
- [x] 3.3 `TestCarpooling_SucheUpdate_NoTeamPush` — Insert, dann Update derselben Suche → genau ein Team-Push insgesamt.
- [x] 3.4 `TestCarpooling_SucheInsert_NoKaderSilent` — Spiel ohne `kader`-Zeile für `(team_id, season_id)` → kein Team-Push, HTTP 204 wie gewohnt.
- [x] 3.5 `TestCarpooling_SucheInsert_MultiTeamGame` — Spiel mit zwei Teams; nur Team A qualifiziert (nächstes Spiel), Team B nicht. Empfängerkreis enthält nur A's Eltern/Trainer.
- [x] 3.6 `TestCarpooling_SucheInsert_BieteTyp_NoTeamPush` — `typ='biete'` löst keinen Team-Push aus.
- [x] 3.7 `TestCarpooling_SucheInsert_SelfExcluded` — Steller ist gleichzeitig Trainer des Kaders → nicht in der Push-Liste.
- [x] 3.8 `TestCarpooling_SucheInsert_PrefRespected` — User mit `notification_preferences (category='carpooling', push_enabled=0)` empfängt keinen Push (Verifikation, dass der Fan-out `notify.Send` nutzt und nicht direkt `push.SendToUsers`).

## 4. Verifikation & Abschluss

- [x] 4.1 `make test` grün (inkl. neuer Tests).
- [x] 4.2 `golangci-lint run ./...` grün.
- [x] 4.3 `openspec validate carpooling-suche-team-push --strict` grün.
- [ ] 4.4 Manuell: zweite Suche zum selben Spiel desselben Users → kein zweiter Push (UPDATE-Pfad).
- [ ] 4.5 Manuell: Suche zum übernächsten Spiel des Teams → kein Team-Push (nur `notifyOpposite`).
- [ ] 4.6 OpenSpec-Change archivieren.

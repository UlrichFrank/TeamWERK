## 1. Backend — `GetParticipants`

- [x] 1.1 `internal/games/handler.go`: SELECT in `GetParticipants` so anpassen, dass bei `g.rsvp_opt_out=1` und `is_extended=0` und fehlendem Response-Eintrag `rsvp_status='confirmed'` zurückgegeben wird. Game-Tabelle joinen, um `rsvp_opt_out` zu lesen.
- [x] 1.2 Extended-Member-Pfad bleibt: `rsvp_status=NULL`.

## 2. Backend — `ListGames`

- [x] 2.1 SELECT in `ListGames` so anpassen, dass `confirmed_count` die opt-out-Logik anwendet — analog zu `ListMyGames`, aber über `kader_members` statt `team_memberships`.
- [x] 2.2 `declined_count` und `maybe_count` bleiben simple Counts.

## 3. Backend — `GetGame`

- [x] 3.1 Response-Struct um `ConfirmedCount`, `DeclinedCount`, `MaybeCount` ergänzen.
- [x] 3.2 SELECT erweitern, dass die drei Counts (opt-out-aware für confirmed) berechnet werden.

## 4. Backend — `ListMyGames` Harmonisierung

- [x] 4.1 SQL-CASE in `ListMyGames` für `confirmed_count`: von `team_memberships`-Join auf `kader_members`-Join umstellen.
- [x] 4.2 Sicherstellen: `inRegularKader`-EXISTS bleibt unverändert (nutzt schon `kader_members`).

## 5. Tests

- [x] 5.1 `TestListGames_OptOutCountsKaderImplicit`: Spiel mit `rsvp_opt_out=1`, 3 Kader-Members, 0 Responses → `confirmed_count=3`.
- [x] 5.2 `TestListGames_OptOutWithDeclinedNotCounted`: 3 Kader-Members, 1 explizit `declined` → `confirmed_count=2`, `declined_count=1`.
- [x] 5.3 `TestGetParticipants_OptOutMarksKaderConfirmed`: Kader-Member ohne Response → `rsvp_status='confirmed'`.
- [x] 5.4 `TestGetParticipants_OptOutExtendedRemainsNull`: Extended-Member ohne Response → `rsvp_status=null`.
- [x] 5.5 `TestGetParticipants_NoOptOutBehavesAsBefore`: `rsvp_opt_out=0` → `rsvp_status` nur bei expliziter Response gesetzt.
- [x] 5.6 `TestGetGame_ReturnsCounts`: Response enthält `confirmed_count`, `declined_count`, `maybe_count`.
- [-] 5.7 `TestListMyGames_OptOutUsesKaderMembers` — abgedeckt durch bestehende Tests `TestListMyGames_RegularKaderAutoConfirmBleibt` und `TestListMyGames_ExtendedKaderKeinAutoConfirm`, die nach der Umstellung weiter grün laufen.

## 6. Build & Validierung

- [x] 6.1 `/usr/local/go/bin/go test ./...` — komplette Suite grün.
- [x] 6.2 `pnpm tsc --noEmit` in `web/` — keine Fehler.

## 7. Commits

- [ ] 7.1 Conventional Commits pro Bereich: `docs(openspec)`, `fix(games)` für die Endpoint-Konsistenz.

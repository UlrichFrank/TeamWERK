## 1. Frontend — kein Team an Mehr-Team-Terminen

- [x] 1.1 `SpieltagDetailModal.tsx`: `team_id` nur setzen, wenn der Termin genau ein Team hat (`game.teams`), sonst `null`
- [x] 1.2 Vitest: Slot-Anlage an Termin mit drei Teams sendet `team_id: null`; an Termin mit einem Team dessen ID

## 2. Backend — Benachrichtigung bei team-losem, spielgebundenem Slot

- [x] 2.1 `duties.eligibleDutyUsers` auf Team-Menge (`[]int`) umstellen
- [x] 2.2 `CreateSlot`: Team-Scope auflösen (`team_id` → [id]; sonst `game_id` → `game_teams`; sonst leer)
- [x] 2.3 Go-Test: `POST /api/duty-slots` mit `team_id: null` + `game_id` benachrichtigt beide Team-Mengen, nicht das unbeteiligte Team

## 3. Backend — Dienst-Erinnerung (T-2)

- [x] 3.1 `openSlot` um `gameID` erweitern, `ds.game_id` mitselektieren
- [x] 3.2 `scheduler.eligibleUsers`: Team-Scope aus `team_id` bzw. `game_teams` auflösen, vereinsweit nur ohne beides
- [x] 3.3 Go-Test: team-loser Slot an Mehr-Team-Spiel erinnert genau die beteiligten Teams

## 4. Verifikation

- [x] 4.1 `make test`, `pnpm -C web test`, `openspec validate --strict`
- [x] 4.2 Datenkorrektur für Bestands-Slots vorbereiten (SQL, nach Bestätigung auf Prod)

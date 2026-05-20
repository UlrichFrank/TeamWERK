## 1. Backend — Board-Handler überarbeiten

- [ ] 1.1 Hilfsfunktion `myTeamIDs(db, userID) []int` anlegen: liefert alle `team_id`s der aktiven Saison wo der User direkt (via `members.user_id`) oder als Elternteil (via `family_links`) Mitglied ist
- [ ] 1.2 `Board`-Handler neu schreiben: lädt alle `duty_slots` (inkl. vergangener) für die Team-IDs der aktiven Saison, joined auf `games` und `duty_types`, setzt `claimed_by_me` via LEFT JOIN auf `duty_assignments WHERE user_id=?`
- [ ] 1.3 Ergebnis zu `[]Group` aggregieren: Slots mit `game_id` in Spielgruppe (sortiert nach `game.date`, `game.time`), Slots ohne `game_id` in „Sonstige Dienste"-Gruppe pro Team; `past`-Flag setzen (`date < today`)
- [ ] 1.4 Response als JSON senden

## 2. Backend — Unclaim-Handler

- [ ] 2.1 `Unclaim`-Handler in `internal/duties/handler.go` anlegen: liest `slotId` aus Pfad, prüft ob Assignment für `user_id` existiert (404 sonst), prüft ob `status != 'fulfilled'` (409 sonst), löscht Assignment, dekrementiert `slots_filled`, ruft `updateAccount` mit subtract auf
- [ ] 2.2 Route `DELETE /api/duty-board/{slotId}/claim` in `cmd/teamwerk/main.go` registrieren (authenticated)

## 3. Frontend — DutyBoardPage neu

- [ ] 3.1 Typen anlegen: `DutySlot { id, duty_type, event_time, slots_total, vacancies, claimed_by_me, role_desc }`, `DutyGroup { game_id, date, event_time, opponent, team_name, label, past, slots }`
- [ ] 3.2 State: `groups: DutyGroup[]`, `showPast: boolean` (default false); `load()` ruft `GET /api/duty-board` ab
- [ ] 3.3 Kachel-Komponente pro Gruppe: Header mit Datum + Gegner (oder Label) + Mannschaftsname; darunter Tabelle mit Spalten Dienst / Uhrzeit / Plätze / Aktion
- [ ] 3.4 Slot-Zeilenstatus: `claimed_by_me && !past` → Austragen-Button; `vacancies > 0 && !past` → Eintragen-Button; `vacancies == 0 && !claimed_by_me` → „Besetzt"; `past && claimed_by_me` → „Eingetragen" (kein Button)
- [ ] 3.5 Claim (`POST`) und Unclaim (`DELETE`) mit `api.post` / `api.delete`, danach `load()` aufrufen; Fehler mit `alert`
- [ ] 3.6 „Vergangene Spieltage einblenden/ausblenden"-Button oberhalb der Kacheln; filtert `groups.filter(g => showPast || !g.past)`
- [ ] 3.7 Hinweistext wenn gefiltertes Ergebnis leer: „Keine offenen Dienste für deine Mannschaften."

## 1. DB-Migrationen

- [ ] 1.1 Migration `018_erweiterter_kader.up.sql` anlegen: `CREATE TABLE kader_extended_members (kader_id, member_id, added_at, PRIMARY KEY(kader_id, member_id))`
- [ ] 1.2 Migration `018_erweiterter_kader.down.sql` anlegen: `DROP TABLE IF EXISTS kader_extended_members`
- [ ] 1.3 Migration `019_game_lineup.up.sql` anlegen: `CREATE TABLE game_lineup (game_id, member_id, added_by, added_at, PRIMARY KEY(game_id, member_id))`
- [ ] 1.4 Migration `019_game_lineup.down.sql` anlegen: `DROP TABLE IF EXISTS game_lineup`
- [ ] 1.5 `make migrate-up` lokal — keine Fehler

## 2. Backend: Kader — Erweiterter Kader

- [ ] 2.1 `kaderDetail`-Struct in `kader/handler.go` um `ExtendedMembers []memberRow` ergänzen
- [ ] 2.2 Funktion `loadExtendedMembers(ctx, kaderId)` analog zu `loadMembers` implementieren
- [ ] 2.3 `loadExtendedMembers` in `ListKader` und `GetKader` aufrufen und Ergebnis befüllen
- [ ] 2.4 `UpdateKader` (PUT /api/admin/kader/{id}): Request-Struct um `ExtendedMembersAdd []int` und `ExtendedMembersRemove []int` erweitern
- [ ] 2.5 Insert-/Delete-Logik für `kader_extended_members` in `UpdateKader` implementieren

## 3. Backend: Games — Participants & Lineup

- [ ] 3.1 Response-Struct `participantItem` definieren: `member_id`, `member_name`, `is_extended`, `rsvp_status` (nullable), `in_lineup`
- [ ] 3.2 Handler `GetParticipants` implementieren (`GET /api/games/{id}/participants`): UNION aus `kader_members` + `kader_extended_members` des Teams (aktive Saison), LEFT JOIN `game_responses` + `game_lineup`
- [ ] 3.3 Handler `SaveLineup` implementieren (`POST /api/games/{id}/lineup`): bulk upsert + delete-diff in `game_lineup`; nur Trainer/Admin erlaubt
- [ ] 3.4 Beide Routen in `cmd/teamwerk/main.go` registrieren
- [ ] 3.5 `SaveLineup` ruft `h.hub.Broadcast("games")` auf

## 4. Frontend: AdminKaderPage — Erweiterter Kader

- [ ] 4.1 `Kader`-Interface in `AdminKaderPage.tsx` um `extended_members: Member[]` erweitern
- [ ] 4.2 Handler `handleAddExtendedMember` und `handleRemoveExtendedMember` implementieren (analog zu Trainer-Handler, nutzen `extended_members_add` / `extended_members_remove`)
- [ ] 4.3 Abschnitt „Erweiterter Kader" unterhalb des Mitglieder-Abschnitts in der Kader-Card rendern: Suchfeld (neuer Component `KaderExtendedSearch` — kann KaderMemberSearch als Vorlage nutzen) + Liste mit × je Eintrag

## 5. Frontend: TermineDetailPage — Aufstellung

- [ ] 5.1 Interface `ParticipantItem` definieren: `member_id`, `member_name`, `is_extended`, `rsvp_status`, `in_lineup`
- [ ] 5.2 Datenquelle für Spieldetail auf `GET /api/games/{id}/participants` umstellen (statt `/games/{id}/responses`)
- [ ] 5.3 State `lineupMap: Record<number, boolean>` analog zu `attendanceMap` für Training einführen
- [ ] 5.4 Funktion `saveLineup` implementieren: sendet gesamte Lineup-Liste an `POST /api/games/{id}/lineup`
- [ ] 5.5 Spalte „Aufstellung" in `ResponseTable` (oder neue `GameTable`) hinzufügen: Trainer → Checkbox mit `onChange → saveLineup`; andere → `<Check>` oder `–`
- [ ] 5.6 Erweiterte Kader-Mitglieder (`is_extended: true`) in der Tabelle visuell kennzeichnen (z. B. kleines Badge „Erw." neben dem Namen)

## 6. Verifikation

- [ ] 6.1 `go build ./...` — keine Fehler
- [ ] 6.2 Kader-Seite: erweitertes Mitglied hinzufügen, erscheint in Liste, × entfernt es
- [ ] 6.3 Spieldetail: alle Kader-Mitglieder erscheinen (inkl. ohne RSVP)
- [ ] 6.4 Spieldetail: Trainer setzt Aufstellung → Häkchen erscheinen sofort bei Spieler/Eltern
- [ ] 6.5 Erweitertes Kader-Mitglied taucht NICHT in Training-Attendance auf

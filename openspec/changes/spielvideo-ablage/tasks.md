## 1. Datenbank

- [ ] 1.1 Migration `006_spielvideos.up.sql` anlegen: Tabelle `videos` (id, youtube_id CHAR(11), title, description, game_date, team_id FK nullable, visibility CHECK('vereinsweit','team'), created_by FK, created_at)
- [ ] 1.2 Migration `006_spielvideos.down.sql` anlegen
- [ ] 1.3 `make migrate-up` lokal ausführen und Schema prüfen

## 2. Backend — Package & Handler

- [ ] 2.1 Package `internal/videos/` anlegen mit `Handler struct{ db *sql.DB }`
- [ ] 2.2 `NewHandler(db)` implementieren
- [ ] 2.3 Hilfsfunktion `canManage(claims)` — true für admin, trainer-Rolle oder trainer-Funktion
- [ ] 2.4 `GET /api/videos` — Videoliste mit `embed_url` und `thumbnail_url`, gefiltert nach Berechtigung
- [ ] 2.5 `POST /api/videos` — Video erfassen, `youtube_id` auf genau 11 Zeichen validieren
- [ ] 2.6 `PUT /api/videos/:id` — Metadaten aktualisieren
- [ ] 2.7 `DELETE /api/videos/:id` — DB-Eintrag löschen
- [ ] 2.8 Routen in `cmd/teamwerk/main.go` registrieren (authenticated-Gruppe)

## 3. Frontend — Videoliste

- [ ] 3.1 `web/src/pages/VideosPage.tsx` anlegen: Videoliste mit Thumbnail und Titel
- [ ] 3.2 YouTube-Embed via `<iframe src={embed_url}>` beim Klick auf ein Video
- [ ] 3.3 Formular zum Erfassen neuer Videos (Admin/Trainer): Titel, YouTube-ID, Datum, Team, Sichtbarkeit
- [ ] 3.4 Löschen-Button für Admin/Trainer mit Bestätigungsdialog
- [ ] 3.5 Route `/videos` in `App.tsx` eintragen
- [ ] 3.6 Nav-Eintrag in `AppShell.tsx` für alle eingeloggten Nutzer

## 4. Stufe 2 — Team-Zugehörigkeit

- [ ] 4.1 DB-Query: prüft ob `user_id` in `team_memberships` oder `team_trainers` für `team_id` aktiv ist
- [ ] 4.2 `GET /api/videos` filtert `visibility = team` auf Mitglieder des jeweiligen Teams
- [ ] 4.3 Frontend: Team-Filter in der Videoliste anzeigen

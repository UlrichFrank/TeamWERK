## 1. Datenbank

- [x] 1.1 Migration `012_game_responses.up.sql` anlegen: Tabelle `game_responses` (game_id FK, member_id FK, responded_by FK, status CHECK('confirmed','declined','maybe'), reason TEXT DEFAULT '', responded_at TEXT, PRIMARY KEY(game_id, member_id))
- [x] 1.2 Migration `012_game_responses.down.sql` anlegen: `DROP TABLE IF EXISTS game_responses`
- [x] 1.3 Migration lokal ausführen und Tabelle verifizieren (`make migrate-up`)

## 2. Backend — Spiel-RSVP-Endpunkte

- [x] 2.1 `GET /api/games/my` implementieren: user-aware Spielliste gefiltert auf eigene Teams (via `game_teams` + `team_memberships`), inkl. `my_rsvp`, `confirmed_count`, `declined_count`, `maybe_count` — analog zu `ListSessions` in `internal/trainings/handler.go`
- [x] 2.2 `POST /api/games/{id}/respond` implementieren: UPSERT in `game_responses`, Rollen-Logik (spieler → eigene member_id, elternteil → family_links prüfen, trainer/admin → beliebige member_id) — analog zu `Respond` in `internal/trainings/handler.go`
- [x] 2.3 `GET /api/games/{id}/responses` implementieren: Rückmeldungs-Liste für Trainer/Admin (member_name, status, reason), Grund-Sichtbarkeit nach Rolle einschränken (nur eigene oder Kinder für elternteil)
- [x] 2.4 Neue Routen in `cmd/teamwerk/main.go` registrieren: `/api/games/my` VOR `/api/games/{id}` eintragen (Chi-Reihenfolge), `/api/games/{id}/respond` und `/api/games/{id}/responses` hinzufügen

## 3. Frontend — TerminePage (/termine)

- [x] 3.1 `web/src/pages/TerminePage.tsx` erstellen: zwei parallele API-Calls (`/api/training-sessions?from=&to=` und `/api/games/my?from=&to=`), Ergebnisse zusammenführen und chronologisch sortieren
- [x] 3.2 Termin-Karten rendern: Training-Karte (Dumbbell-Icon, Zeit, Ort, RSVP-Buttons für Spieler) und Spiel-Karte (Home/MapPin-Icon, Gegner, Zeit, RSVP-Buttons für Spieler) — kein Button ist vorausgewählt
- [x] 3.3 RSVP-Interaktion für Trainings in TerminePage: `POST /api/training-sessions/{id}/respond` wie bisher in `TrainingsPage.tsx`
- [x] 3.4 RSVP-Interaktion für Spiele in TerminePage: `POST /api/games/{id}/respond` mit identischer UX wie Training-RSVP
- [x] 3.5 Trainer-Ansicht: keine RSVP-Buttons, Karte klickbar → navigate zu `/termine/training/:id` bzw. `/termine/spiel/:id`
- [x] 3.6 Abgesagte Trainings: Badge „Abgesagt", keine RSVP-Buttons, opacity-60
- [x] 3.7 `useLiveUpdates` einbinden: bei `'trainings'`- oder `'games'`-Event neu laden

## 4. Frontend — TermineDetailPage (/termine/:type/:id)

- [x] 4.1 `web/src/pages/TermineDetailPage.tsx` erstellen mit `useParams` für `type` (training|spiel) und `id`
- [x] 4.2 Training-Detailansicht: Daten von `/api/training-sessions/:id` laden, Rückmeldungs-Tabelle (Name, Status, Grund) rendern — analog zu `TrainingsDetailPage.tsx`
- [x] 4.3 Spiel-Detailansicht: Daten von `/api/games/:id` (Header-Infos) + `/api/games/:id/responses` (Rückmeldungen) laden, Rückmeldungs-Tabelle rendern
- [x] 4.4 Anwesenheits-Tracking nur für Training-Typ: Checkboxen und `POST /api/training-sessions/:id/attendances` nur wenn `type === 'training'` und Termin in der Vergangenheit
- [x] 4.5 Zurück-Link auf `/termine`, `useLiveUpdates` einbinden

## 5. Routing & Navigation

- [x] 5.1 In `web/src/App.tsx`: Route `/termine` mit `TerminePage`, Routen `/termine/training/:id` und `/termine/spiel/:id` mit `TermineDetailPage` hinzufügen
- [x] 5.2 In `web/src/App.tsx`: `<Navigate from="/trainings" to="/termine" />` und `<Navigate from="/trainings/:id" to="/termine/training/:id" />` einrichten
- [x] 5.3 In `web/src/components/AppShell.tsx`: Nav-Eintrag „Trainings" → „Termine" umbenennen, Pfad auf `/termine` ändern

## 6. Aufräumen

- [x] 6.1 `web/src/pages/TrainingsPage.tsx` und `web/src/pages/TrainingsDetailPage.tsx` löschen (nach Verifikation der Redirects)
- [x] 6.2 Imports der gelöschten Seiten aus `App.tsx` entfernen
- [x] 6.3 Manueller End-to-End-Test: RSVP für Training abgeben, RSVP für Spiel abgeben, Trainer-Übersicht für beide Typen prüfen, Redirect von `/trainings` auf `/termine` prüfen

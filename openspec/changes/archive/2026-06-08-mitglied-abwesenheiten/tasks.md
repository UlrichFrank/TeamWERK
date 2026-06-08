## 1. Datenbank-Migration

- [x] 1.1 Migration `030_member_absences.up.sql` anlegen: Tabelle `member_absences` (id, member_id FK, type CHECK, start_date, end_date, note, created_by FK, created_at)
- [x] 1.2 In Migration 030: `ALTER TABLE members ADD COLUMN absences_public INTEGER NOT NULL DEFAULT 0`
- [x] 1.3 In Migration 030: `ALTER TABLE training_responses ADD COLUMN absence_id INTEGER REFERENCES member_absences(id) ON DELETE CASCADE`
- [x] 1.4 In Migration 030: `ALTER TABLE game_responses ADD COLUMN absence_id INTEGER REFERENCES member_absences(id) ON DELETE CASCADE`
- [x] 1.5 `030_member_absences.down.sql` anlegen (DROP TABLE + ALTER TABLE … DROP COLUMN für alle Änderungen)

## 2. Backend — Package `internal/absences/`

- [x] 2.1 `internal/absences/handler.go` anlegen: `Handler struct{ db, hub }`, `NewHandler`
- [x] 2.2 `GET /api/absences/preview` implementieren: gibt Events mit bestehender `confirmed`-Response im angefragten Zeitraum zurück
- [x] 2.3 `POST /api/absences` implementieren: Abwesenheit anlegen, Berechtigung prüfen (eigener Member oder family_links-Kind), auto-decline aller Training/Spiel-Events im Zeitraum (INSERT OR REPLACE mit absence_id)
- [x] 2.4 `DELETE /api/absences/{id}` implementieren: nur eigene Abwesenheit oder Admin, CASCADE löscht Responses automatisch
- [x] 2.5 `GET /api/absences` implementieren: eigene + Kinder-Abwesenheiten zurückgeben
- [x] 2.6 `GET /api/absences/calendar` implementieren: Abwesenheiten für Datumsbereich; eigene/Kinder immer, andere Members nur wenn `absences_public = 1`

## 3. Backend — Profil-Endpunkt

- [x] 3.1 `PUT /api/profile/absence-visibility` in `internal/members/handler.go` implementieren: setzt `members.absences_public` für den eigenen Member

## 4. Backend — Auto-decline bei neuen Events

- [x] 4.1 In `internal/trainings/handler.go` (CreateTrainingSession): nach dem INSERT alle Kader-Members prüfen, die eine Abwesenheit für das neue Datum haben → auto-declined Response anlegen
- [x] 4.2 In `internal/games/handler.go` (CreateGame): nach dem INSERT analog Abwesenheiten prüfen → auto-declined Responses anlegen

## 5. Backend — Response-Sperre

- [x] 5.1 In `internal/trainings/handler.go` (RSVP-Endpoint): vor dem Schreiben prüfen ob `absence_id IS NOT NULL` → HTTP 403 zurückgeben
- [x] 5.2 In `internal/games/handler.go` (RSVP-Endpoint): analog → HTTP 403 wenn `absence_id IS NOT NULL`

## 6. Backend — Routing

- [x] 6.1 Routen in `cmd/teamwerk/main.go` registrieren: alle `/api/absences`-Endpunkte unter `auth.Middleware`, `PUT /api/profile/absence-visibility` ebenfalls

## 7. Frontend — Abwesenheiten-Seite

- [x] 7.1 `web/src/pages/AbsenzenPage.tsx` anlegen: Liste eigener Abwesenheiten (und Kinder), Neu-anlegen-Formular (Typ-Auswahl, Datumsbereich, Notiz)
- [x] 7.2 Confirmation-Modal einbauen: zeigt Preview-Ergebnisse vor dem Speichern; bei leerer Liste direkt speichern
- [x] 7.3 Route in `App.tsx` registrieren (`/abwesenheiten`), Nav-Eintrag in `AppShell.tsx` für Rollen `spieler` und `elternteil`

## 8. Frontend — Profil-Toggle

- [x] 8.1 In `ProfilPage.tsx` (oder äquivalenter Profil-Seite) Toggle „Abwesenheiten für Trainer sichtbar" hinzufügen, der `PUT /api/profile/absence-visibility` aufruft

## 9. Frontend — Kalender-Banner

- [x] 9.1 `GET /api/absences/calendar` in `KalenderPage.tsx` integrieren (zusammen mit den anderen Event-Fetches)
- [x] 9.2 Abwesenheits-Banner rendern: blassgelbe Linie mit gelbem Rahmen über betroffene Wochentage, auf Höhe der Tageszahl, Wochensegmente bei Wochengrenze
- [x] 9.3 Sichtbarkeitslogik: Banner nur anzeigen wenn `claims.role === 'spieler'/'elternteil'` (eigene/Kind) oder Trainer + `absences_public`
- [x] 9.4 Events im Abwesenheitszeitraum: declined-Badge zeigen, kein RSVP-Button wenn `absence_id` gesetzt (Tooltip „Durch Abwesenheit gesetzt")

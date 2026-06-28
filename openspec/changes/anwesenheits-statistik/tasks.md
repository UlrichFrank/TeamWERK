## 1. Datenmodell & Migration

- [x] 1.1 Migration `internal/db/migrations/012_game_attendances.up.sql` + `.down.sql` anlegen (Tabelle + Index, analog `training_attendances`, siehe design.md D5)
- [x] 1.2 `make migrate-up` lokal ausführen und Schema verifizieren

## 2. Backend — Spiel-Anwesenheits-Routen (`internal/games/`)

- [x] 2.1 Handler-Methode `PostGameAttendances` in `internal/games/handler.go`: Bulk-Upsert auf `game_attendances`, nur vergangene Spiele (422 sonst), Authz Trainer eigenes Team / sportliche_leitung / admin (403 sonst), 404 für nicht existierende Spiele
- [x] 2.2 Handler-Methode `GetGameAttendances`: Liste aller Kader-Mitglieder (Stamm + erweitert dedupliziert, mit `is_extended`), inkl. `rsvp_status`, `reason`, `present` (nullable)
- [x] 2.3 Beide Routen in `internal/app/router.go` unter dem passenden Auth-Tier eintragen
- [x] 2.4 `PostGameAttendances` ruft `h.hub.Broadcast("attendance-changed")` auf
- [x] 2.5 Tests in `internal/games/`: Happy-Path POST (200), zukünftiges Spiel (422), fremdes Team als Trainer (403), unauthentifiziert (401), nicht existierendes Spiel (404), GET happy path inkl. is_extended + nullable present, GET unauthorisiert (403)

## 3. Backend — Aggregations-Package (`internal/attendance/`)

- [x] 3.1 Neues Package `internal/attendance/` mit Handler-Struct `Handler{ db *sql.DB; hub *hub.EventHub }` und Konstruktor `NewHandler`
- [x] 3.2 Klassifikations-Funktion `Classify(present *bool, declined bool, absenceID *int64) Category` (Reihenfolge: ANWESEND → FEHLT → ENTSCHULDIGT → IGNORIERT) plus Unit-Tests inkl. der Edge-Cases aus `attendance-statistics`
- [x] 3.3 Route `GET /api/teams/{id}/attendance-stats?season=<id>`: Zähler je Mitglied, Blöcke `regular_members` / `extended_members`, Team-Durchschnitte, Default = aktive Saison, Authz Trainer/SL/Admin
- [x] 3.4 Route `GET /api/members/{id}/attendance-stats?season=<id>`: Zähler + vollständige Termin-Liste mit `category` und `reason`, Authz eigener/Eltern/Trainer-SL/Admin
- [ ] 3.5 Route `GET /api/teams/{id}/attendance-open`: Liste vergangener, nicht-cancelled Termine ohne `attendance`-Row, Authz Trainer/SL/Admin
- [ ] 3.6 Cancelled Sessions/Games konsequent aus Aggregation entfernen (`status != 'cancelled'`-Filter)
- [ ] 3.7 Routen in `internal/app/router.go` registrieren (Auth-Tiers gemäß design.md D7)
- [x] 3.8 Architektur-Test `internal/arch/arch_test.go` um das neue Package erweitern (Composition-Layer: darf trainings/games/members/kader/absences lesen)
- [ ] 3.9 Tests: Aggregation (drei Säulen korrekt, Stamm vs. erweitert, cancelled ignoriert, Saisonbezug), Authz (alle Rollen, inkl. Eltern via family_links), 401/403/404-Pfade je Endpoint

## 4. Backend — Reminder-Scheduler (`internal/scheduler/`)

- [ ] 4.1 Neue Funktion `RunAttendanceReminders(ctx, db, push, cfg)` registrieren, Trigger 1×/Tag (19:00 lokal) im bestehenden Scheduler-Loop
- [ ] 4.2 Aggregations-Query: für jeden Trainer (`kader_trainers` ↔ `members` ↔ `users`) der aktiven Saison alle offenen Termine seiner Teams sammeln (vergangene, nicht cancelled, ohne `attendance`-Row)
- [ ] 4.3 Idempotenz via `notification_log` (`kind='attendance-reminder'`, `context=YYYY-MM-DD`) per `INSERT OR IGNORE` — nur bei neuem Eintrag senden
- [ ] 4.4 Push-Body bauen: Anzahl + erste 3 Termine im Format `"<Teamname> <Wochentag DD.MM.> (Training|Spiel)"`, Hinweis-Suffix "… und N weitere" wenn >3
- [ ] 4.5 Versand nicht-blockierend via `go push.SendToUsers(...)`, Tap-Ziel `/team/{firstOpenTeamId}/anwesenheit`
- [ ] 4.6 Saison-Cut-off: keine Push wenn keine aktive Saison existiert
- [ ] 4.7 Tests: mehrfacher Job-Lauf an einem Tag → max 1 Push/Trainer; Trainer ohne offene Termine erhält nichts; Saisonende verhindert Push; Body-Format (2 / 3 / 5 Termine); Stop-Bedingung greift nach erster `attendance`-Row

## 5. Frontend — Trainer-Anwesenheits-Seite

- [ ] 5.1 Neue Seite `web/src/pages/TeamAnwesenheitPage.tsx`, Route `/team/:id/anwesenheit` in `App.tsx` registrieren
- [ ] 5.2 Banner oben mit Anzahl offener Erfassungen aus `GET /api/teams/{id}/attendance-open`, Klick öffnet Detail-Liste mit Links zu Einzelterminen
- [ ] 5.3 Tabelle Stammkader: Spalten Spieler | Trainings (✓/⊘/✗/Quote%) | Spiele (✓/⊘/✗/Quote%); Team-Durchschnittszeile am Fuß
- [ ] 5.4 Sub-Tabelle "Erweiterter Kader (N)" mit gleichem Layout und eigener Durchschnittszeile (nur wenn Mitglieder vorhanden)
- [ ] 5.5 Mobile-Card-Layout (siehe `docs/agent/05-frontend.md`), brand-Tokens, `lucide-react`-Icons (`Check`, `MinusCircle`, `X`), Touch-Targets ≥ 44px
- [ ] 5.6 Klick auf Spielerzeile öffnet Member-Detail (interne Navigation zur Spieler-Sicht mit `member_id`-Parameter)
- [ ] 5.7 `useLiveUpdates((event) => { if (event === 'attendance-changed') reload() })` integrieren
- [ ] 5.8 Nav-Eintrag in `AppShell.tsx` für Trainer und sportliche Leitung

## 6. Frontend — Spieler-/Eltern-Sicht

- [ ] 6.1 Neue Seite oder Tab `ProfilAnwesenheitPage.tsx` (Routen-/Tab-Entscheidung im Task umsetzen), Route `/profil/anwesenheit` oder als Tab in bestehender Profil-Komponente
- [ ] 6.2 Kind-Auswahl bei Eltern mit mehreren verlinkten Kindern (Default: erstes Kind)
- [ ] 6.3 Drei-Säulen-Anzeige (Stacked-Bar grün/gelb/rot) plus Quote für Trainings und Spiele getrennt
- [ ] 6.4 Tabellarische Listen "Alle Trainings" und "Alle Spiele" mit Datum, Titel, Status-Badge (anwesend/entschuldigt/fehlt/—), Begründung; Cancelled Termine als grauer Badge
- [ ] 6.5 brand-Tokens, lucide-Icons, Mobile-Card-Layout

## 7. Frontend — Spiel-Detailseite

- [ ] 7.1 Bestehende Spiel-Detailseite (`/termine/spiel/:id`) um Sektion "Anwesenheit erfassen" für Trainer/SL/Admin erweitern (analog Training)
- [ ] 7.2 Bulk-Form mit Checkbox pro Mitglied (Stamm + erweitert klar getrennt), Speichern ruft `POST /api/games/{id}/attendances` auf
- [ ] 7.3 Erfassungssektion nur sichtbar wenn Spiel-Datum ≤ heute; sonst Hinweistext

## 8. Verifikation & Abschluss

- [ ] 8.1 `make test` (inkl. Architektur-Test) grün
- [ ] 8.2 `make lint` grün
- [ ] 8.3 `pnpm -C web build` + `pnpm -C web test` + `pnpm -C web lint` grün
- [ ] 8.4 `openspec validate anwesenheits-statistik --strict` grün
- [ ] 8.5 `/verify-change` ausführen — Build/Test/Lint + Projekt-Invarianten (Route→Tests, Mutation→Broadcast+useLiveUpdates, brand-Tokens, lucide-Icons, Migrationsnummer)
- [ ] 8.6 Manueller Smoke-Test im lokalen Stack: Spiel-Anwesenheit erfassen → Live-Update → Team-Stats + Member-Stats korrekt → Banner verschwindet

# Tasks: Spielplan Event Wizard

## 1. Datenbank-Migration

- [ ] 1.1 Nächste freie Migrations-Nummer ermitteln (nach 023)
- [ ] 1.2 Migration `024_game_teams.up.sql` anlegen: `event_type TEXT NOT NULL DEFAULT 'heim'` zu `games` hinzufügen via Tabellen-Rebuild (CHECK-Constraint)
- [ ] 1.3 In selber Migration: `game_teams (game_id, team_id PK)` Tabelle anlegen
- [ ] 1.4 In selber Migration: bestehende `games.team_id`-Daten nach `game_teams` migrieren (`INSERT INTO game_teams SELECT id, team_id FROM games`)
- [ ] 1.5 In selber Migration: `games` ohne `team_id` neu aufbauen (Tabellen-Rebuild)
- [ ] 1.6 `.down.sql` anlegen: `game_teams` droppen, `games` mit `team_id` (NOT NULL, DEFAULT 0) zurückbauen, `event_type` entfernen
- [ ] 1.7 Migration lokal testen: `make migrate-up` + `make migrate-down`

## 2. Backend — Queries auf `game_teams` umstellen

- [ ] 2.1 In `internal/games/handler.go`: `ListGames` — `games.team_id` durch `JOIN game_teams` ersetzen, `teams: [{id,name}]` pro Game zurückgeben
- [ ] 2.2 `GetGame` — ebenso auf `game_teams` JOIN umstellen
- [ ] 2.3 `UpdateGame` — `team_id`-Feld entfernen, `team_ids []int` akzeptieren, `game_teams` aktualisieren (DELETE + INSERT)
- [ ] 2.4 `DeleteGame` — CASCADE auf `game_teams` prüfen (sollte automatisch via FK)
- [ ] 2.5 `RegenerateSlots` — Team-Liste aus `game_teams` lesen statt `games.team_id`
- [ ] 2.6 `loadSameDayContext` und andere Hilfsfunktionen auf neues Schema prüfen und anpassen

## 3. Backend — `CreateGame` erweitern

- [ ] 3.1 Request-Struct: `TeamID int` → `TeamIDs []int`, `EventType string` (heim/auswärts/generisch), `TemplateID *int` (optional) hinzufügen
- [ ] 3.2 Validierung: `team_ids` nicht leer → HTTP 400; `event_type` gültig → HTTP 400
- [ ] 3.3 Trainer-Check: falls Caller `role=trainer` UND `event_type != 'generisch'` → prüfen ob alle `team_ids` in `team_trainers WHERE user_id=caller` → HTTP 403 sonst
- [ ] 3.4 Transaktion: Game anlegen, `game_teams`-Einträge anlegen, pro Team Slots generieren (aus `template_id` oder `findTemplateForGame`)
- [ ] 3.5 `eventName` aus `event_type` ableiten: `heim` → „Heimspiel vs. [Gegner]", `auswärts` → „Auswärtsspiel vs. [Gegner]", `generisch` → Freitext aus `opponent`-Feld (Umbenennung im Backend: `opponent` bleibt als Feld, wird für Eventname genutzt)

## 4. Backend — Berechtigungen anpassen

- [ ] 4.1 In `cmd/teamwerk/main.go`: `GET /api/games` und `GET /api/games/{id}` aus der `RequireRole("admin","trainer")`-Gruppe herausnehmen und in die allgemeine `auth.Middleware`-Gruppe verschieben
- [ ] 4.2 `POST /api/admin/games`, `PUT /api/admin/games/{id}`, `DELETE /api/admin/games/{id}`, `POST /api/admin/games/{id}/regenerate` in neue Gruppe `RequireRole("admin","vorstand","trainer")` verschieben

## 5. Frontend — AppShell & Routing

- [ ] 5.1 In `AppShell.tsx`: Nav-Eintrag „Spielplan" — `roles`-Array entfernen (für alle authed sichtbar)
- [ ] 5.2 In `SpielplanPage.tsx`: Button „Heimspiel anlegen" → „Event anlegen"; Sichtbarkeit auf `admin`, `vorstand`, `trainer` ausweiten

## 6. Frontend — Wizard Schritt 1: Typ wählen

- [ ] 6.1 State `wizardStep` (1-4) und `eventType` ('heim'|'auswärts'|'generisch') hinzufügen
- [ ] 6.2 Schritt-1-UI: drei Kacheln/Buttons für Heimspiel, Auswärtsspiel, Sonstiges Event
- [ ] 6.3 Bei Klick auf Typ: `eventType` setzen, `wizardStep` auf 2

## 7. Frontend — Wizard Schritt 2: Details

- [ ] 7.1 Felder: Datum, Uhrzeit (immer); Gegner (heim/auswärts) ODER Eventname (generisch)
- [ ] 7.2 Mannschafts-Auswahl: Single-Select für heim/auswärts, Multi-Select (Checkboxen) für generisch
- [ ] 7.3 Beim Laden: für `role=trainer` + heim/auswärts nur eigene Teams per `GET /api/teams` (gefiltert); für generisch oder admin/vorstand alle Teams
- [ ] 7.4 Validierung: Datum + Mannschaft(en) Pflichtfelder; „Weiter"-Button disabled solange unvollständig

## 8. Frontend — Wizard Schritt 3: Vorlage wählen

- [ ] 8.1 Beim Eintreten in Schritt 3: `GET /api/admin/duty-templates` laden, nach `template_type == eventType` filtern
- [ ] 8.2 Gefilterte Vorlagen als Radio-Buttons oder Kacheln anzeigen (Name + Typ)
- [ ] 8.3 Falls keine Vorlage verfügbar: Hinweis „Keine passende Vorlage — Event wird ohne Dienste angelegt" + Weiter-Button
- [ ] 8.4 Bei Vorlagen-Auswahl: `GET /api/admin/duty-templates/{id}/preview?time=...` aufrufen (Bug-Fix: alter Endpunkt `/api/admin/game-template/preview` entfernen)

## 9. Frontend — Wizard Schritt 4: Dienste bestätigen

- [ ] 9.1 Preview-Slots mit Checkboxen anzeigen (alle initial angehakt)
- [ ] 9.2 „Ohne Dienste"- und „Bestätigen"-Buttons wie bisher
- [ ] 9.3 `POST /api/admin/games` mit `team_ids`, `event_type`, `template_id`, `slots[]` abschicken
- [ ] 9.4 Erfolg: Dialog schließen, Spielplan neu laden

## 10. Frontend — Game-Interface anpassen

- [ ] 10.1 `Game`-Interface: `team_id: number` → `teams: {id: number, name: string}[]` und `event_type: string` hinzufügen
- [ ] 10.2 Kalender-Darstellung: bei Multi-Team-Event alle Team-Namen anzeigen (z.B. kommagetrennt oder erste N + „…")

## 11. Verifikation

- [ ] 11.1 Heimspiel als Trainer anlegen → nur eigene Mannschaft wählbar, Heim-Vorlage erscheint
- [ ] 11.2 Auswärtsspiel als Admin anlegen → alle Mannschaften wählbar, Auswärts-Vorlage erscheint
- [ ] 11.3 Generisches Event mit 3 Mannschaften anlegen → 3×Slots generiert, alle 3 Teams in `game_teams`
- [ ] 11.4 Trainer versucht Heimspiel für fremde Mannschaft → HTTP 403
- [ ] 11.5 Spieler öffnet Spielplan → Seite sichtbar, kein „Event anlegen"-Button
- [ ] 11.6 Alter Preview-Endpunkt `/api/admin/game-template/preview` → SPA-Fallback (kein API-Handler mehr)
- [ ] 11.7 `make migrate-down` + `make migrate-up` ohne Fehler

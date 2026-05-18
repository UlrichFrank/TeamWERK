## 1. Datenbank-Migration

- [ ] 1.1 Migration `007_games.up.sql` anlegen: Tabellen `game_templates`, `game_template_items`, `games`
- [ ] 1.2 Migration `007_games.up.sql`: `duty_slots`-Tabelle via Rebuild-Pattern um `game_id` (nullable FK → games) erweitern
- [ ] 1.3 Migration `007_games.down.sql` anlegen (Rollback)
- [ ] 1.4 Migration in `internal/db/migrations/` kopieren und `make migrate-up` lokal ausführen

## 2. Backend — Package `internal/games`

- [ ] 2.1 `internal/games/handler.go` anlegen: `type Handler struct{ db *sql.DB }` + `NewHandler`
- [ ] 2.2 `GET /api/games` implementieren: Spielliste mit slot_count/filled_count per JOIN (admin + trainer)
- [ ] 2.3 `GET /api/games/{id}` implementieren: Spieldetail mit verknüpften Duty Slots
- [ ] 2.4 `POST /api/admin/games` implementieren: Spiel anlegen + Slots aus übergebenem `slots`-Array generieren (nicht aus Template direkt — Vorschau kommt vom Client)
- [ ] 2.7 `GET /api/admin/game-template/preview?date=&time=&team_id=` implementieren: gibt berechnete Slot-Vorschau zurück (Uhrzeiten aus Anker+Offset berechnet)
- [ ] 2.5 `PUT /api/admin/games/{id}` implementieren: Spiel bearbeiten (Datum, Uhrzeit, Gegner)
- [ ] 2.6 `DELETE /api/admin/games/{id}` implementieren: Spiel löschen (Slots bleiben via ON DELETE SET NULL)

## 3. Backend — Template-API

- [ ] 3.1 `GET /api/admin/game-template` implementieren: aktives Template mit allen Items abrufen
- [ ] 3.2 `PUT /api/admin/game-template` implementieren: Template-Items ersetzen (DELETE + INSERT in Transaktion)
- [ ] 3.3 FK-Validierung für `duty_type_id` bei Template-Speicherung

## 4. Router & Integration

- [ ] 4.1 `games.Handler` in `cmd/teamwerk/main.go` instanziieren und übergeben
- [ ] 4.2 Routen für `GET /api/games`, `GET /api/games/{id}` unter `auth.RequireRole("admin","trainer")` registrieren
- [ ] 4.3 Routen für `POST/PUT/DELETE /api/admin/games` und `GET/PUT /api/admin/game-template` unter `auth.RequireRole("admin")` registrieren
- [ ] 4.4 `go build ./...` — kein Compilerfehler

## 5. Frontend — Spielplan-Kalender

- [ ] 5.1 `web/src/pages/SpielplanPage.tsx` anlegen: Monatsansicht mit Spieltagen
- [ ] 5.2 Kalender-Grid-Komponente: 7-Spalten-Grid, Tage des Monats, Spiele als Kacheln
- [ ] 5.3 Besetzungsampel-Logik: grün/gelb/rot basierend auf `filled_count / slot_count`
- [ ] 5.4 Monat-Navigation (Vorheriger / Nächster Monat) mit State
- [ ] 5.5 Route `/spielplan` in `App.tsx` unter `AppShell` registrieren
- [ ] 5.6 Nav-Eintrag „Spielplan" in `AppShell.tsx` für admin + trainer ergänzen

## 6. Backend — Neugenerierung (Overwrite)

- [ ] 6.1 `POST /api/admin/games/{id}/regenerate` implementieren: löscht Slots mit `slots_filled = 0`, legt neue an (atomar); gibt Anzahl beibehaltener belegter Slots zurück
- [ ] 6.2 Warnung im Response wenn belegte Slots nicht überschrieben wurden (`kept_slots: N`)

## 7. Frontend — Spieltag-Detail

- [ ] 7.1 `web/src/pages/SpieltagDetailPage.tsx` anlegen: Spiel-Metadaten + Slot-Zeitleiste
- [ ] 7.2 Slot-Karte mit Diensttyp, Uhrzeit, Rollenbezeichnung und Fortschrittsbalken
- [ ] 7.3 „+ Dienst hinzufügen"-Button: Formular mit Diensttyp, Uhrzeit, Personenanzahl; `game_id` vorbelegt
- [ ] 7.4 „Dienste neu generieren"-Button: öffnet Vorschau-Dialog (Overwrite-Flow)
- [ ] 7.5 Overwrite-Vorschau: zeigt neue Slots + Warnung falls belegte Slots erhalten bleiben
- [ ] 7.6 Einzelnen Slot bearbeiten (Inline-Edit oder Modal): Uhrzeit, Personenanzahl, Rollenbezeichnung
- [ ] 7.7 Einzelnen Slot löschen mit Bestätigung
- [ ] 7.8 Route `/spielplan/:gameId` in `App.tsx` registrieren

## 8. Frontend — Template-Konfiguration (Admin)

- [ ] 8.1 Template-Formular in `AdminSettingsPage.tsx` oder eigene Seite integrieren
- [ ] 8.2 Template-Items: Liste mit Diensttyp-Dropdown, Anker (start/end), Offset, Personenanzahl, Rollenbezeichnung
- [ ] 8.3 Items hinzufügen / entfernen + Speichern via `PUT /api/admin/game-template`

## 9. Frontend — Spiel anlegen mit Vorschau (Admin)

- [ ] 9.1 „Spiel anlegen"-Dialog in `SpielplanPage.tsx`: Datum, Uhrzeit, Gegner, Mannschaft
- [ ] 9.2 Nach Formulareingabe: `GET /api/admin/game-template/preview?...` aufrufen und Vorschauliste anzeigen
- [ ] 9.3 Einzelne Vorschau-Slots entfernbar (Checkbox oder ×-Button pro Item)
- [ ] 9.4 Option „Ohne Dienste anlegen" (leert alle Vorschau-Items)
- [ ] 9.5 Bestätigen → `POST /api/admin/games` mit ausgewählten Slots → Kalender neu laden

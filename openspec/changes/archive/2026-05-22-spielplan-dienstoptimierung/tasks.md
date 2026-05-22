## 1. Datenbank-Migration

- [x] 1.1 Migration `004_games_template_id.up.sql` anlegen: `ALTER TABLE games ADD COLUMN template_id INTEGER REFERENCES game_templates(id);`
- [x] 1.2 Migration `004_games_template_id.down.sql` anlegen: `ALTER TABLE games DROP COLUMN template_id;` (oder Tabellen-Rebuild wenn SQLite-Version kein DROP COLUMN unterstützt)
- [x] 1.3 Lokal `make migrate-up` ausführen und prüfen, dass Migration erfolgreich durchläuft

## 2. Backend: PreviewSlots — `date`-Parameter

- [x] 2.1 In `PreviewSlots` (`internal/games/handler.go`): neuen Query-Parameter `date` auslesen
- [x] 2.2 Wenn `date` gegeben und `game_id` leer: `loadSameDayContext` mit `date` + aktiver Season aufrufen, dann `gameTime` (den `time`-Parameter) sortiert in `allGameTimes` einfügen
- [x] 2.3 Bestehenden `game_id`-Pfad unverändert lassen (Vorrang wenn beide angegeben)
- [x] 2.4 `applyBehavior`-Aufruf auch für den neuen `date`-Pfad aktivieren (analog zum `game_id`-Pfad)

## 3. Backend: CreateGame — `template_id` speichern

- [x] 3.1 In `CreateGame`: SQL-INSERT für `games` um `template_id`-Spalte erweitern
- [x] 3.2 `req.TemplateID` als `sql.NullInt64` oder `*int` in den INSERT übergeben (NULL wenn nicht angegeben)

## 4. Backend: GetGame — `template_id` zurückgeben

- [x] 4.1 In `GetGame`: SELECT-Query um `g.template_id` erweitern
- [x] 4.2 Response-Struct `g` um Feld `TemplateID *int \`json:"template_id"\`` ergänzen
- [x] 4.3 `Scan` um `template_id` erweitern (als `sql.NullInt64`, dann in `*int` konvertieren)

## 5. Backend: RegenerateSlots — template-basiert

- [x] 5.1 Request-Struct in `RegenerateSlots` ändern: `TemplateID *int \`json:"template_id"\`` statt `Slots`-Array
- [x] 5.2 Template-Auflösung: wenn `req.TemplateID` nicht nil → nutzen; sonst `games.template_id` aus DB laden; wenn beides nil → HTTP 400 zurückgeben
- [x] 5.3 `loadTemplateItems` mit aufgelöstem Template aufrufen
- [x] 5.4 `loadSameDayContext` und `applyBehavior` auf Template-Items anwenden (analog zu PreviewSlots)
- [x] 5.5 Slots generieren (bisherige Schleife, aber Template-Items statt `req.Slots` als Quelle)
- [x] 5.6 Nach erfolgreicher Generierung `games.template_id` auf den verwendeten Wert aktualisieren

## 6. Backend: Kompilieren & Smoke-Test

- [x] 6.1 `go build ./...` läuft ohne Fehler durch
- [ ] 6.2 Backend lokal starten und manuell PreviewSlots mit `?date=` testen

## 7. Frontend: SpielplanPage

- [x] 7.1 In `handleFetchPreview`: `date`-Parameter (ISO-Date aus dem Formular) an den PreviewSlots-Endpunkt anhängen
- [x] 7.2 In `doCreateGame`: `template_id` aus dem ausgewählten Template an `POST /admin/games` übergeben

## 8. Frontend: SpieltagDetailPage — Regenerierungs-Dialog

- [x] 8.1 Defekten `GET /admin/game-template/preview`-Aufruf entfernen
- [x] 8.2 `game.template_id` aus `GetGame`-Response lesen und in State speichern
- [x] 8.3 Neuen Dialog-State anlegen: `regenOpen`, `regenTemplateID`, `regenPreview`, `regenLoading`
- [x] 8.4 Dialog-Komponente: Template-Dropdown (alle aktiven Templates, vorausgewählt: `game.template_id`)
- [x] 8.5 Dialog-Komponente: Bei Template-Auswahl `GET /admin/duty-templates/{id}/preview?time=...&game_id=...` aufrufen und Preview anzeigen
- [x] 8.6 Dialog-Komponente: Fehlermeldung wenn kein Template ausgewählt und kein gespeichertes Template
- [x] 8.7 "Anwenden"-Button: `POST /admin/games/{id}/regenerate` mit `{template_id}` aufrufen, Dialog schließen, Spieldetails neu laden
- [x] 8.8 Ladezustände und Fehlerfeedback im Dialog korrekt abbilden

## 9. Abschluss

- [x] 9.1 `pnpm build` läuft ohne Fehler durch
- [ ] 9.2 Manueller E2E-Test: neues Heimspiel anlegen, Dienstvorschau prüft same-day-Kontext
- [ ] 9.3 Manueller E2E-Test: Regenerierungs-Dialog öffnen, Template wählen, Preview sehen, anwenden
- [ ] 9.4 `make deploy` auf VPS ausführen (inkl. Migration 004)

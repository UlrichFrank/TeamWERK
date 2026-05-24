## 1. Datenbank-Migration

- [x] 1.1 Migration `011_age_class_game_rules.up.sql` anlegen: Tabelle `age_class_game_rules (age_class TEXT PRIMARY KEY CHECK(...), half_duration_minutes INTEGER NOT NULL, break_minutes INTEGER NOT NULL)` mit Seed-Daten A(30,15), B(25,10), C(25,10), D(20,10)
- [x] 1.2 In derselben Migration: `ALTER TABLE game_templates RENAME COLUMN game_duration_minutes TO duration_minutes`
- [x] 1.3 Migration `011_age_class_game_rules.down.sql` anlegen: Spalte zurückbenennen, Tabelle droppen
- [x] 1.4 `make migrate-up` lokal ausführen und Seed-Daten prüfen

## 2. Backend — Handler (internal/config)

- [x] 2.1 Struct `AgeClassRule` mit Feldern `AgeClass`, `HalfDurationMinutes`, `BreakMinutes` in `internal/config` definieren
- [x] 2.2 `GetAgeClassRules(db) ([]AgeClassRule, error)` implementieren (alle vier Zeilen, sortiert nach age_class)
- [x] 2.3 `UpdateAgeClassRule(db, ageClass string, half, break int) error` implementieren mit Validierung (Werte > 0, gültige Klasse)
- [x] 2.4 Handler `GetAgeClassRulesHandler` (GET `/api/admin/age-class-rules`) registrieren — zugänglich für admin, vorstand, trainer
- [x] 2.5 Handler `UpdateAgeClassRuleHandler` (PUT `/api/admin/age-class-rules/{ageClass}`) registrieren — nur admin
- [x] 2.6 Routen in `cmd/teamwerk/main.go` eintragen

## 3. Backend — Spaltenumbenennung nachziehen

- [x] 3.1 Alle Referenzen auf `game_duration_minutes` in Go-Code (internal/games, internal/config) auf `duration_minutes` umbenennen
- [x] 3.2 API-Response für `GET /api/admin/game-template` und zugehörige Structs auf `duration_minutes` aktualisieren

## 4. Backend — Slot-Zeitberechnung (internal/games)

- [x] 4.1 Hilfsfunktion `effectiveEventDuration(db, eventType string, templateID, teamID int) (int, error)` implementieren: branch nach `event_type` — heim/auswärts → `age_class_game_rules`, generisch → `duration_minutes` der Vorlage
- [x] 4.2 Slot-Generierungslogik in `internal/games` auf `effectiveEventDuration` umstellen
- [x] 4.3 Fehlerfall heim/auswärts ohne Team-Altersklasse → HTTP 422 mit Fehlermeldung
- [x] 4.4 Fehlerfall generisch ohne `duration_minutes` in Vorlage → HTTP 422 mit Fehlermeldung

## 5. Frontend — Admin-Seite Altersklassen

- [x] 5.1 `web/src/pages/AdminAgeClassRulesPage.tsx` anlegen: Tabelle mit vier Zeilen (A–D), jeweils `half_duration_minutes` und `break_minutes` als Zahlen-Inputs, Speichern-Button pro Zeile
- [x] 5.2 Frontend-Validierung: Werte müssen > 0 sein, sonst Fehlermeldung ohne API-Call
- [x] 5.3 Route `/admin/altersklassen` in `App.tsx` registrieren (nur für admin)
- [x] 5.4 Nav-Eintrag „Altersklassen" in `AppShell.tsx` für Rolle `admin` eintragen (unter Admin-Bereich)

## 6. Frontend — Spielplan-Vorlagen anpassen

- [x] 6.1 In der Vorlagen-UI (`AdminGameTemplatePage` o. Ä.) das Feld `game_duration_minutes` → `duration_minutes` umbenennen und nur für generische Vorlagen anzeigen/aktivieren
- [x] 6.2 Label im UI: „Dauer (Minuten)" statt „Spieldauer"
- [x] 6.3 Seite im Browser testen: Altersklassen laden/bearbeiten, generische Vorlage mit Dauer, Heim-Vorlage ohne Dauer-Feld

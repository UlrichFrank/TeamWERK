## 1. Datenbank-Migration

- [x] 1.1 Nächste freie Migrations-Nummer ermitteln (nach 020)
- [x] 1.2 Migration `.up.sql` anlegen: `ALTER TABLE game_templates ADD COLUMN template_type TEXT NOT NULL DEFAULT 'generisch'`
- [x] 1.3 CHECK-Constraint für `template_type` per Tabellen-Rebuild hinzufügen (`CREATE TABLE ... AS SELECT ... DROP ... RENAME`)
- [x] 1.4 `UPDATE game_templates SET template_type = 'generisch' WHERE is_active = 1` in Migration
- [x] 1.5 `.down.sql` anlegen (Tabelle zurückbauen ohne `template_type`-Spalte)
- [x] 1.6 Migration lokal testen: `make migrate-up` + `make migrate-down`

## 2. Backend — REST-Endpunkte umbenennen

- [x] 2.1 In `internal/games/handler.go`: alle Handler-Methoden für Template-Endpunkte prüfen
- [x] 2.2 In `cmd/teamwerk/main.go`: Route `/api/admin/game-template` entfernen
- [x] 2.3 Neue Routen unter `/api/admin/duty-templates` registrieren (GET Liste, GET/:id, POST, PUT/:id, DELETE/:id)
- [x] 2.4 `GET /api/admin/duty-templates` implementieren: alle Vorlagen mit Items zurückgeben
- [x] 2.5 `POST /api/admin/duty-templates` implementieren: neue Vorlage mit `template_type` anlegen
- [x] 2.6 `PUT /api/admin/duty-templates/{id}` implementieren: Name, `template_type` und Items aktualisieren
- [x] 2.7 `DELETE /api/admin/duty-templates/{id}` implementieren: Vorlage + Items löschen
- [x] 2.8 `template_type`-Validierung: nur `heim`, `auswärts`, `generisch` erlauben → HTTP 400 sonst

## 3. Backend — Slot-Generierung anpassen

- [x] 3.1 `findTemplateForGame(isHome bool)` Hilfsfunktion schreiben: sucht zuerst spezifischen Typ, dann `generisch`
- [x] 3.2 `CreateGame`-Handler: `findTemplateForGame` statt `is_active=1 LIMIT 1`
- [x] 3.3 `RegenerateSlots`-Handler: `findTemplateForGame` statt `is_active=1 LIMIT 1`
- [x] 3.4 `PreviewSlots`-Handler: `findTemplateForGame` statt `is_active=1 LIMIT 1`
- [x] 3.5 Fehlerfall „kein Template gefunden" mit klarer Fehlermeldung behandeln

## 4. Frontend — neue Pages anlegen

- [x] 4.1 `web/src/pages/AdminDutyTemplatesPage.tsx` erstellen: Tabelle mit allen Vorlagen (Name, Typ, Items-Anzahl), Löschen-Button, Link zur Detailseite
- [x] 4.2 `web/src/pages/AdminDutyTemplateDetailPage.tsx` erstellen: Detailseite nach Muster `MemberDetailPage` — Felder: Name, `template_type`-Select, Items-Liste bearbeiten, Speichern
- [x] 4.3 In `web/src/App.tsx`: Route `/admin/spielplan-template` entfernen, neue Routen `/admin/dienstplan-vorlagen` + `/admin/dienstplan-vorlagen/:id` anlegen
- [x] 4.4 In `web/src/components/AppShell.tsx`: Nav-Eintrag von „Spiel-Vorlage" → „Dienstplan-Vorlagen", Pfad aktualisieren
- [x] 4.5 API-Calls in den neuen Pages auf `/admin/duty-templates` umstellen
- [x] 4.6 `AdminGameTemplatePage.tsx` entfernen (nach erfolgreichem Test)

## 5. Frontend — Typ-Auswahl und UX

- [x] 5.1 `template_type`-Select mit Optionen `heim`, `auswärts`, `generisch` auf Detailseite implementieren
- [x] 5.2 Warnung in der Listenansicht anzeigen wenn zwei Vorlagen den gleichen `template_type` haben
- [x] 5.3 Löschen-Aktion mit Bestätigungs-Dialog absichern

## 6. Verifikation

- [ ] 6.1 Heimspiel anlegen → prüfen ob Heim-Vorlage verwendet wird
- [ ] 6.2 Auswärtsspiel anlegen → prüfen ob Auswärts-Vorlage verwendet wird
- [ ] 6.3 Slot-Generierung ohne passende Vorlage → Fehlermeldung prüfen
- [ ] 6.4 Alter API-Pfad `/api/admin/game-template` → HTTP 404 prüfen
- [ ] 6.5 Alle CRUD-Operationen für Vorlagen im Frontend testen

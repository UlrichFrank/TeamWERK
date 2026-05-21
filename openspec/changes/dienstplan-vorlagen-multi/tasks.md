## 1. Datenbank-Migration

- [ ] 1.1 Nächste freie Migrations-Nummer ermitteln (nach 020)
- [ ] 1.2 Migration `.up.sql` anlegen: `ALTER TABLE game_templates ADD COLUMN template_type TEXT NOT NULL DEFAULT 'generisch'`
- [ ] 1.3 CHECK-Constraint für `template_type` per Tabellen-Rebuild hinzufügen (`CREATE TABLE ... AS SELECT ... DROP ... RENAME`)
- [ ] 1.4 `UPDATE game_templates SET template_type = 'generisch' WHERE is_active = 1` in Migration
- [ ] 1.5 `.down.sql` anlegen (Tabelle zurückbauen ohne `template_type`-Spalte)
- [ ] 1.6 Migration lokal testen: `make migrate-up` + `make migrate-down`

## 2. Backend — REST-Endpunkte umbenennen

- [ ] 2.1 In `internal/games/handler.go`: alle Handler-Methoden für Template-Endpunkte prüfen
- [ ] 2.2 In `cmd/teamwerk/main.go`: Route `/api/admin/game-template` entfernen
- [ ] 2.3 Neue Routen unter `/api/admin/duty-templates` registrieren (GET Liste, GET/:id, POST, PUT/:id, DELETE/:id)
- [ ] 2.4 `GET /api/admin/duty-templates` implementieren: alle Vorlagen mit Items zurückgeben
- [ ] 2.5 `POST /api/admin/duty-templates` implementieren: neue Vorlage mit `template_type` anlegen
- [ ] 2.6 `PUT /api/admin/duty-templates/{id}` implementieren: Name, `template_type` und Items aktualisieren
- [ ] 2.7 `DELETE /api/admin/duty-templates/{id}` implementieren: Vorlage + Items löschen
- [ ] 2.8 `template_type`-Validierung: nur `heim`, `auswärts`, `generisch` erlauben → HTTP 400 sonst

## 3. Backend — Slot-Generierung anpassen

- [ ] 3.1 `findTemplateForGame(isHome bool)` Hilfsfunktion schreiben: sucht zuerst spezifischen Typ, dann `generisch`
- [ ] 3.2 `CreateGame`-Handler: `findTemplateForGame` statt `is_active=1 LIMIT 1`
- [ ] 3.3 `RegenerateSlots`-Handler: `findTemplateForGame` statt `is_active=1 LIMIT 1`
- [ ] 3.4 `PreviewSlots`-Handler: `findTemplateForGame` statt `is_active=1 LIMIT 1`
- [ ] 3.5 Fehlerfall „kein Template gefunden" mit klarer Fehlermeldung behandeln

## 4. Frontend — neue Pages anlegen

- [ ] 4.1 `web/src/pages/AdminDutyTemplatesPage.tsx` erstellen: Tabelle mit allen Vorlagen (Name, Typ, Items-Anzahl), Löschen-Button, Link zur Detailseite
- [ ] 4.2 `web/src/pages/AdminDutyTemplateDetailPage.tsx` erstellen: Detailseite nach Muster `MemberDetailPage` — Felder: Name, `template_type`-Select, Items-Liste bearbeiten, Speichern
- [ ] 4.3 In `web/src/App.tsx`: Route `/admin/spielplan-template` entfernen, neue Routen `/admin/dienstplan-vorlagen` + `/admin/dienstplan-vorlagen/:id` anlegen
- [ ] 4.4 In `web/src/components/AppShell.tsx`: Nav-Eintrag von „Spiel-Vorlage" → „Dienstplan-Vorlagen", Pfad aktualisieren
- [ ] 4.5 API-Calls in den neuen Pages auf `/admin/duty-templates` umstellen
- [ ] 4.6 `AdminGameTemplatePage.tsx` entfernen (nach erfolgreichem Test)

## 5. Frontend — Typ-Auswahl und UX

- [ ] 5.1 `template_type`-Select mit Optionen `heim`, `auswärts`, `generisch` auf Detailseite implementieren
- [ ] 5.2 Warnung in der Listenansicht anzeigen wenn zwei Vorlagen den gleichen `template_type` haben
- [ ] 5.3 Löschen-Aktion mit Bestätigungs-Dialog absichern

## 6. Verifikation

- [ ] 6.1 Heimspiel anlegen → prüfen ob Heim-Vorlage verwendet wird
- [ ] 6.2 Auswärtsspiel anlegen → prüfen ob Auswärts-Vorlage verwendet wird
- [ ] 6.3 Slot-Generierung ohne passende Vorlage → Fehlermeldung prüfen
- [ ] 6.4 Alter API-Pfad `/api/admin/game-template` → HTTP 404 prüfen
- [ ] 6.5 Alle CRUD-Operationen für Vorlagen im Frontend testen

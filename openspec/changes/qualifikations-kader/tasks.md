## 1. Datenbank-Migration

- [ ] 1.1 Migration `015_qualifikations_kader.up.sql` anlegen: `ALTER TABLE kader ADD COLUMN type TEXT NOT NULL DEFAULT 'regular' CHECK(type IN ('regular','qualification'))` und `ADD COLUMN is_active INTEGER NOT NULL DEFAULT 1`
- [ ] 1.2 In derselben Migration bestehenden Index `kader_unique` droppen und zwei partielle Ersatz-Indizes erstellen (`kader_unique_active_regular` und `kader_unique_active_quali`)
- [ ] 1.3 Migration `015_qualifikations_kader.down.sql` anlegen (Spalten via Tabellen-Rebuild entfernen, alten Index wiederherstellen)
- [ ] 1.4 Migration lokal ausführen (`make migrate-up`) und prüfen, dass bestehende Kader `type='regular'` und `is_active=1` haben

## 2. Backend — kader-Handler

- [ ] 2.1 `kaderRow`-Struct in `handler.go` um Felder `Type string` und `IsActive bool` erweitern
- [ ] 2.2 `scanKaderRow` und `kaderSelectSQL` aktualisieren: `k.type`, `k.is_active` einlesen
- [ ] 2.3 `ListKader` (`GET /api/admin/kader`): `WHERE k.is_active=1` zur bestehenden Query hinzufügen
- [ ] 2.4 `InitializeKader` / `createSingleKader`: optionales Feld `type` aus Request-Body lesen (Default: `'regular'`), in INSERT schreiben; neuer Kader startet mit `is_active=0` wenn `type='qualification'`, sonst `is_active=1`
- [ ] 2.5 Neuen Handler `ActivateKader` implementieren (`PUT /api/admin/kader/{id}/activate`): in einer Transaktion alle Kader desselben `(season_id, age_class, gender, type)` auf `is_active=0`, dann Ziel-Kader auf `is_active=1`
- [ ] 2.6 Neuen Handler `DeactivateKader` implementieren (`PUT /api/admin/kader/{id}/deactivate`): Kader auf `is_active=0` setzen
- [ ] 2.7 Neue Routen in `main.go` registrieren: `r.Put("/api/admin/kader/{id}/activate", kaderH.ActivateKader)` und `r.Put("/api/admin/kader/{id}/deactivate", kaderH.DeactivateKader)`
- [ ] 2.8 `copy.go` (`CopyFromSeason`) prüfen: stellt sicher, dass kopierte Kader `type='regular'` und `is_active=1` erhalten

## 3. Frontend — Saisons-Tab (AdminSettingsPage)

- [ ] 3.1 In `SaisonsTab` Kader-Daten für die aktive Saison nachladen (`GET /api/admin/kader` nach Saison-Aktivierung oder bei Tab-Mount)
- [ ] 3.2 Pro Kader-Gruppe (Altersklasse + Geschlecht) eine Zeile rendern: Name des aktiven regulären Kaders + optionalen Quali-Kader-Slot
- [ ] 3.3 „Quali-Kader anlegen"-Button pro Gruppe: öffnet Modal mit Feldern Name (Text), Altersklasse (readonly, aus Gruppe), Geschlecht (readonly); POST zu `/api/admin/kader` mit `type='qualification'`
- [ ] 3.4 Nach Anlegen des Quali-Kaders direkt `PUT /api/admin/kader/{id}/activate` aufrufen und Liste neu laden
- [ ] 3.5 Bestehenden Quali-Kader deaktivieren: Button „Quali-Kader beenden" → `PUT /api/admin/kader/{id}/deactivate` + Reload

## 4. Frontend — Kader-Übersicht (AdminKaderPage)

- [ ] 4.1 `kaderRow`-Typ um `type: 'regular' | 'qualification'` und `is_active: boolean` erweitern
- [ ] 4.2 Qualifikationskader visuell kennzeichnen (z.B. Badge „Quali" neben dem Namen)
- [ ] 4.3 Sicherstellen, dass `ListKader`-Query bereits `is_active=1` filtert (durch Backend-Änderung 2.3 bereits erledigt — nur verifizieren)

## 1. Backend — Saison-Edit-Endpoint

- [x] 1.1 In `internal/config/handler.go` (oder passendem Package): `UpdateSeason`-Handler implementieren — parst `name`, `start_date`, `end_date` aus Request-Body, validiert auf Vollständigkeit, führt UPDATE aus, gibt 200 + aktualisierten Datensatz zurück
- [x] 1.2 Route in `main.go` registrieren: `PUT /api/admin/seasons/{id}` in admin+vorstand Middleware-Gruppe
- [x] 1.3 Fehlerfall: 404 wenn Season-ID nicht existiert; 400 bei fehlenden Pflichtfeldern

## 2. Frontend — AdminSettingsPage erstellen

- [x] 2.1 `web/src/pages/AdminSettingsPage.tsx` anlegen
- [x] 2.2 Tab-Leiste mit drei Tabs: „Verein", „Saisons", „Altersklassen"; aktiver Tab via `useSearchParams` (`?tab=verein|saisons|altersklassen`), Default: `verein`
- [x] 2.3 Tab-Wechsel setzt `?tab=`-Param, kein Neuladen der Seite
- [x] 2.4 Jeder Tab lädt seine Daten lazy beim ersten Aktivieren

## 3. Frontend — Tab: Verein

- [x] 3.1 Inhalt und Logik aus `AdminClubPage.tsx` 1:1 übernehmen (GET/PUT `/api/admin/club`, Formular: Vereinsname + Adresse, Save-Button)

## 4. Frontend — Tab: Saisons (Modal-Muster)

- [x] 4.1 Liste bestehender Saisons laden (GET `/api/admin/seasons`)
- [x] 4.2 Tabellen-Header-Zeile mit „Saison anlegen"-Button oben rechts (wie Diensttypen-Page)
- [x] 4.3 Create-Modal: Preset-Dropdown (auto-füllt Name+Datum), Name-Input, Startdatum, Enddatum — `POST /api/admin/seasons`
- [x] 4.4 Jede Saison-Zeile: „Bearbeiten"-Button → Edit-Modal via `EditModal`-Komponente; Felder: Name, Startdatum, Enddatum (vorbefüllt); bei `is_active=true` Info-Hinweis „Aktive Saison" im Modal-Header — `PUT /api/admin/seasons/{id}`
- [x] 4.5 „Aktivieren"-Button in Zeile bleibt erhalten (nur für inaktive Saisons) — `PUT /api/admin/seasons/{id}/activate`
- [x] 4.6 „Löschen"-Button in Zeile bleibt erhalten (nur für inaktive Saisons) — `DELETE /api/admin/seasons/{id}`
- [x] 4.7 Mobile: MobileCard pro Saison mit Bearbeiten/Aktivieren/Löschen-Actions
- [x] 4.8 Saison-Preset-Logik aus `AdminSeasonsPage` übernehmen (`generateSeasonOptions`)
- [x] 4.9 Escape-Key schließt offene Modals (`useEscapeKey`)

## 5. Frontend — Tab: Altersklassen

- [x] 5.1 Inhalt und Logik aus `AdminAgeClassRulesPage.tsx` 1:1 übernehmen (GET/PUT `/api/admin/age-class-rules/{ageClass}`, Tabelle mit inline-Edit pro Zeile)
- [x] 5.2 Outer-Padding (`p-8 sm:p-8 px-4 py-4`) entfernen — die neue Page hat ihren eigenen Layout-Wrapper

## 6. Frontend — Navigation & Routen

- [x] 6.1 `AppShell.tsx`: Im Abschnitt „Kaderplanung" die drei Einträge `{ to: '/admin/verein', ... }`, `{ to: '/admin/altersklassen', ... }`, `{ to: '/admin/saisons', ... }` durch einen einzigen `{ to: '/admin/einstellungen', label: 'Einstellungen', roles: ['admin', 'vorstand'] }` ersetzen
- [x] 6.2 `App.tsx`: Neue Route `path="admin/einstellungen"` mit `<RoleRoute roles={['admin','vorstand']}><AdminSettingsPage /></RoleRoute>`
- [x] 6.3 `App.tsx`: Drei Redirect-Routen hinzufügen:
  - `path="admin/verein"` → `<Navigate to="/admin/einstellungen?tab=verein" replace />`
  - `path="admin/saisons"` → `<Navigate to="/admin/einstellungen?tab=saisons" replace />`
  - `path="admin/altersklassen"` → `<Navigate to="/admin/einstellungen?tab=altersklassen" replace />`
- [x] 6.4 Imports für die drei alten Pages aus `App.tsx` entfernen

## 7. Aufräumen

- [x] 7.1 `web/src/pages/AdminSeasonsPage.tsx` löschen
- [x] 7.2 `web/src/pages/AdminClubPage.tsx` löschen
- [x] 7.3 `web/src/pages/AdminAgeClassRulesPage.tsx` löschen

## 8. Testen

- [ ] 8.1 Manuell: `/admin/verein` → Redirect zu `?tab=verein` ✓
- [ ] 8.2 Manuell: `/admin/saisons` → Redirect zu `?tab=saisons` ✓
- [ ] 8.3 Manuell: `/admin/altersklassen` → Redirect zu `?tab=altersklassen` ✓
- [ ] 8.4 Manuell: Saison anlegen via Modal ✓
- [ ] 8.5 Manuell: Saison bearbeiten via Modal (aktive + inaktive) ✓
- [ ] 8.6 Manuell: Verein-Tab speichern ✓
- [ ] 8.7 Manuell: Altersklassen inline-Edit speichern ✓
- [ ] 8.8 Manuell: Sidebar zeigt nur noch „Einstellungen" statt drei Einträge ✓

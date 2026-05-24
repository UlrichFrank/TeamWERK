## Why

Die Kaderplanung-Sektion in der Sidebar hat aktuell vier Einträge (Verein, Altersklassen, Kader, Saisons), von denen drei reine Konfigurationsseiten ohne Kader-Bezug sind. Das verstreut administrative Grundeinstellungen über mehrere Klicks. Eine konsolidierte „Einstellungen"-Seite fasst Verein, Saisons und Altersklassen zusammen und macht die Sidebar schlanker.

Zusätzlich ist die Saison-Verwaltung als einzige Adminseite noch nicht auf das Diensttypen-Modal-Muster migriert: das Anlegen-Formular steht inline auf der Seite und Saisons lassen sich nach dem Anlegen nicht mehr bearbeiten.

## What Changes

- **Neue Seite** `AdminSettingsPage` unter `/admin/einstellungen` mit drei Tabs: „Verein", „Saisons", „Altersklassen"
- **Saisons-Tab**: Muster wie `AdminDutyTypesPage` — „Saison anlegen"-Button oben rechts öffnet Modal; jede Saison-Zeile hat einen „Bearbeiten"-Button der ein Edit-Modal öffnet
- **Neuer Backend-Endpoint** `PUT /api/admin/seasons/{id}` zum Bearbeiten von Name und Datum einer bestehenden Saison
- **Drei alte Pages entfernt**: `AdminSeasonsPage`, `AdminClubPage`, `AdminAgeClassRulesPage` werden gelöscht
- **Navigation konsolidiert**: „Verein", „Altersklassen", „Saisons" im AppShell durch einen Eintrag „Einstellungen" ersetzt
- **Redirects**: `/admin/verein`, `/admin/saisons`, `/admin/altersklassen` leiten auf `/admin/einstellungen` weiter (React Router)

## Capabilities

### New Capabilities

- `einstellungen-seite`: Einzelne Seite mit Tab-Navigation für Verein-, Saison- und Altersklassen-Verwaltung
- `saison-bearbeiten`: Bestehende Saisons können über einen modalen Dialog bearbeitet werden (Name, Start-/Enddatum)

### Modified Capabilities

- `club-config`: Route ändert sich von `/admin/verein` → `/admin/einstellungen` (Tab „Verein"); Funktionalität bleibt identisch
- `games`: Altersklassen-Regeln (Halbzeit/Pause) sind jetzt unter `/admin/einstellungen` (Tab „Altersklassen") statt `/admin/altersklassen`

## Impact

**Backend:**
- Neuer Endpoint: `PUT /api/admin/seasons/{id}` (body: `name`, `start_date`, `end_date`) — nur admin/vorstand

**Frontend:**
- Neue Datei: `web/src/pages/AdminSettingsPage.tsx`
- Gelöscht: `AdminSeasonsPage.tsx`, `AdminClubPage.tsx`, `AdminAgeClassRulesPage.tsx`
- `App.tsx`: 3 alte Routen entfernen, neue Route + 3 Redirects
- `AppShell.tsx`: 3 Einträge → 1 Eintrag „Einstellungen"

**Keine API-Breaking-Changes** außer dem neuen PUT-Endpoint.

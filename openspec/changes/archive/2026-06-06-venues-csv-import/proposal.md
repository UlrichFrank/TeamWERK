## Why

Veranstaltungsorte müssen bisher manuell einzeln angelegt werden. Der BWHV veröffentlicht eine offizielle Hallenliste (~1000 Hallen) im CSV-Format, die regelmäßig aktualisiert wird. Ein CSV-Import ermöglicht den Erstaufbau und spätere Updates mit einem einzigen Klick.

## What Changes

- Split-Button auf `/admin/veranstaltungsorte`: linke Hälfte "+ Neuer Ort" (unverändert), rechte Hälfte öffnet Dropdown mit "Import CSV"
- Import-Modal mit Datei-Upload und Ergebnis-Anzeige (importiert / aktualisiert / übersprungen / Fehler)
- Neuer Backend-Endpoint `POST /api/admin/venues/import` (multipart/form-data)
- Upsert-Verhalten: neue Hallen werden angelegt, bestehende (gleicher Name) werden aktualisiert — `is_home_venue` wird beim Import nie überschrieben

## Capabilities

### New Capabilities
- `venue-csv-import`: CSV-Datei (BWHV-Hallenliste) hochladen und Veranstaltungsorte per Upsert importieren

### Modified Capabilities

## Impact

- `internal/venues/handler.go`: neue `Import`-Methode
- `cmd/teamwerk/main.go`: neue Route `POST /api/admin/venues/import` unter Admin-only
- `web/src/pages/AdminVenuesPage.tsx`: Split-Button + Import-Modal

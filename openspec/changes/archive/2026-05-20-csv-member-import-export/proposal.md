## Why

Mitgliedsdaten müssen für externe Zwecke (Verbandsmeldungen, Vereinsarchiv, Datensicherung) exportierbar und für Massenänderungen oder Erstbefüllung importierbar sein. Der bestehende CSV-Export ist unvollständig — er fehlt Felder wie Geschlecht, Trikotnummer, Mitgliedsnummer und Verknüpfungen zu Benutzern und Erziehungsberechtigten.

## What Changes

- **Export erweitern**: GET /api/members/export liefert alle 12 Felder eines Mitglieds (inkl. user_email und bis zu 2 Erziehungsberechtigten-Emails via family_links), keine Mannschafts-/Saisondaten
- **Import neu**: POST /api/members/import — idempotenter CSV-Import mit zwei Modi (nur ergänzen / aktualisieren), detailliertem Importbericht und strikter Non-Destructive-Policy (niemals Felder leeren, niemals Links entfernen, niemals Mitglieder löschen)
- **Import-Dialog**: Neues Modal auf /mitglieder (Admin only) mit File-Input, Modus-Auswahl und Berichtsanzeige

## Capabilities

### New Capabilities

- `member-csv-export`: Vollständiger CSV-Export aller Mitglieder mit allen Feldern und Verknüpfungen
- `member-csv-import`: Idempotenter CSV-Import mit Konflikt-Modus-Option und detailliertem Importbericht

### Modified Capabilities

*(keine bestehenden Specs betroffen)*

## Impact

- **Backend**: `internal/members/handler.go` — Export-Handler erweitern, neuer Import-Handler
- **Router**: `cmd/teamwerk/main.go` — neue Route POST /api/members/import (Admin only)
- **Frontend**: `web/src/pages/MembersPage.tsx` — Import-Button + Modal hinzufügen
- **Keine neuen Abhängigkeiten** — encoding/csv bereits importiert, strings/unicode für CSV-Parsing

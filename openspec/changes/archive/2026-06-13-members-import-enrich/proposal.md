## Why

Der CSV-Import unterstützt bisher nur „Anlegen" (nur neue Mitglieder) und „Aktualisieren" (überschreibt vorhandene Felder). Es gibt keinen sicheren Weg, leere Felder bestehender Mitglieder aus einer externen Quelle zu befüllen, ohne Gefahr zu laufen, korrekte Daten zu überschreiben. Dies führt dazu, dass Datenpflege-Importe manuell geprüft oder gar nicht durchgeführt werden.

## What Changes

- Neuer Import-Modus `enrich`: ergänzt nur leere Felder bestehender Mitglieder, legt keine neuen an.
- Matching via Vorname+Nachname+Geburtsdatum; wenn kein Geburtsdatum in der CSV vorhanden ist, Fallback auf Vorname+Nachname mit Mehrdeutigkeitsprüfung.
- Nicht gefundene Zeilen werden als `not_found` im Report ausgewiesen (kein Fehler, kein Anlegen).
- IBAN wird nur ergänzt wenn das DB-Feld leer ist; MOD-97-Validierung läuft weiterhin.
- Neuer Zähler `not_found` im `ImportReport`.
- Frontend: drittes Radio-Button „Nur leere Felder ergänzen", `not_found`-Badge im Summary, graue Zeilen-Darstellung.

## Capabilities

### New Capabilities

- `members-csv-enrich-mode`: Sicherer CSV-Import-Modus der ausschließlich leere Felder bestehender Mitglieder befüllt — ohne Neuanlage und ohne Überschreibung vorhandener Werte.

### Modified Capabilities

## Impact

- `internal/members/handler.go`: neuer `mode == "enrich"` Branch, Matching-Logik-Erweiterung, `ImportRow.Status` um `not_found` erweitert, `ImportReport` um `NotFound int` ergänzt.
- `web/src/pages/MembersPage.tsx`: drittes Radio-Button, `not_found`-Rendering (Icon, Farbe), neuer Badge im Summary.
- Keine neuen Migrations, keine neuen Abhängigkeiten, keine API-Breaking-Changes (additive Erweiterung).

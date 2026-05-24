## Why

Nutzer können auf der Mitfahrgelegenheiten-Seite nicht schnell erkennen, bei welchen Spielen sie selbst eingetragen sind. Ein „Meine"-Filter ermöglicht es, auf einen Blick nur die relevanten Spiele zu sehen.

## What Changes

- Neues Toggle rechts oben neben der `<h1>` (Button-Gruppe „Alle | Meine"), analog zum „Alle Dienste / Meine Dienste"-Toggle auf der Dienste-Seite
- Im Modus „Meine": nur Spiele anzeigen, bei denen der eingeloggte Nutzer mindestens einen Eintrag hat (biete oder suche)
- Tab-Counts (Auswärtsspiele / Heimspiele / Events) passen sich dem aktiven Filter an
- Rein client-seitig: `isOwn`-Flag ist bereits in der API-Response vorhanden, kein neuer Endpunkt nötig

## Capabilities

### New Capabilities

- `mitfahrgelegenheiten-meine-filter`: Client-seitiger Toggle-Filter auf der Mitfahrgelegenheiten-Seite zum Einschränken auf eigene Einträge

### Modified Capabilities

- `mitfahrgelegenheiten-board`: Filterlogik wird zum bestehenden Board hinzugefügt (kein Anforderungswechsel, nur UI-Erweiterung)

## Impact

- `web/src/pages/MitfahrgelegenheitenPage.tsx`: State `viewMine`, Filter-Logik, Button-Gruppe
- Keine Backend-Änderungen
- Keine neuen Dependencies

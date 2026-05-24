## Why

Bei Auswärtsspielen ist die Organisation von Fahrgemeinschaften aktuell außerhalb der App (WhatsApp, Telefon). Eltern und Spieler wissen nicht, wer Plätze anbietet oder sucht. Eine koordinierte Übersicht reduziert Aufwand und erhöht die Teilnahme. Das Dashboard-Accordion „Fahrtgemeinschaften" ist derzeit ein Platzhalter ohne koordinative Funktion.

## What Changes

- **Neue Seite** `/mitfahrgelegenheiten`: Listet kommende Auswärtsspiele mit je einer Übersicht der Angebote (biete Mitfahrt) und Gesuche (suche Mitfahrt) pro Spiel
- **Neues Schema** `mitfahrgelegenheiten`-Tabelle: Nutzer können sich pro Spiel als Fahrer oder Mitfahrer eintragen
- **Nav-Eintrag** im Bereich „Dienste", unterhalb von „Dienste" (nach Kalender)
- **Dashboard-Sektion** „Fahrtgemeinschaften" wird zu einem Link auf die neue Seite mit Kurzinfo (nächstes Auswärtsspiel + Angebots-/Gesuche-Zähler)
- **Fahrzeuginfo** (`vehicle_info`-Tabelle) bleibt in `/profil`; wird aber optional als Standardwert für neue Fahrer-Einträge verwendet

## Capabilities

### New Capabilities

- `mitfahrgelegenheiten-board`: Nutzer können Mitfahrangebote und -gesuche pro Auswärtsspiel eintragen, einsehen und zurückziehen
- `mitfahrgelegenheiten-nav`: Navigations-Eintrag und Dashboard-Link zur neuen Seite

### Modified Capabilities

- `dashboard-migration`: Dashboard-Sektion „Fahrtgemeinschaften" ändert Inhalt von inline VehicleSection zu kompaktem Link mit Kurzinfo

## Impact

**Backend:**
- Migration 013: neue Tabelle `mitfahrgelegenheiten`
- Neues Package `internal/carpooling/` mit Handler für CRUD der Einträge
- Neuer Endpoint `GET /api/mitfahrgelegenheiten` (alle Auswärtsspiele + Einträge)
- Neuer Endpoint `POST /api/mitfahrgelegenheiten` (eigenen Eintrag anlegen/aktualisieren)
- Neuer Endpoint `DELETE /api/mitfahrgelegenheiten/{id}` (eigenen Eintrag zurückziehen)
- `/api/dashboard`: `vehicleInfo`-Feld ergänzt um Kurzinfo für nächstes Auswärtsspiel

**Frontend:**
- `web/src/pages/MitfahrgelegenheitenPage.tsx` (neu)
- `web/src/components/AppShell.tsx`: Nav-Eintrag hinzufügen
- `web/src/pages/DashboardPage.tsx`: `VehicleSection`-Komponente durch Link-Karte ersetzen

**Keine Breaking Changes.** Keine neuen externen Abhängigkeiten.

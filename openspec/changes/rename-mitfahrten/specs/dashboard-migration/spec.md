## MODIFIED Requirements

### Requirement: Dashboard-Sektion Fahrtgemeinschaften wird zu Link

Die Fahrtgemeinschaften-Sektion des Dashboards SHALL die bisherige `VehicleSection`-Komponente durch eine kompakte Link-Karte zur `/mitfahrten`-Seite ersetzen. Die Karte SHALL eine Kurzinfo zum nächsten Auswärtsspiel (Datum, Gegner, Angebots-/Gesuch-Zähler) und einen Link zu `/mitfahrten` zeigen.

#### Scenario: Dashboard Fahrtgemeinschaften-Sektion mit nächstem Auswärtsspiel
- **WHEN** es ein zukünftiges Auswärtsspiel gibt
- **THEN** zeigt die Sektion: Datum + Gegner + „X Angebote, Y Gesuche" + Link `[→ Mitfahrten]`

#### Scenario: Kein Auswärtsspiel in Zukunft
- **WHEN** kein zukünftiges Auswärtsspiel existiert
- **THEN** zeigt die Sektion: „Keine Auswärtsfahrten geplant" + Link zur Seite

#### Scenario: Klick auf Link
- **WHEN** der Nutzer auf den Link in der Fahrtgemeinschaften-Sektion klickt
- **THEN** navigiert er zu `/mitfahrten`

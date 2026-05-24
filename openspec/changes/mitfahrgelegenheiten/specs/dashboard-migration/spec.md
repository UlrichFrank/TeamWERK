## MODIFIED Requirements

### Requirement: Dashboard-Sektion Fahrtgemeinschaften wird zu Link

Die bisherige `VehicleSection`-Komponente in der „Fahrtgemeinschaften"-Accordion-Sektion wird durch eine kompakte Link-Karte zur neuen `/mitfahrgelegenheiten`-Seite ersetzt.

**Vorher:** Zeigt Fahrzeuginfo (Sitzplätze) und Link zu `/profil`.

**Nachher:** Zeigt Kurzinfo zum nächsten Auswärtsspiel (Datum, Gegner, Angebots-/Gesuch-Zähler) und einen Link zu `/mitfahrgelegenheiten`.

#### Scenario: Dashboard Fahrtgemeinschaften-Sektion mit nächstem Auswärtsspiel
- **WHEN** es ein zukünftiges Auswärtsspiel gibt
- **THEN** zeigt die Sektion: Datum + Gegner + „X Angebote, Y Gesuche" + Link `[→ Mitfahrgelegenheiten]`

#### Scenario: Kein Auswärtsspiel in Zukunft
- **WHEN** kein zukünftiges Auswärtsspiel existiert
- **THEN** zeigt die Sektion: „Keine Auswärtsfahrten geplant" + Link zur Seite

#### Scenario: Klick auf Link
- **WHEN** der Nutzer auf den Link in der Fahrtgemeinschaften-Sektion klickt
- **THEN** navigiert er zu `/mitfahrgelegenheiten`

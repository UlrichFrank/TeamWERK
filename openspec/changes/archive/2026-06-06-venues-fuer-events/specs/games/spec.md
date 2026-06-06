## MODIFIED Requirements

### Requirement: Spiel anlegen
Spiele (Heim, Auswärts, Generisch) SHALL optional mit einem Venue verknüpft werden können. Bei Heimspielen wird venue_id automatisch auf den Heimhallen-Venue vorausgefüllt, sofern einer als `is_home_venue=true` markiert ist.

#### Scenario: Heimspiel mit Heimhallen-Autofill
- **WHEN** Nutzer öffnet Formular für neues Heimspiel
- **THEN** venue_id wird automatisch auf den is_home_venue-Venue gesetzt; Nutzer kann überschreiben

#### Scenario: Auswärtsspiel ohne Venue
- **WHEN** Nutzer legt Auswärtsspiel ohne Venue-Auswahl an
- **THEN** Spiel wird ohne venue_id gespeichert (null)

#### Scenario: Venue in Response eingebettet
- **WHEN** GET /api/games oder GET /api/games/{id} aufgerufen wird
- **THEN** Response enthält venue-Objekt mit id, name, street, city, postal_code, note (oder null wenn kein Venue)

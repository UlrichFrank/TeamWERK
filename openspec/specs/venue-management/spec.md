### Requirement: Venue anlegen
Admins SHALL Veranstaltungsorte mit Name, Straße, Stadt, PLZ, Land (Default: DE) und optionaler Notiz anlegen können.

#### Scenario: Erfolgreiche Anlage
- **WHEN** Admin sendet POST /api/admin/venues mit name, street, city, postal_code
- **THEN** Venue wird gespeichert und mit id, allen Feldern und created_at zurückgegeben

#### Scenario: Pflichtfelder fehlen
- **WHEN** Admin sendet POST ohne name, street, city oder postal_code
- **THEN** Server antwortet mit 400 Bad Request

---

### Requirement: Venue bearbeiten
Admins SHALL alle Felder eines bestehenden Venues aktualisieren können.

#### Scenario: Erfolgreiche Aktualisierung
- **WHEN** Admin sendet PUT /api/admin/venues/{id} mit geänderten Feldern
- **THEN** Venue wird aktualisiert und zurückgegeben

#### Scenario: Venue nicht gefunden
- **WHEN** Admin sendet PUT /api/admin/venues/{id} mit unbekannter id
- **THEN** Server antwortet mit 404 Not Found

---

### Requirement: Venue löschen
Admins SHALL Venues löschen können. Referenzierende Events verlieren dabei ihren Ort (venue_id → NULL).

#### Scenario: Erfolgreiches Löschen
- **WHEN** Admin sendet DELETE /api/admin/venues/{id}
- **THEN** Venue wird gelöscht; venue_id in referenzierenden games/training_series/training_sessions wird NULL

---

### Requirement: Venues auflisten
Admins und Trainer SHALL alle Venues abrufen können.

#### Scenario: Liste abrufen
- **WHEN** GET /api/admin/venues aufgerufen wird
- **THEN** Alle Venues werden als Array zurückgegeben (name, street, city, postal_code, country, note, is_home_venue)

---

### Requirement: Heimhalle markieren
Admins SHALL genau einen Venue als Heimhalle (`is_home_venue = true`) markieren können.

#### Scenario: Heimhalle setzen
- **WHEN** Admin setzt is_home_venue=true auf einem Venue
- **THEN** Alle anderen Venues erhalten is_home_venue=false; nur dieser Venue hat is_home_venue=true

#### Scenario: Heimhalle deaktivieren
- **WHEN** Admin setzt is_home_venue=false auf dem aktuellen Heimhallen-Venue
- **THEN** Kein Venue ist mehr als Heimhalle markiert (is_home_venue=false bei allen)

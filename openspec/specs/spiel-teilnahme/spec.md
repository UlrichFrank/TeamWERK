### Requirement: Spieldetail-Seite zeigt alle Kader-Mitglieder

Die Spieldetail-Seite (`/termine/spiel/{id}`) SHALL alle Mitglieder aus `kader_members` und `kader_extended_members` des zugehörigen Teams anzeigen, unabhängig davon, ob sie eine RSVP-Antwort abgegeben haben. Die Daten kommen aus dem neuen Endpoint `GET /api/games/{id}/participants`.

#### Scenario: Mitglied ohne RSVP erscheint in der Teilnahme-Tabelle

- **WHEN** ein reguläres Kader-Mitglied keine RSVP-Antwort für ein Spiel abgegeben hat
- **THEN** erscheint es trotzdem in der Teilnahme-Tabelle mit `rsvp_status: null` (Anzeige: „–")

#### Scenario: Erweitertes Kader-Mitglied erscheint ohne RSVP

- **WHEN** ein erweitertes Kader-Mitglied für das Team eingetragen ist
- **THEN** erscheint es in der Teilnahme-Tabelle mit `rsvp_status: null` und ohne die Möglichkeit, RSVP zu geben

#### Scenario: Spieldetail nutzt /participants statt /responses

- **WHEN** die Spieldetail-Seite geladen wird
- **THEN** lädt das Frontend `GET /api/games/{id}/participants` (nicht mehr `GET /api/games/{id}/responses`) als Datenquelle für die Teilnahme-Tabelle

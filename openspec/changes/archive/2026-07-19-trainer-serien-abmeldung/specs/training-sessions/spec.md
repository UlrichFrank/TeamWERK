## ADDED Requirements

### Requirement: Session-Antwort kennzeichnet den Abmelde-Status je Mitglied

`GET /api/training-sessions` und `GET /api/training-sessions/{id}` SHALL für jedes gelistete Mitglied ausweisen, ob es für die Serie der Session eine greifende Serien-Abmeldung (`serien-abmeldung`-Ableitung) hat, über ein Feld `unavailable: { reason, permanent } | null` (`permanent = true`, wenn `end_date IS NULL`). Der betroffene Spieler SHALL in der Anwesenheits-/RSVP-Liste weiterhin sichtbar bleiben (nicht ausgeblendet). Der Spieler SHALL seinen eigenen Abmelde-Status sehen können, ohne ihn ändern zu können.

#### Scenario: Abgemeldeter Spieler wird mit Status geliefert

- **WHEN** ein Nutzer `GET /api/training-sessions/{id}` für eine Session abruft, in der ein Mitglied für die Serie abgemeldet ist
- **THEN** enthält der Eintrag dieses Mitglieds `unavailable` mit `reason` und `permanent`, und das Mitglied bleibt Teil der Liste

#### Scenario: Nicht abgemeldeter Spieler hat unavailable = null

- **WHEN** kein greifender Abmelde-Eintrag für das Mitglied und die Serie existiert
- **THEN** ist `unavailable` für dieses Mitglied `null`

#### Scenario: Spieler sieht eigenen Abmelde-Status

- **WHEN** ein für die Serie abgemeldeter Spieler die Session abruft
- **THEN** wird sein eigener Eintrag mit `unavailable` geliefert (Anzeige „dauerhaft abgemeldet"), ohne dass ihm eine Änderungs- oder Löschaktion angeboten wird

## MODIFIED Requirements

### Requirement: Notification bei neuem Angebot
Wenn ein neues Bieter-Angebot eingestellt wird, SHALL das System allen Nutzern, die ein aktives Gesuch für dasselbe Spiel haben, eine Push-Benachrichtigung senden.

#### Scenario: Bieter legt Angebot an
- **WHEN** `POST /api/mitfahrten` mit `typ='biete'` erfolgreich ist
- **THEN** erhalten alle Nutzer mit einem offenen Gesuch (`typ='suche'`) für dasselbe Spiel eine Push-Notification mit dem Hinweis auf das neue Angebot und Deep-Link auf `/mitfahrten`

### Requirement: Notification bei neuer Suchanfrage
Wenn ein neues Gesuch eingestellt wird, SHALL das System allen Nutzern, die ein aktives Angebot für dasselbe Spiel haben, eine Push-Benachrichtigung senden.

#### Scenario: Sucher legt Gesuch an
- **WHEN** `POST /api/mitfahrten` mit `typ='suche'` erfolgreich ist
- **THEN** erhalten alle Nutzer mit einem offenen Angebot (`typ='biete'`) für dasselbe Spiel eine Push-Notification mit dem Hinweis auf das neue Gesuch und Deep-Link auf `/mitfahrten`

### Requirement: Notification bei zurückgezogenem Angebot
Wenn ein Bieter-Eintrag gelöscht oder deaktiviert wird, SHALL das System alle Sucher, die eine aktive Paarung mit diesem Angebot haben, per Push-Benachrichtigung informieren.

#### Scenario: Bieter zieht Angebot zurück
- **WHEN** `DELETE /api/mitfahrten/{id}` für einen Bieter-Eintrag aufgerufen wird
- **THEN** erhalten alle Sucher mit einer `pending`- oder `confirmed`-Paarung zu diesem Angebot eine Push-Notification über den Rückzug mit Deep-Link auf `/mitfahrten`

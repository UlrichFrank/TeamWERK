## ADDED Requirements

### Requirement: Anwärter als gültiger Member-Status
Das System SHALL `anwaerter` als validen Wert für `members.status` akzeptieren. Ein Anwärter ist ein Spieler in der Probezeit, der noch kein vollwertiges Vereinsmitglied ist und keinen eigenen System-Account benötigt.

#### Scenario: Vorstand legt Anwärter an
- **WHEN** der Vorstand ein neues Mitglied mit Status `anwaerter` anlegt (nur Name + Geburtsdatum als Pflichtfelder)
- **THEN** wird der Datensatz mit `status = 'anwaerter'` gespeichert und ist in der Mitgliederliste sichtbar

#### Scenario: Ungültiger Status wird abgelehnt
- **WHEN** ein Request `PUT /api/members/:id/status` mit einem unbekannten Status-Wert gesendet wird
- **THEN** antwortet die API mit HTTP 400

### Requirement: Anwärter über erweiterten Kader einbindbar
Das System SHALL es dem Vorstand ermöglichen, einen Anwärter dem erweiterten Kader (`kader_extended_members`) eines Teams zuzuordnen.

#### Scenario: Anwärter in Spieltag-Teilnehmerliste
- **WHEN** ein Anwärter im erweiterten Kader eines Teams eingetragen ist und ein Spiel dieses Teams stattfindet
- **THEN** erscheint der Anwärter in der Spieltag-Teilnehmerliste (`GET /api/games/:id/participants`) mit `is_extended = true`

#### Scenario: Anwärter in Aufstellung aufnehmbar
- **WHEN** ein Trainer einen Anwärter aus der Teilnehmerliste in die Aufstellung aufnimmt
- **THEN** wird der Anwärter in `game_lineup` eingetragen und erscheint in der Aufstellungsansicht

### Requirement: Visueller Anwärter-Hinweis im Kader
Das System SHALL in der Kader-Ansicht einen Badge oder ein Label für Mitglieder mit Status `anwaerter` anzeigen, damit Trainer den Anwärter-Status auf einen Blick erkennen.

#### Scenario: Badge in Kader-Ansicht
- **WHEN** ein Mitglied mit Status `anwaerter` im Kader (primär oder erweitert) eingetragen ist
- **THEN** zeigt die Kader-Ansicht ein "Anwärter"-Badge neben dem Namen des Mitglieds

### Requirement: Manueller Upgrade zu aktivem Mitglied
Das System SHALL es dem Vorstand ermöglichen, den Status eines Anwärters auf `aktiv` zu setzen, wenn die Aufnahme beschlossen wurde.

#### Scenario: Status-Upgrade durch Vorstand
- **WHEN** der Vorstand `PUT /api/members/:id/status` mit `{"status": "aktiv"}` aufruft
- **THEN** wird der Status des Anwärters auf `aktiv` gesetzt und der Datensatz kann vollständig befüllt werden (Passnummer, IBAN etc.)

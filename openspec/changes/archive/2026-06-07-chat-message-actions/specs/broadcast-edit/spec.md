## ADDED Requirements

### Requirement: Broadcasts können vom Sender bearbeitet werden
Das System SHALL es dem Sender einer Broadcast-Mitteilung ermöglichen, deren Text nachträglich zu ändern. Nur der ursprüngliche Sender darf bearbeiten. Bereits gelesene Mitteilungen werden nach der Bearbeitung nicht erneut als ungelesen markiert.

#### Scenario: Edit-Button für Sender in der Broadcast-Ansicht
- **WHEN** der Sender eine eigene Broadcast-Mitteilung in der Detailansicht öffnet
- **THEN** ist ein Bearbeiten-Button (Pencil-Icon) neben dem bestehenden Löschen-Button sichtbar

#### Scenario: Edit-Modal öffnen
- **WHEN** der Sender auf den Bearbeiten-Button klickt
- **THEN** öffnet sich ein Modal mit dem aktuellen Broadcast-Text im Textarea-Feld

#### Scenario: Bearbeitung speichern
- **WHEN** der Sender den Text ändert und speichert
- **THEN** sendet das Frontend PUT `/api/chat/broadcasts/{id}` mit dem neuen Text, das Backend aktualisiert `body` und setzt `edited_at = CURRENT_TIMESTAMP`, das Modal schließt sich, die Broadcast-Liste wird neu geladen

#### Scenario: Kein Edit durch andere Nutzer
- **WHEN** ein Nicht-Sender PUT `/api/chat/broadcasts/{id}` aufruft
- **THEN** antwortet das Backend mit HTTP 403

#### Scenario: Bearbeiteter Broadcast-Indikator
- **WHEN** eine bearbeitete Broadcast-Mitteilung angezeigt wird (editedAt gesetzt)
- **THEN** erscheint in der Detailansicht der Hinweis „(bearbeitet)" unterhalb des Datums

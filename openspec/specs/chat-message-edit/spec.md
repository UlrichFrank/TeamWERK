# chat-message-edit Specification

## Purpose
Eigene Chat-Nachrichten können nachträglich bearbeitet werden, ohne Zeitlimit.

## Requirements

### Requirement: Eigene Nachrichten können bearbeitet werden
Das System SHALL es dem Absender ermöglichen, den Text einer eigenen Nachricht nachträglich zu ändern. Es gibt kein Zeitlimit. Bearbeitete Nachrichten sind als solche gekennzeichnet.

#### Scenario: Bearbeiten via Rechtsklick auf Desktop
- **WHEN** der Nutzer auf einer eigenen Nachrichten-Bubble einen Rechtsklick ausführt
- **THEN** enthält das Kontext-Menü den Eintrag „Bearbeiten" (Pencil-Icon)

#### Scenario: Bearbeiten-Modus öffnen
- **WHEN** der Nutzer „Bearbeiten" im Kontext-Menü wählt
- **THEN** erscheint eine Edit-Leiste über dem Eingabefeld mit Pencil-Icon und Text „Nachricht bearbeiten", das Eingabefeld wird mit dem aktuellen Nachrichtentext befüllt

#### Scenario: Bearbeitung speichern
- **WHEN** der Nutzer den Text im Eingabefeld ändert und sendet
- **THEN** sendet das Frontend PUT `/api/chat/messages/{id}` mit dem neuen Text, die Nachricht wird in der Liste aktualisiert, der Edit-Modus wird geschlossen

#### Scenario: Bearbeiteter Nachricht-Indikator
- **WHEN** eine Nachricht mit gesetztem `editedAt` angezeigt wird
- **THEN** erscheint unterhalb des Zeitstempels der Hinweis „(bearbeitet)" in gedämpfter Farbe

#### Scenario: Keine Bearbeitung fremder Nachrichten
- **WHEN** der Nutzer auf einer fremden Nachrichten-Bubble einen Rechtsklick ausführt
- **THEN** enthält das Kontext-Menü keinen „Bearbeiten"-Eintrag

#### Scenario: Keine Bearbeitung gelöschter Nachrichten
- **WHEN** eine Nachricht bereits gelöscht wurde
- **THEN** MUSS PUT `/api/chat/messages/{id}` mit HTTP 404 oder 403 antworten

#### Scenario: Edit-Modus abbrechen
- **WHEN** der Nutzer auf das X-Icon in der Edit-Leiste klickt
- **THEN** wird der Edit-Modus beendet, das Eingabefeld geleert, keine Änderung gespeichert

## ADDED Requirements

### Requirement: Event löschen auf Detailseite

Berechtigte User (admin, vorstand, trainer) SHALL auf der Spieltag-Detailseite ein Event löschen können.

#### Scenario: Löschen-Button sichtbar für berechtigte Rollen
- **WHEN** ein User mit Rolle admin, vorstand oder trainer die Spieltag-Detailseite aufruft
- **THEN** ist ein „Event löschen"-Button sichtbar

#### Scenario: Löschen-Button nicht sichtbar für andere Rollen
- **WHEN** ein User mit Rolle spieler oder elternteil die Spieltag-Detailseite aufruft
- **THEN** ist kein „Event löschen"-Button vorhanden

#### Scenario: Bestätigungs-Dialog erscheint
- **WHEN** ein berechtigter User auf „Event löschen" klickt
- **THEN** erscheint ein Bestätigungs-Dialog mit dem Eventnamen und einem Hinweis dass alle zugehörigen Dienste mitgelöscht werden

#### Scenario: Löschen abbrechen
- **WHEN** der User im Dialog auf „Abbrechen" klickt
- **THEN** schließt der Dialog ohne Aktion

#### Scenario: Löschen bestätigen
- **WHEN** der User im Dialog auf „Endgültig löschen" klickt
- **THEN** wird `DELETE /api/admin/games/{id}` aufgerufen
- **THEN** bei Erfolg wird der User zur Spielplan-Übersicht (`/spielplan`) weitergeleitet

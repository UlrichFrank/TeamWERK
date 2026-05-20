## ADDED Requirements

### Requirement: Erziehungsberechtigten entfernen

Das System MUSS das Entfernen einer Erziehungsberechtigten-Verknüpfung ermöglichen.

`DELETE /api/admin/family-links` mit Body:
```json
{ "parent_user_id": 12, "member_id": 6 }
```

Der Endpoint MUSS mit 204 No Content antworten wenn die Verknüpfung erfolgreich gelöscht wurde.
Der Endpoint MUSS mit 404 antworten wenn die Verknüpfung nicht existiert.
Der Endpoint ist nur für Nutzer mit Rolle `admin` zugänglich.

Jeder verknüpfte Erziehungsberechtigte MUSS in der UI einen Entfernen-Button haben.

#### Scenario: Verknüpfung erfolgreich entfernen

- **WHEN** `DELETE /api/admin/family-links` mit gültiger `parent_user_id` und `member_id` gesendet wird
- **THEN** antwortet der Server mit 204 No Content
- **THEN** erscheint der entfernte Nutzer nicht mehr in `GET /admin/members/{id}/parents`

#### Scenario: Nicht-existierende Verknüpfung entfernen

- **WHEN** `DELETE /api/admin/family-links` mit einer Kombination gesendet wird, die nicht existiert
- **THEN** antwortet der Server mit 404 Not Found

#### Scenario: Entfernen-Button in der UI

- **WHEN** ein Erziehungsberechtigter verknüpft ist
- **THEN** zeigt die UI neben seinem Namen einen Entfernen-Button
- **WHEN** der Button geklickt wird
- **THEN** wird die Verknüpfung entfernt und die Liste aktualisiert

### Requirement: Maximale Anzahl Erziehungsberechtigte

Ein Mitglied DARF maximal zwei Erziehungsberechtigte verknüpft haben.

Das Backend MUSS bei `POST /api/admin/family-links` prüfen, ob bereits 2 Verknüpfungen für `member_id` existieren, und mit 409 Conflict antworten wenn ja.

Das Frontend MUSS den Hinzufügen-Button deaktivieren wenn bereits 2 Erziehungsberechtigte verknüpft sind.

#### Scenario: Dritter Erziehungsberechtigter wird abgelehnt

- **WHEN** für ein Mitglied bereits 2 Erziehungsberechtigte existieren
- **THEN** antwortet `POST /api/admin/family-links` mit 409 Conflict

#### Scenario: Hinzufügen-Button deaktiviert bei 2 Einträgen

- **WHEN** bereits 2 Erziehungsberechtigte verknüpft sind
- **THEN** ist der Button zum Hinzufügen deaktiviert oder nicht sichtbar

### Requirement: Alle Nutzer als Erziehungsberechtigte verknüpfbar

Das System MUSS alle aktiven System-Nutzer als Erziehungsberechtigte verknüpfbar machen,
unabhängig von ihrer Rolle (`admin`, `trainer`, `spieler`, `elternteil`).

Das Dropdown in der UI DARF NICHT nach Rolle filtern; bereits verknüpfte Nutzer MÜSSEN
aus dem Dropdown ausgeblendet werden.

#### Scenario: Nutzer mit Rolle spieler als Erziehungsberechtigter

- **WHEN** das Dropdown für Erziehungsberechtigte geöffnet wird
- **THEN** erscheinen Nutzer aller Rollen (nicht nur `elternteil`)

#### Scenario: Bereits verknüpfte Nutzer nicht im Dropdown

- **WHEN** Nutzer A bereits als Erziehungsberechtigter verknüpft ist
- **THEN** erscheint Nutzer A nicht im Dropdown für weitere Verknüpfungen

### Requirement: Benennung Erziehungsberechtigte

Der Begriff „Elternteile" MUSS in der gesamten UI durch „Erziehungsberechtigte" ersetzt werden.
Die API-Pfade und DB-Tabellennamen bleiben unverändert (`family_links`).

#### Scenario: Korrekte Bezeichnung in der UI

- **WHEN** die Mitglieder-Detailseite aufgerufen wird
- **THEN** lautet die Abschnittsüberschrift „Erziehungsberechtigte" (nicht „Elternteile")

## MODIFIED Requirements

### Requirement: Beitrittsanfrage mit getrenntem Vor- und Nachname
Das System SHALL beim Einreichen einer Beitrittsanfrage `first_name` und `last_name` als separate Felder entgegennehmen. Das kombinierte Feld `name` entfällt. Beide Felder sind Pflichtfelder.

#### Scenario: Nutzer reicht Beitrittsanfrage ein
- **WHEN** ein Besucher `POST /api/auth/request-membership` mit `{ "first_name": "Max", "last_name": "Mustermann", "email": "…", "comment": "…" }` aufruft
- **THEN** speichert das System die Anfrage mit `first_name` und `last_name` getrennt, Status `pending`, und benachrichtigt Trainer und Admins per E-Mail

#### Scenario: Fehlender Vorname wird abgelehnt
- **WHEN** `POST /api/auth/request-membership` ohne `first_name` oder mit leerem `first_name` aufgerufen wird
- **THEN** antwortet das System mit HTTP 400

#### Scenario: Admin-Ansicht zeigt Vor- und Nachname
- **WHEN** ein Admin oder Trainer die Liste der offenen Beitrittsanfragen aufruft
- **THEN** wird für jede Anfrage Vorname und Nachname separat (oder als „Vorname Nachname") angezeigt

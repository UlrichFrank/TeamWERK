# user-list-filters Specification

## Purpose

Diese Spezifikation beschreibt die Capability `user-list-filters`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Nutzerliste zeigt family_link-Status
`GET /api/users` SHALL im Response-Objekt jedes Users ein Feld `has_family_link: bool` enthalten.
`has_family_link` ist `true`, wenn für diesen User mindestens ein Eintrag in `family_links` als `parent_user_id` existiert.

#### Scenario: Eltern-User hat has_family_link true
- **WHEN** ein User existiert, der in `family_links` als `parent_user_id` eingetragen ist
- **THEN** hat dieser User im Response `has_family_link: true`

#### Scenario: User ohne Family-Link hat has_family_link false
- **WHEN** ein User keinen Eintrag als `parent_user_id` in `family_links` hat
- **THEN** hat dieser User im Response `has_family_link: false`

### Requirement: Nutzerliste nach fehlender Mitgliedsverknüpfung filtern
`GET /api/users` SHALL einen optionalen Query-Parameter `unlinked=1` unterstützen.
Wenn gesetzt, MÜSSEN nur User zurückgegeben werden, für die gilt:
- Kein Mitglied hat `members.user_id = u.id` (kein direktes Mitglied)
- AND `NOT EXISTS (SELECT 1 FROM family_links WHERE parent_user_id = u.id)` (kein Family-Link)

#### Scenario: Filter liefert nur vollständig entkoppelte User
- **WHEN** `GET /api/users?unlinked=1` mit admin-Token aufgerufen wird
- **THEN** enthält die Antwort nur User ohne direkte Mitgliedsverknüpfung und ohne Family-Link

#### Scenario: User mit direktem Mitglied wird ausgeschlossen
- **WHEN** ein User über `members.user_id` mit einem Mitglied verknüpft ist
- **THEN** erscheint er nicht in der Antwort von `GET /api/users?unlinked=1`

#### Scenario: Eltern-User mit nur Family-Link wird ausgeschlossen
- **WHEN** ein User als `parent_user_id` in `family_links` eingetragen ist, aber kein direktes Mitglied hat
- **THEN** erscheint er nicht in der Antwort von `GET /api/users?unlinked=1`

### Requirement: "Mitglied erstellen"-Button erscheint nur für wirklich unverknüpfte User
In der AdminUsersPage SHALL der "Mitglied erstellen"-Button nur angezeigt werden, wenn der User weder ein direktes Mitglied (`member_id == null`) noch einen Family-Link (`has_family_link == false`) hat.

#### Scenario: Eltern-User ohne direktes Mitglied bekommt keinen "Mitglied erstellen"-Button
- **WHEN** ein User `member_id: null` und `has_family_link: true` hat
- **THEN** wird der "Mitglied erstellen"-Button für diesen User nicht angezeigt

#### Scenario: Vollständig unverknüpfter User bekommt "Mitglied erstellen"-Button
- **WHEN** ein User `member_id: null` und `has_family_link: false` hat
- **THEN** wird der "Mitglied erstellen"-Button für diesen User angezeigt

### Requirement: Filter-Toggle "Ohne Mitgliedsverknüpfung" in der Nutzerlisten-UI
Die AdminUsersPage SHALL einen Toggle "Ohne Mitgliedsverknüpfung" anzeigen.
Wenn aktiviert, wird die Nutzerliste mit `?unlinked=1` neu geladen und die Paginierung zurückgesetzt.

#### Scenario: Toggle aktivieren filtert die Liste
- **WHEN** der Nutzer den Toggle "Ohne Mitgliedsverknüpfung" aktiviert
- **THEN** wird die Nutzerliste mit `?unlinked=1` neu geladen
- **AND** nur User ohne jede Mitgliedsverknüpfung werden angezeigt

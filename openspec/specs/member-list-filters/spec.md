# member-list-filters Specification

## Purpose

Diese Spezifikation beschreibt die Capability `member-list-filters`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Mitgliederliste nach fehlender Nutzerverknüpfung filtern
`GET /api/members` SHALL einen optionalen Query-Parameter `unlinked_user=1` unterstützen.
Wenn gesetzt, MÜSSEN nur Mitglieder zurückgegeben werden, für die gilt:
- `members.user_id IS NULL`
- AND `NOT EXISTS (SELECT 1 FROM family_links WHERE member_id = m.id)`

Der Filter ist nur für Rollen mit `wideSearch`-Berechtigung (admin, vorstand, sportliche_leitung) wirksam.

#### Scenario: Filter liefert nur Mitglieder ohne jede Nutzerzuordnung
- **WHEN** `GET /api/members?unlinked_user=1` mit admin-Token aufgerufen wird
- **THEN** enthält die Antwort nur Mitglieder, die weder eine direkte `user_id` noch einen family_links-Eintrag haben

#### Scenario: Mitglied mit direktem User-Account wird ausgeschlossen
- **WHEN** `GET /api/members?unlinked_user=1` aufgerufen wird
- **THEN** sind Mitglieder mit gesetztem `user_id` nicht in der Antwort enthalten

#### Scenario: Mitglied mit Family-Link-Elternteil wird ausgeschlossen
- **WHEN** `GET /api/members?unlinked_user=1` aufgerufen wird
- **THEN** sind Mitglieder, für die ein `family_links`-Eintrag existiert, nicht in der Antwort enthalten

#### Scenario: Filter kombinierbar mit Suche
- **WHEN** `GET /api/members?unlinked_user=1&search=Müller` aufgerufen wird
- **THEN** werden nur Mitglieder zurückgegeben, die beide Bedingungen erfüllen

### Requirement: Mitgliederliste nach offenen Änderungsanträgen filtern
`GET /api/members` SHALL einen optionalen Query-Parameter `has_draft=1` unterstützen.
Wenn gesetzt, MÜSSEN nur Mitglieder zurückgegeben werden, für die mindestens ein Eintrag in `member_change_drafts` existiert.

Der Filter ist nur für Rollen mit `wideSearch`-Berechtigung wirksam.

#### Scenario: Filter liefert nur Mitglieder mit offenem Draft
- **WHEN** `GET /api/members?has_draft=1` mit admin-Token aufgerufen wird
- **THEN** enthält die Antwort nur Mitglieder mit mindestens einem Eintrag in `member_change_drafts`

#### Scenario: Mitglied ohne Drafts wird ausgeschlossen
- **WHEN** `GET /api/members?has_draft=1` aufgerufen wird
- **THEN** sind Mitglieder ohne Einträge in `member_change_drafts` nicht in der Antwort

#### Scenario: Filter kombinierbar mit unlinked_user
- **WHEN** `GET /api/members?has_draft=1&unlinked_user=1` aufgerufen wird
- **THEN** werden nur Mitglieder zurückgegeben, die beide Bedingungen gleichzeitig erfüllen

### Requirement: Filter-Toggles in der Mitgliederlisten-UI
Die MembersPage SHALL zwei neue Checkbox-/Toggle-Filter für Vorstand/Admin anzeigen:
- "Ohne App-Account" (`unlinked_user=1`)
- "Mit Änderungsantrag" (`has_draft=1`)

Die Filter MÜSSEN serverseitig wirken (Query-Params werden an `GET /api/members` übergeben) und die Paginierung zurücksetzen wenn sie geändert werden.

#### Scenario: Toggle "Ohne App-Account" aktivieren
- **WHEN** der Nutzer den Toggle "Ohne App-Account" aktiviert
- **THEN** wird die Mitgliederliste neu geladen mit `?unlinked_user=1`
- **AND** die Paginierung springt auf Seite 1

#### Scenario: Toggle "Mit Änderungsantrag" aktivieren
- **WHEN** der Nutzer den Toggle "Mit Änderungsantrag" aktiviert
- **THEN** wird die Mitgliederliste neu geladen mit `?has_draft=1`
- **AND** die Paginierung springt auf Seite 1

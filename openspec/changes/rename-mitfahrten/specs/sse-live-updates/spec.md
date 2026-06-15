## MODIFIED Requirements

### Requirement: SSE-Endpoint sendet typisierte Refresh-Signale

Der Server SHALL einen SSE-Endpoint `GET /api/events` bereitstellen. Der Endpoint SHALL authentifizierte Verbindungen offen halten und typisierte Event-Strings senden (`data: <event-typ>\n\n`), wenn eine Mutation in einem der folgenden Bereiche stattfindet: `mitfahrten`, `members`, `duties`, `games`, `settings`, `trainings`, `venues`, `absences`, `kader`.

**`"members"`-Event** SHALL auch bei folgenden Operationen gesendet werden:
- CreateFamilyLink / DeleteFamilyLink
- LinkUser (Nutzer mit Mitglied verknüpfen)
- UpdateProfile
- AddPhone / UpdatePhone / DeletePhone
- UpdateVehicle
- UpdateChildAccount / UpdateChildBank

**`"games"`-Event** SHALL auch bei Template-CRUD gesendet werden:
- CreateTemplate / UpdateTemplate / DeleteTemplate
- CreateTemplateItem / UpdateTemplateItem / DeleteTemplateItem

**`"kader"`-Event** SHALL bei allen Kader-Mutationen gesendet werden (siehe sse-kader-sync/spec.md).

#### Scenario: Neuer Mitfahrten-Eintrag löst Event aus

- **WHEN** ein Nutzer `POST /api/mitfahrten` (Upsert) aufruft
- **THEN** erhalten alle verbundenen SSE-Clients innerhalb von 1 Sekunde `data: mitfahrten`

#### Scenario: Paarungsanfrage löst Event aus

- **WHEN** ein Nutzer `POST /api/mitfahrt-paarungen` aufruft
- **THEN** erhalten alle verbundenen SSE-Clients `data: mitfahrten`

#### Scenario: Mitglieds-Mutation löst Event aus

- **WHEN** ein Admin oder Trainer ein Mitglied anlegt, bearbeitet oder dessen Status ändert
- **THEN** erhalten alle verbundenen SSE-Clients `data: members`

#### Scenario: Familienlink wird angelegt

- **WHEN** ein Vorstand oder Elternteil `POST /api/family-links` aufruft
- **THEN** erhalten alle verbundenen SSE-Clients `data: members`

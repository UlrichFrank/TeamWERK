## MODIFIED Requirements

### Requirement: SSE-Endpoint sendet typisierte Refresh-Signale

Der Server SHALL einen SSE-Endpoint `GET /api/events` bereitstellen. Der Endpoint SHALL authentifizierte Verbindungen offen halten und typisierte Event-Strings senden (`data: <event-typ>\n\n`), wenn eine Mutation in einem der folgenden Bereiche stattfindet: `mitfahrgelegenheiten`, `members`, `duties`, `games`, `settings`, `trainings`, `venues`, `absences`, `kader`.

Zusätzlich zu den bestehenden Broadcasts gelten folgende Erweiterungen:

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

**`"kader"`-Event** (neu) SHALL bei allen Kader-Mutationen gesendet werden (siehe sse-kader-sync/spec.md).

#### Scenario: Familienlink wird angelegt

- **WHEN** ein Vorstand oder Elternteil `POST /api/family-links` aufruft
- **THEN** erhalten alle verbundenen SSE-Clients `data: members`

#### Scenario: Profil wird aktualisiert

- **WHEN** ein Nutzer `PUT /api/profile/me` aufruft
- **THEN** erhalten alle verbundenen SSE-Clients `data: members`

#### Scenario: Duty-Template wird bearbeitet

- **WHEN** ein Vorstand ein Template erstellt, bearbeitet oder löscht
- **THEN** erhalten alle verbundenen SSE-Clients `data: games`

### Requirement: Frontend ersetzt manuellen Reload durch EventSource

Alle relevanten Pages SHALL eine `useLiveUpdates`-Verbindung zum SSE-Endpoint aufbauen und bei einem passenden `message`-Event die Daten still neu laden. Die folgende Tabelle definiert die vollständige Zuordnung:

| Seite | Abonnierte Events |
|---|---|
| DashboardPage | `games`, `trainings`, `duties`, `absences`, `mitfahrgelegenheiten` |
| AdminTrainingsPage | `trainings` |
| AdminSettingsPage | `settings` |
| MemberDetailPage | `members` |
| AdminKaderPage | `kader` |
| MeinTeamPage | `members`, `kader` |
| AdminUsersPage | `members` |
| MembershipRequestsPage | `members` |
| AdminDutyTypesPage | `duties` |
| AdminDutyTemplatesPage | `duties` |
| AdminDutyTemplateDetailPage | `duties` |
| (bereits vorhanden) KalenderPage | `games`, `trainings`, `absences` |
| (bereits vorhanden) TerminePage | `trainings`, `games` |
| (bereits vorhanden) DutyPage | `duties` |
| (bereits vorhanden) MembersPage | `members` |
| (bereits vorhanden) MitfahrgelegenheitenPage | `mitfahrgelegenheiten` |
| (bereits vorhanden) AdminVenuesPage | `venues` |

#### Scenario: Dashboard aktualisiert sich nach Spielplan-Änderung

- **WHEN** ein Trainer ein Spiel anlegt oder bearbeitet
- **THEN** lädt das Dashboard des eingeloggten Nutzers die Daten still neu

#### Scenario: AdminTrainingsPage aktualisiert sich

- **WHEN** eine Trainings-Serie oder -Session erstellt, bearbeitet oder gelöscht wird
- **THEN** lädt die AdminTrainingsPage still neu

#### Scenario: MemberDetailPage aktualisiert sich

- **WHEN** ein Admin ein Mitglied bearbeitet
- **THEN** lädt die geöffnete MemberDetailPage still neu

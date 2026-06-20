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

**`"kader"`-Event** (neu) SHALL bei allen Kader-Mutationen gesendet werden (siehe sse-kader-sync/spec.md).

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

#### Scenario: Profil wird aktualisiert

- **WHEN** ein Nutzer `PUT /api/profile/me` aufruft
- **THEN** erhalten alle verbundenen SSE-Clients `data: members`

#### Scenario: Dienst-Mutation löst Event aus

- **WHEN** ein Admin oder Trainer einen Dienst-Slot anlegt, bearbeitet oder löscht, oder eine Zuweisung erfüllt/als Geldersatz markiert
- **THEN** erhalten alle verbundenen SSE-Clients `data: duties`

#### Scenario: Duty-Template wird bearbeitet

- **WHEN** ein Vorstand ein Template erstellt, bearbeitet oder löscht
- **THEN** erhalten alle verbundenen SSE-Clients `data: games`

#### Scenario: Trainings-Mutation löst Event aus

- **WHEN** ein Nutzer eine Trainings-Session oder Trainingsserie erstellt, bearbeitet oder löscht, oder einen RSVP abgibt
- **THEN** erhalten alle verbundenen SSE-Clients `data: trainings`

#### Scenario: Keepalive verhindert Verbindungsabbruch

- **WHEN** 30 Sekunden keine Mutation stattgefunden hat
- **THEN** sendet der Server einen SSE-Kommentar (`: ping`) um die Verbindung offen zu halten

### Requirement: Frontend ersetzt manuellen Reload durch EventSource

Alle relevanten Pages SHALL eine `useLiveUpdates`-Verbindung zum SSE-Endpoint aufbauen und bei einem passenden `message`-Event die Daten still neu laden (ohne sichtbaren Ladespinner). Die SSE-Verbindung MUSS nach einem Access-Token-Refresh automatisch neu aufgebaut werden, um Reconnect-Schleifen mit abgelaufenen Tokens zu vermeiden.

| Seite | Abonnierte Events |
|---|---|
| DashboardPage | `games`, `trainings`, `duties`, `absences`, `mitfahrten` |
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
| (bereits vorhanden) MitfahrtenPage | `mitfahrten` |
| (bereits vorhanden) AdminVenuesPage | `venues` |

#### Scenario: Seite aktualisiert sich bei fremder Änderung

- **WHEN** ein anderer Nutzer einen Eintrag anlegt, ändert oder löscht
- **THEN** lädt die Seite des beobachtenden Nutzers die Daten neu ohne sichtbaren Ladespinner

#### Scenario: EventSource wird beim Verlassen der Seite aufgeräumt

- **WHEN** der Nutzer eine Seite mit `useLiveUpdates` verlässt
- **THEN** wird die SSE-Verbindung geschlossen (`es.close()`)

#### Scenario: Page ignoriert nicht relevante Events

- **WHEN** ein `members`-Event eintrifft und die aktuelle Seite nur auf `duties`-Events abonniert ist
- **THEN** lädt die Seite NICHT neu

#### Scenario: EventSource wird nach Token-Refresh neu aufgebaut

- **WHEN** der Access Token durch den 401-Interceptor erneuert wurde
- **THEN** baut `useLiveUpdates` eine neue EventSource-Verbindung auf
- **THEN** gibt es keine Reconnect-Schleife mit dem abgelaufenen Token

#### Scenario: Dashboard aktualisiert sich nach Spielplan-Änderung

- **WHEN** ein Trainer ein Spiel anlegt oder bearbeitet
- **THEN** lädt das Dashboard des eingeloggten Nutzers die Daten still neu

#### Scenario: AdminTrainingsPage aktualisiert sich

- **WHEN** eine Trainings-Serie oder -Session erstellt, bearbeitet oder gelöscht wird
- **THEN** lädt die AdminTrainingsPage still neu

#### Scenario: MemberDetailPage aktualisiert sich

- **WHEN** ein Admin ein Mitglied bearbeitet
- **THEN** lädt die geöffnete MemberDetailPage still neu

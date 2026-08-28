## MODIFIED Requirements

### Requirement: Frontend ersetzt manuellen Reload durch EventSource

Alle relevanten Pages SHALL eine `useLiveUpdates`-Verbindung zum SSE-Endpoint aufbauen und bei einem passenden `message`-Event die Daten still neu laden (ohne sichtbaren Ladespinner). Die SSE-Verbindung MUSS nach einem Access-Token-Refresh automatisch neu aufgebaut werden, um Reconnect-Schleifen mit abgelaufenen Tokens zu vermeiden.

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
| (bereits vorhanden) KalenderPage | `games`, `trainings`, `absences` |
| (bereits vorhanden) TerminePage | `trainings`, `games` |
| (bereits vorhanden) DutyPage | `duties` |
| (bereits vorhanden) MembersPage | `members` |
| (bereits vorhanden) MitfahrgelegenheitenPage | `mitfahrgelegenheiten` |
| (bereits vorhanden) AdminVenuesPage | `venues` |

Die Zeile `AdminDutyTemplateDetailPage` entfällt: die Vorlagen-Detailseite und ihre Route
`/admin/dienstplan-vorlagen/{id}` existieren nicht mehr. Der Item-Editor sitzt jetzt im
Modal der Listenseite (`AdminDutyTemplatesPage`), die `duties` bereits abonniert — der
Dauer-Modus musste sonst in zwei Masken doppelt gepflegt werden.

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

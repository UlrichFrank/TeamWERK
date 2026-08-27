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

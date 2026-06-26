# sse-live-updates Specification

## Purpose

Diese Spezifikation beschreibt die Capability `sse-live-updates`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: SSE-Endpoint sendet typisierte Refresh-Signale

Der Server SHALL einen SSE-Endpoint `GET /api/events` bereitstellen. Der Endpoint SHALL authentifizierte Verbindungen offen halten und typisierte Event-Strings senden (`data: <event-typ>\n\n`), wenn eine Mutation in einem der folgenden Bereiche stattfindet: `mitfahrgelegenheiten`, `members`, `duties`, `games`, `settings`, `trainings`, `venues`, `absences`, `kader`.

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

#### Scenario: Neuer Mitfahrgelegenheiten-Eintrag löst Event aus

- **WHEN** ein Nutzer `POST /api/mitfahrgelegenheiten` (Upsert) aufruft
- **THEN** erhalten alle verbundenen SSE-Clients innerhalb von 1 Sekunde `data: mitfahrgelegenheiten`

#### Scenario: Paarungsanfrage löst Event aus

- **WHEN** ein Nutzer `POST /api/mitfahrt-paarungen` aufruft
- **THEN** erhalten alle verbundenen SSE-Clients `data: mitfahrgelegenheiten`

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

### Requirement: Auth via Cookie am SSE-Endpoint

Da `EventSource` keine Custom-Header unterstützt, SHALL der SSE-Endpunkt `GET /api/events` über das HttpOnly-Refresh-Token-Cookie authentifiziert werden. Die Nutzung eines `?token=<jwt>`-Query-Parameters MUST entfernt werden, da Access Tokens in URL-Query-Parametern in Server-Logs, Browser-Verlauf und Proxy-Logs erscheinen. Das Backend MUST den Cookie-basierten Auth-Pfad in der Middleware für den SSE-Endpunkt unterstützen.

#### Scenario: Verbindungsaufbau mit gültigem Cookie

- **WHEN** ein eingeloggter Nutzer `GET /api/events` mit einem gültigen HttpOnly-Refresh-Token-Cookie aufruft
- **THEN** wird die Verbindung akzeptiert und offen gehalten

#### Scenario: Verbindungsaufbau ohne Token schlägt fehl

- **WHEN** ein nicht-authentifizierter Request den SSE-Endpoint aufruft (kein Cookie, kein Bearer Token)
- **THEN** antwortet der Server mit HTTP 401

#### Scenario: Access Token NICHT im Query-Parameter

- **WHEN** ein Client `GET /api/events?token=<jwt>` aufruft (altes Verhalten)
- **THEN** wird der `?token`-Query-Parameter NICHT als Authentifizierungsmittel akzeptiert

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
| AdminDutyTemplateDetailPage | `duties` |
| (bereits vorhanden) KalenderPage | `games`, `trainings`, `absences` |
| (bereits vorhanden) TerminePage | `trainings`, `games` |
| (bereits vorhanden) DutyPage | `duties` |
| (bereits vorhanden) MembersPage | `members` |
| (bereits vorhanden) MitfahrgelegenheitenPage | `mitfahrgelegenheiten` |
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

### Requirement: SSE-Endpoint sendet Versions-Event beim Verbindungsaufbau

Der SSE-Handler SHALL beim Aufbau jeder neuen Verbindung als erstes Event `data: __version:<hash>\n\n` senden, bevor reguläre Mutations-Events gesendet werden. Der `<hash>` ist der zur Compile-Zeit eingebettete Build-Hash.

#### Scenario: Neuer Client empfängt Versions-Event beim Connect

- **WHEN** ein authentifizierter Client `GET /api/events?token=<jwt>` aufruft
- **THEN** sendet der Server innerhalb von 100ms das Event `data: __version:<hash>`
- **THEN** folgen danach reguläre Mutations-Events (keepalive, domain-events)

#### Scenario: Reconnect nach Server-Neustart sendet neuen Hash

- **WHEN** ein Client nach einem Server-Neustart die SSE-Verbindung neu aufbaut
- **THEN** sendet der neue Server seinen aktuellen Build-Hash als `__version:`-Event
- **THEN** unterscheidet sich dieser Hash vom Hash des vorherigen Servers (da neues Binary)

#### Scenario: Bestehende useLiveUpdates-Nutzung bleibt unverändert

- **WHEN** eine Seite `useLiveUpdates` nutzt und ein `__version:`-Event empfängt
- **THEN** wird das Event NICHT an den `onEvent`-Callback weitergeleitet
- **THEN** verarbeitet `useLiveUpdates` nur Events ohne `__version:`-Prefix

# permissions Specification

## Purpose

Definiert die verbindliche Autorisierungs-Matrix von TeamWERK: welche Persona
(Kombination aus System-Rolle, Vereinsfunktionen und Eltern-Status) welche
Backend-Routen erreichen darf und welche Frontend-Routen, Navigations-Items und
Page-internen Aktionen sichtbar sind. Dient als Quelle der Wahrheit für die
mechanischen Drift-Tests (`TestPermissionMatrix_Backend`, Vitest-Smoke-Tests).

## Requirements

### Requirement: Persona-Definition

Das System SHALL die folgenden 12 Personas als Test-Fixtures bereitstellen. Sie decken alle praktisch relevanten Kombinationen aus System-Rolle, Vereinsfunktion(en) und Eltern-Status ab, mit besonderem Fokus auf den im Verein häufigen Fall, dass funktionsführende Mitglieder gleichzeitig Eltern sind.

| Persona-ID | role | club_functions | is_parent |
|---|---|---|---|
| `admin` | `admin` | `[]` | `false` |
| `vorstand` | `standard` | `["vorstand"]` | `false` |
| `vorstand_elternteil` | `standard` | `["vorstand"]` | `true` |
| `vorstand_beisitzer` | `standard` | `["vorstand_beisitzer"]` | `false` |
| `kassierer` | `standard` | `["kassierer"]` | `false` |
| `trainer` | `standard` | `["trainer"]` | `false` |
| `trainer_elternteil` | `standard` | `["trainer"]` | `true` |
| `sportliche_leitung` | `standard` | `["sportliche_leitung"]` | `false` |
| `sportliche_leitung_elternteil` | `standard` | `["sportliche_leitung"]` | `true` |
| `spieler` | `standard` | `["spieler"]` | `false` |
| `elternteil` | `standard` | `[]` | `true` |
| `medien` | `standard` | `["medien"]` | `false` |

Kurzcodes für die folgenden Matrix-Tabellen: `a`=admin, `v`=vorstand, `ve`=vorstand_elternteil, `vb`=vorstand_beisitzer, `ka`=kassierer, `t`=trainer, `te`=trainer_elternteil, `s`=sportliche_leitung, `se`=sportliche_leitung_elternteil, `sp`=spieler, `e`=elternteil, `m`=medien.

`medien` (Migration `024`, Spielbericht-Freigabe) ist die einzige Vereinsfunktion mit einem eigenen Router-Tier, das keine andere Persona außer `vorstand` und `admin` erreicht; ohne eigene Persona wäre dieses Tier nur über den Admin-Bypass geprüft. Die Persona-Listen in `internal/permissions/personas_test.go` und `web/src/test/personas.ts` SHALL diese Tabelle spiegeln; jede Expected-Map der Tier-Matrix MUST für alle 12 Personas einen Eintrag tragen.

**Bewusst nicht abgedeckte Realfälle** (akzeptierte Test-Lücken):
- *Spieler-Trainer* (Trainer der auch spielt) — relevant für die Priorisierung der Duty-Soll-Berechnung; nicht in dieser Test-Matrix.
- *Spielendes Elternteil* (Mitglied, das selbst spielt UND Kind im Verein hat) — kein eigener Persona-Eintrag.
- *Kind ohne eigenes Konto* (Proxy-Konto) — hat keine eigene Identität; alle Zugriffe laufen über die Persona `elternteil`.

#### Scenario: Persona-Tokens werden konsistent für Backend- und Frontend-Tests verwendet
- **WHEN** ein Backend-Test eine Persona-ID anfragt und ein Frontend-Test dieselbe ID anfragt
- **THEN** liefern beide ein JWT mit identischen Werten für `role`, `club_functions`, `is_parent`

#### Scenario: Expected-Map ohne medien-Eintrag failt
- **WHEN** eine Expected-Map der Tier-Matrix keinen Eintrag für `medien` trägt
- **THEN** meldet `TestPermissionMatrix_Backend` die fehlende Persona für den betroffenen Endpoint

### Requirement: Public Endpoints sind ohne Auth zugänglich

Die Routen `POST /api/auth/login`, `POST /api/auth/refresh`, `POST /api/auth/logout`, `POST /api/auth/request-membership`, `POST /api/auth/register`, `GET /api/auth/token-info`, `POST /api/auth/forgot-password`, `POST /api/auth/reset-password`, `GET /api/profile/email/confirm`, `GET /api/files/{id}/download`, `GET /api/members/{id}/sepa-mandat/download` SHALL ohne Bearer-Token erreichbar sein und dürfen NICHT mit 401 antworten, nur weil kein Token vorliegt. `GET /api/uploads/*` ist NICHT mehr Teil dieser Liste und SHALL nicht ohne Authentifizierung ausgeliefert werden (siehe Anforderung „Upload-Auslieferung erfordert Authentifizierung").

#### Scenario: Login ohne Token
- **WHEN** ein Aufruf an `POST /api/auth/login` mit gültigem Body und ohne `Authorization`-Header gemacht wird
- **THEN** antwortet der Server NICHT mit 401 (200/400 je nach Body-Validität ist erlaubt)

#### Scenario: Unauthentifizierter Upload-Zugriff wird abgelehnt
- **WHEN** ein Aufruf an `GET /api/uploads/<datei>` ohne gültiges Refresh-Cookie gemacht wird
- **THEN** antwortet der Server mit 401 und liefert die Datei NICHT aus

### Requirement: Authenticated-Endpoints erfordern gültiges Bearer-Token

Alle Endpoints unterhalb der `auth.Middleware`-Group SHALL mit HTTP 401 antworten, wenn kein gültiger Access-Token vorliegt. Jede der 12 Personas mit gültigem Token SHALL diese Endpoints prinzipiell erreichen können — Filter auf Inhaltsebene werden in den jeweiligen Domain-Requirements behandelt.

Betroffene Endpoint-Gruppen (Auswahl, vollständige Liste im Matrix-Test):

- **Profil-Self:** `GET/PUT /api/profile/me`, `/vehicle`, `/account`, `/phones`, `/visibility`, `/reminder-preference`, `/absence-visibility`, `/notification-preferences`, `POST /api/profile/password`, `POST /api/profile/email`
- **Kind-Profil:** `GET/PUT /api/profile/kind/{memberId}/...`, `POST/DELETE /api/profile/kind/{memberId}/photo|phones`
- **Dashboard:** `GET /api/dashboard`
- **Dienste (Self-Service):** `GET /api/duty-board`, `POST/DELETE /api/duty-board/{slotId}/claim`, `GET /api/duty-types/{id}/instruction`, `GET /api/duty-slots`, `GET /api/duty-slots/{id}/assignments`, `GET /api/duty-fairness/rangliste`
- **Mitfahrgelegenheiten:** `GET/POST /api/mitfahrgelegenheiten`, `DELETE /api/mitfahrgelegenheiten/{id}`, `POST /api/mitfahrt-paarungen` (+ confirm/reject)
- **Push:** `GET /api/push/vapid-public-key`, `POST/DELETE /api/push/subscribe`
- **Dokumente:** `GET /api/folders`, `POST /api/folders`, `GET /api/folders/{id}/contents`, … (Pro-Folder-Permission filtert auf Inhaltsebene)
- **Games-Read + RSVP:** `GET /api/games`, `/games/{id}`, `/games/my`, `POST /api/games/{id}/respond`, `GET /api/games/{id}/responses|participants`, `POST /api/games/{id}/lineup`
- **Trainings-Read + RSVP:** `GET /api/training-sessions`, `/training-sessions/{id}`, `POST /api/training-sessions/{id}/respond`, `GET /api/training-sessions/{id}/attendances`
- **Saisonfenster:** `GET /api/seasons/active` (nur `id`/`name`/`start_date`/`end_date` der laufenden Saison; die vollständige Saisonliste `GET /api/seasons` bleibt gegated)
- **Teams:** `GET /api/teams`, `/teams/names`, `/teams/my`, `/teams/{id}/roster`
- **Übungsgruppen:** `GET /api/practice-groups/my` (Handler-Scope)
- **Videos:** `GET /api/videos`, `/videos/{id}`, `POST /api/videos`, `GET /api/videos/upload-eligible-games` (Berechtigung pro Team/Spiel im Handler)
- **Spielberichte (Autor):** `GET/POST /api/match-reports`, `GET /api/match-reports/{id}`, `PUT /api/match-reports/{id}`, `POST …/submit-for-review`, Bilder (Zustands- und Autor-Gate im Handler)
- **Chat:** alle `/api/chat/*`-Konversation- und Broadcast-Endpoints (außer `POST /api/chat/broadcasts` und `GET /api/chat/broadcast-targets` — beide hängen an der Ziel-Allowlist des Absenders, siehe `chat-broadcasts`)
- **Absences:** alle `/api/absences*`-Endpoints (Ownership-Check im Handler)

#### Scenario: 401 ohne Bearer-Token
- **WHEN** `GET /api/duty-board` ohne `Authorization`-Header aufgerufen wird
- **THEN** antwortet der Server mit 401

#### Scenario: Jede Persona darf Self-Service-Endpoint aufrufen
- **WHEN** eine beliebige Persona einen Aufruf an `GET /api/dashboard` mit gültigem Token sendet
- **THEN** antwortet der Server mit 200 (Inhaltsfilterung ist Sache des Handlers)

### Requirement: Trainer-und-Sportliche-Leitung-Gate

Die Routen unter `RequireClubFunction("trainer", "sportliche_leitung")` SHALL nur für `admin`, `trainer`, `trainer_elternteil`, `sportliche_leitung`, `sportliche_leitung_elternteil` mit 2xx antworten. Für `vorstand`, `vorstand_elternteil`, `vorstand_beisitzer`, `kassierer`, `spieler`, `elternteil`, `medien` SHALL der Server mit 403 antworten. Das Gate ist bewusst enger als das Vorstand-Trainer-sL-Gate: Trainingsbetrieb und Anwesenheit sind Trainer-Arbeit, der reine Vorstand hat hier keinen Zugang.

Betroffene Endpoints:

- Trainingsserien: `GET/POST /api/training-series`, `PUT/DELETE /api/training-series/{id}`, `GET/POST /api/training-series/{id}/unavailabilities`, `DELETE /api/training-series/{id}/unavailabilities/{uid}`
- Trainingseinheiten: `POST /api/training-sessions`, `PUT/DELETE /api/training-sessions/{id}`
- Anwesenheit: `POST /api/training-sessions/{id}/attendances`, `POST /api/games/{id}/attendances`, `DELETE /api/training-sessions/{id}/attendance-tracking`, `DELETE /api/games/{id}/attendance-tracking`, `POST/DELETE /api/training-sessions/{id}/attendance-excluded`, `POST/DELETE /api/games/{id}/attendance-excluded`
- Dienst-Erfüllung: `POST /api/duty-assignments/{id}/fulfill`, `POST /api/duty-assignments/{id}/cash-substitute`

Nicht mehr in diesem Gate (frühere Fehlzuordnung der Spec): `GET /api/venues` liegt im Vorstand-Trainer-sL-Gate; `GET/POST/DELETE /api/membership-requests…` und `POST /api/auth/invite` liegen im Vorstand-Gate; `POST/PUT/DELETE /api/duty-slots` liegen im Vorstand-Trainer-sL-Gate.

| a | v | ve | vb | ka | t | te | s | se | sp | e | m |
|---|---|---|---|---|---|---|---|---|---|---|---|
| ✅ | ❌ | ❌ | ❌ | ❌ | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |

#### Scenario: vorstand wird vom Trainer-Gate geblockt
- **WHEN** Persona `vorstand` `POST /api/training-sessions` aufruft
- **THEN** antwortet der Server mit 403

#### Scenario: trainer_elternteil wird vom Trainer-Gate durchgelassen
- **WHEN** Persona `trainer_elternteil` `POST /api/training-sessions` aufruft
- **THEN** antwortet der Server NICHT mit 403 (Erfolgs- oder Validation-Status, abhängig vom Body)

#### Scenario: medien wird vom Trainer-Gate geblockt
- **WHEN** Persona `medien` `POST /api/duty-assignments/{id}/fulfill` aufruft
- **THEN** antwortet der Server mit 403

### Requirement: Vorstand-Trainer-Sportliche-Leitung-Gate

Die Routen unter `RequireClubFunction("vorstand", "trainer", "sportliche_leitung")` SHALL für `admin`, `vorstand`, `vorstand_elternteil`, `trainer`, `trainer_elternteil`, `sportliche_leitung`, `sportliche_leitung_elternteil` mit 2xx antworten und für `vorstand_beisitzer`, `kassierer`, `spieler`, `elternteil`, `medien` mit 403. Das Tier beweist nur die Funktion; ob ein Trainer auf ein *fremdes* Objekt wirken darf, entscheiden Handler-Gates (siehe „Objektrechte werden mechanisch geprüft“ und §10).

Betroffene Endpoints (Mutationen):
- Spiele: `POST /api/games`, `PUT/DELETE /api/games/{id}`, `PUT /api/games/{id}/note`, `POST /api/games/{id}/regenerate`, `POST /api/games/regenerate-day`, `PUT /api/trainings/{id}/note`
- Spielorte: `GET/POST/DELETE /api/venues`, `PUT/DELETE /api/venues/{id}`, `POST /api/venues/import`
- Dienst-Slots: `POST /api/duty-slots`, `PUT/DELETE /api/duty-slots/{id}`
- Heimspieltag-Ausrichter: `POST /api/game-days/host/preview`, `POST /api/game-days/host/apply`
- Kader: `POST /api/kader`, `PUT/DELETE /api/kader/{id}`, `PATCH /api/kader/{id}/games-per-season`, `POST /api/kader/copy-from-season`, `POST /api/kader/auto-assign`
- Übungsgruppen: `POST /api/practice-groups`, `PUT/DELETE /api/practice-groups/{id}`
- Änderungsanträge: `POST /api/members/{id}/change-drafts/{draftId}/accept`, `DELETE /api/members/{id}/change-drafts/{draftId}`

Betroffene Endpoints (read-only):
- `GET /api/duty-types`, `GET /api/duty-templates`, `GET /api/duty-templates/{id}`, `GET /api/duty-templates/{id}/preview`
- `GET /api/duty-slots/export` (Dienst-CSV; liest nur, ohne Belegung und ohne Namen)
- `GET /api/kader`, `GET /api/kader/{id}`, `GET /api/kader/{id}/member-suggestions`, `GET /api/kader/{id}/extended-member-suggestions`
- `GET /api/practice-groups`, `GET /api/practice-groups/{id}`, `GET /api/practice-groups/{id}/member-suggestions`
- `GET /api/age-class-rules`, `GET /api/training-group-categories`

Nicht mehr in diesem Gate: `GET /api/seasons` hat ein eigenes Saisons-Lese-Gate (Kassierer eingeschlossen).

| a | v | ve | vb | ka | t | te | s | se | sp | e | m |
|---|---|---|---|---|---|---|---|---|---|---|---|
| ✅ | ✅ | ✅ | ❌ | ❌ | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |

#### Scenario: vorstand_beisitzer wird vom kombinierten Gate geblockt
- **WHEN** Persona `vorstand_beisitzer` `GET /api/kader` aufruft
- **THEN** antwortet der Server mit 403

#### Scenario: vorstand_elternteil hat denselben Zugriff wie vorstand
- **WHEN** Persona `vorstand_elternteil` `POST /api/games` mit gültigem Body aufruft
- **THEN** antwortet der Server NICHT mit 403

#### Scenario: vorstand liest die Spielorte
- **WHEN** Persona `vorstand` `GET /api/venues` aufruft
- **THEN** antwortet der Server NICHT mit 403

### Requirement: Vorstand-Gate

Die Routen unter `RequireClubFunction("vorstand")` SHALL ausschließlich für `admin`, `vorstand` und `vorstand_elternteil` mit 2xx antworten. Für alle anderen Personas (inklusive `vorstand_beisitzer`, `kassierer`, `trainer`, `trainer_elternteil`, `sportliche_leitung`, `sportliche_leitung_elternteil`, `spieler`, `elternteil`, `medien`) SHALL der Server mit 403 antworten.

Betroffene Endpoints:
- **Mitglieder (schreibend):** `POST /api/members`, `POST /api/members/import`, `PUT /api/members/{id}`, `PUT /api/members/{id}/status`, `PUT /api/members/{id}/user`, `DELETE /api/members/{id}`, `POST /api/members/{id}/proxy-account`, `POST /api/members/{id}/welcome-email`, `POST /api/users/{id}/create-member`, `POST/DELETE /api/family-links`
- **Nutzer:** `GET/POST /api/users`, `PUT/DELETE /api/users/{id}`, `PUT /api/users/{id}/role`, `PUT /api/users/{id}/recovery-email`
- **Einladungen und Beitrittsanfragen:** `POST /api/auth/invite`, `GET /api/invitations`, `DELETE /api/invitations/{id}`, `POST /api/invitations/{id}/send`, `PUT /api/invitations/{id}/member`, `POST /api/invitations/import-csv`, `GET /api/membership-requests`, `POST /api/membership-requests/{id}/approve|reject`, `DELETE /api/membership-requests/{id}`
- **Saisons und Teams:** `POST /api/seasons`, `PUT/DELETE /api/seasons/{id}`, `PUT /api/seasons/{id}/activate`, `PUT /api/seasons/{id}/duty-targets`, `POST /api/teams`, `PUT /api/teams/{id}`
- **Dienste:** `POST /api/duty-types`, `PUT/DELETE /api/duty-types/{id}`, `PUT /api/duty-types/{id}/instruction`, `POST /api/duty-templates`, `PUT/DELETE /api/duty-templates/{id}`, `POST /api/duty-slots/bulk-regen/preview|apply`
- **Heimspieltage:** `POST /api/ausrichter`, `PUT/DELETE /api/ausrichter/{id}`, `PUT /api/settings/bewirtung`
- **Spielimport:** `POST /api/games/import/h4a/preview|apply`
- **Stammdaten:** `PUT /api/age-class-rules/{ageClass}`, `POST /api/training-group-categories`, `DELETE /api/training-group-categories/{name}`, `POST /api/stammvereine`, `PUT/DELETE /api/stammvereine/{id}`
- **Uploads:** `POST/DELETE /api/upload/member-photo/{id}`

Nicht mehr in diesem Gate (frühere Fehlzuordnung der Spec): `GET /api/members/{id}`, `GET /api/members/{id}/parents`, `GET /api/members/export`, `GET/PUT /api/club` und `POST /api/upload/sepa-mandat/{id}` liegen im Vorstand-Kassierer-Gate; `GET /api/members` im Mitgliederlisten-Gate.

| a | v | ve | vb | ka | t | te | s | se | sp | e | m |
|---|---|---|---|---|---|---|---|---|---|---|---|
| ✅ | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |

#### Scenario: trainer wird vom Vorstand-Gate geblockt
- **WHEN** Persona `trainer` `GET /api/membership-requests` aufruft
- **THEN** antwortet der Server mit 403

#### Scenario: admin bypasst Vorstand-Gate
- **WHEN** Persona `admin` `POST /api/teams` mit gültigem Body aufruft
- **THEN** antwortet der Server NICHT mit 403

#### Scenario: kassierer wird vom Vorstand-Gate geblockt
- **WHEN** Persona `kassierer` `POST /api/members` aufruft
- **THEN** antwortet der Server mit 403

### Requirement: Admin-Only-Gate

Die Route `POST /api/impersonate/{id}` SHALL ausschließlich für Persona `admin` mit 2xx antworten. Für alle anderen 10 Personas SHALL der Server mit 403 antworten.

| a | v | ve | vb | ka | t | te | s | se | sp | e |
|---|---|---|---|---|---|---|---|---|---|---|
| ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |

#### Scenario: vorstand kann nicht impersonaten
- **WHEN** Persona `vorstand` `POST /api/impersonate/42` aufruft
- **THEN** antwortet der Server mit 403

---

### Requirement: Frontend-RoleRoute-Sichtbarkeit

Die folgenden Frontend-Routen aus `web/src/App.tsx` SHALL pro Persona entweder ihre Page rendern (✅) oder per `<Navigate to="/" replace>` umleiten (➜). `RoleRoute` prüft `admin` gegen `user.role`, alle anderen Werte gegen `user.clubFunctions`.

| Route | a | v | ve | vb | ka | t | te | s | se | sp | e | m |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| `/`, `/profil`, `/profil/kind/:memberId`, `/profil/anwesenheit`, `/dokumente(/...)`, `/dienste`, `/dienste/rangliste`, `/dienste/anleitung/:typeId`, `/mitfahrgelegenheiten`, `/kalender(/:gameId)`, `/termine(/:type/:id)`, `/mein-team`, `/chat`, `/spielberichte(/:id)`, `/videos(/:id)` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `/mitglieder(/:id)`, `/mitglieder/:memberId/sepa-mandat/anzeigen` | ✅ | ✅ | ✅ | ➜ | ✅ | ➜ | ➜ | ➜ | ➜ | ➜ | ➜ | ➜ |
| `/einstellungen`, `/beitragslauf`, `/tresor` | ✅ | ✅ | ✅ | ➜ | ✅ | ➜ | ➜ | ➜ | ➜ | ➜ | ➜ | ➜ |
| `/nutzer`, `/anfragen`, `/diensttypen`, `/dienstplan-vorlagen(/:id)`, `/veranstaltungsorte` | ✅ | ✅ | ✅ | ➜ | ➜ | ➜ | ➜ | ➜ | ➜ | ➜ | ➜ | ➜ |
| `/kader`, `/uebungsgruppen` | ✅ | ✅ | ✅ | ➜ | ➜ | ✅ | ✅ | ✅ | ✅ | ➜ | ➜ | ➜ |
| `/anwesenheit`, `/team/:id/anwesenheit` | ✅ | ➜ | ➜ | ➜ | ➜ | ✅ | ✅ | ✅ | ✅ | ➜ | ➜ | ➜ |
| `/trainingstagebuch`, `/team/:id/trainingstagebuch` | ✅ | ✅ | ✅ | ➜ | ➜ | ✅ | ✅ | ✅ | ✅ | ➜ | ➜ | ➜ |
| `/profil/trainingstagebuch` | ➜ | ➜ | ➜ | ➜ | ➜ | ➜ | ➜ | ➜ | ➜ | ✅ | ➜ | ➜ |
| `/spielberichte/pruefen` | ✅ | ✅ | ✅ | ➜ | ➜ | ➜ | ➜ | ➜ | ➜ | ➜ | ➜ | ✅ |
| `/wartung` | ✅ | ➜ | ➜ | ➜ | ➜ | ➜ | ➜ | ➜ | ➜ | ➜ | ➜ | ➜ |

`/profil/trainingstagebuch` ist die einzige Route, die `admin` ohne die Funktion `spieler` umleitet — das Tagebuch ist persönlich, es gibt keinen Admin-Bypass (siehe „Trainingstagebuch-Schreibzugriff“).

#### Scenario: spieler wird von /mitglieder umgeleitet
- **WHEN** Persona `spieler` mit initialer URL `/mitglieder` rendert
- **THEN** wird `<Navigate to="/" replace>` aktiv und die Dashboard-Page sichtbar

#### Scenario: trainer darf /kader sehen
- **WHEN** Persona `trainer` mit initialer URL `/kader` rendert
- **THEN** wird `AdminKaderPage` gerendert (kein Redirect)

#### Scenario: vorstand_elternteil hat dieselben RoleRoute-Rechte wie vorstand
- **WHEN** Persona `vorstand_elternteil` mit initialer URL `/mitglieder` rendert
- **THEN** wird `MembersPage` gerendert (kein Redirect)

#### Scenario: kassierer erreicht /mitglieder lesend
- **WHEN** Persona `kassierer` mit initialer URL `/mitglieder` rendert
- **THEN** wird `MembersPage` gerendert, ohne die an `manage_members` gebundenen Aktionen (Anlegen, Import)

#### Scenario: medien erreicht die Freigabe-Seite
- **WHEN** Persona `medien` mit initialer URL `/spielberichte/pruefen` rendert
- **THEN** wird die Freigabe-Seite gerendert (kein Redirect)

#### Scenario: admin wird vom eigenen Trainingstagebuch umgeleitet
- **WHEN** Persona `admin` mit initialer URL `/profil/trainingstagebuch` rendert
- **THEN** wird `<Navigate to="/" replace>` aktiv

### Requirement: Sidebar-Navigations-Items

Die Sichtbarkeit der Navigation SHALL serverseitig in `policy.NavFor` entschieden und über `GET /api/me` (`nav[]`) ausgeliefert werden; `AppShell` SHALL nur die vom Server gelieferten Ziele rendern. Pro Persona SHALL die Navigation die folgenden Items zeigen (alle nicht aufgelisteten Items sind ausgeblendet):

**Modul „Nutzer“**
- „Dashboard“ — alle Personas
- „Mein Profil“ — alle Personas mit `role != admin`; für `admin` nur, wenn ein eigener Mitglieds-Datensatz oder ein Kind (`is_parent`) existiert
- „Mein Trainingstagebuch“ — nur Vereinsfunktion `spieler` (kein Admin-Bypass)
- Kind-Sublinks (dynamisch) — Personas mit `children` aus `/api/profile/me`

**Modul „Spielbetrieb“**
- „Kalender“, „Termine“, „Videos“ — alle
- „Anwesenheit“ — `admin`, `trainer`, `sportliche_leitung`
- „Trainingstagebuch“ — `admin`, `trainer`, `sportliche_leitung`, `vorstand`

**Modul „Verein“**
- „Mein Team“, „Dokumente“, „Dienste“, „Dienst-Rangliste“, „Mitfahrten“, „Nachrichten“, „Spielberichte“ — alle
- „Berichte prüfen“ (AppShell-Beschriftung „Spielbericht prüfen“, Ziel `/spielberichte/pruefen`) — `admin`, `medien`, `vorstand`

**Modul „Verwaltung“**

| Item | a | v | ve | vb | ka | t | te | s | se | sp | e | m |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| Kader, Übungsgruppen | ✅ | ✅ | ✅ | – | – | ✅ | ✅ | ✅ | ✅ | – | – | – |
| Nutzerverwaltung | ✅ | ✅ | ✅ | – | – | – | – | – | – | – | – | – |
| Mitglieder | ✅ | ✅ | ✅ | – | ✅ | – | – | – | – | – | – | – |
| Diensttypen, Dienstplan-Vorlagen, Veranstaltungsorte | ✅ | ✅ | ✅ | – | – | – | – | – | – | – | – | – |
| Beitragslauf, Tresor, Einstellungen | ✅ | ✅ | ✅ | – | ✅ | – | – | – | – | – | – | – |
| Wartungsmodus | ✅ | – | – | – | – | – | – | – | – | – | – | – |

Wenn alle Items eines Moduls für eine Persona ausgeblendet sind, SHALL auch der Modul-Header nicht gerendert werden. `vorstand_beisitzer`, `spieler`, `elternteil` und `medien` sehen kein Verwaltungs-Modul.

#### Scenario: spieler sieht kein Verwaltungs-Modul
- **WHEN** Persona `spieler` rendert AppShell
- **THEN** ist weder das Modul-Header-Element „VERWALTUNG“ noch eines seiner Items im DOM

#### Scenario: trainer sieht Verwaltung nur mit Kader-Item
- **WHEN** Persona `trainer` rendert AppShell
- **THEN** ist das Modul „VERWALTUNG“ sichtbar mit genau den Kader-Items „Kader“ und „Übungsgruppen“, ohne Nutzer-, Dienst- oder Finanz-Items

#### Scenario: kassierer sieht die Finanz-Einträge
- **WHEN** Persona `kassierer` rendert AppShell
- **THEN** enthält das Modul „VERWALTUNG“ genau „Mitglieder“, „Beitragslauf“, „Tresor“ und „Einstellungen“

#### Scenario: medien sieht „Berichte prüfen“, aber kein Verwaltungs-Modul
- **WHEN** Persona `medien` rendert AppShell
- **THEN** ist der Eintrag mit Ziel `/spielberichte/pruefen` im DOM und das Modul „VERWALTUNG“ nicht

#### Scenario: admin sieht kein „Mein Profil"
- **WHEN** Persona `admin` ohne eigenen Mitglieds-Datensatz und ohne Kind rendert AppShell
- **THEN** ist der Nav-Item „Mein Profil“ nicht im DOM (`policy.NavFor` zeigt ihn dem Admin nur mit Mitglied oder Kind)

### Requirement: Inline-Gates auf Pages

Page-interne Sichtbarkeit SHALL aus den Capabilities aus `GET /api/me` (`hasCapability`) oder aus per-Objekt-`can.*`-Flags abgeleitet werden, nicht aus `role` oder `clubFunctions`. Die folgenden Gates gelten pro Persona (✅ sichtbar, – ausgeblendet):

| Page · Element | Capability | a | v | ve | vb | ka | t | te | s | se | sp | e | m |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| MembersPage / MemberDetailPage · Anlegen, Import, Verwaltungs-Tabs | `manage_members` | ✅ | ✅ | ✅ | – | – | – | – | – | – | – | – | – |
| KalenderPage / TerminePage · Termin bearbeiten, RSVP nach Cutoff | `manage_games` | ✅ | ✅ | ✅ | – | – | ✅ | ✅ | ✅ | ✅ | – | – | – |
| KalenderPage · H4A-Import | `import_games` | ✅ | ✅ | ✅ | – | – | – | – | – | – | – | – | – |
| KalenderPage · Massen-Dienstregeneration | `bulk_regen_duties` | ✅ | ✅ | ✅ | – | – | – | – | – | – | – | – | – |
| DutyPage · Slot bearbeiten/löschen; KalenderPage · Dienst-CSV | `manage_duties` | ✅ | ✅ | ✅ | – | – | ✅ | ✅ | ✅ | ✅ | – | – | – |
| KalenderPage · Training anlegen; TermineDetailPage · Trainer-Ansicht | `manage_trainings` | ✅ | – | – | – | – | ✅ | ✅ | ✅ | ✅ | – | – | – |
| ChatPage · Mitteilungs-Composer | `broadcast_messages` | ✅ | ✅ | ✅ | – | – | ✅ | ✅ | ✅ | ✅ | – | – | – |
| ChatPage · fremde Nachricht löschen | `moderate_chat` | ✅ | – | – | – | – | – | – | – | – | – | – | – |
| DocumentsPage · Root-Ordner anlegen | `create_root_folder` | ✅ | ✅ | ✅ | – | – | – | – | – | – | – | – | – |
| DeleteReasonFields · „ohne Benachrichtigung löschen“ | `suppress_event_notification` | ✅ | ✅ | ✅ | – | – | – | – | – | – | – | – | – |
| AdminSettingsPage · Tabs Verein / Beiträge | `manage_club` / `manage_fees` | ✅ | ✅ | ✅ | – | ✅ | – | – | – | – | – | – | – |
| AdminSettingsPage · Tabs Saisons, Altersklassen, Stammvereine / Heimspieltage | `manage_seasons` / `manage_duty_types` | ✅ | ✅ | ✅ | – | – | – | – | – | – | – | – | – |

Drei Gates folgen bewusst dem Teilnehmer- statt dem Verwaltungsmodell und lesen Claims statt Capabilities:

- **KalenderPage · „Abwesenheit eintragen“** = `spieler` ∨ `trainer` ∨ `is_parent` — Admin ist kein Sonderfall; `vorstand`, `vorstand_beisitzer`, `kassierer`, `sportliche_leitung`, `medien` bleiben ausgenommen, weil sie keine automatische Termin-Teilnahme haben.
- **MatchReportFormPage · Prüfen/Veröffentlichen** = `admin` ∨ `medien` ∨ `vorstand` — `medien` ist keine Capability (siehe `me-capabilities`), das Gate liest `clubFunctions`.

#### Scenario: spieler sieht keine Slot-Mutation-Actions auf DutyPage
- **WHEN** Persona `spieler` rendert die DutyPage
- **THEN** ist kein Element mit `data-testid="duty-slot-create"` (oder vergleichbares Marker-Pattern) im DOM

#### Scenario: elternteil sieht „Abwesenheit anlegen" im KalenderPage
- **WHEN** Persona `elternteil` rendert KalenderPage
- **THEN** ist der Button „Abwesenheit anlegen“ im DOM und enabled

#### Scenario: vorstand_elternteil sieht sowohl Vorstand-Actions als auch Abwesenheit-anlegen
- **WHEN** Persona `vorstand_elternteil` rendert KalenderPage
- **THEN** sind sowohl „Spiel anlegen“ (via `manage_games`) als auch „Abwesenheit anlegen“ (via `is_parent`) im DOM

#### Scenario: kassierer sieht in den Einstellungen nur Verein und Beiträge
- **WHEN** Persona `kassierer` rendert AdminSettingsPage
- **THEN** sind die Tabs „Verein“ und „Beiträge“ im DOM, „Saisons“ und „Heimspieltage“ nicht

### Requirement: Drift-Schutz

Wenn eine neue Backend-Route in `internal/app/router.go` registriert wird, SHALL der Test `TestPermissionMatrix_Backend` failen, solange kein Eintrag in der Matrix-Tabelle existiert.

Wenn eine neue Frontend-Route in `web/src/App.tsx` registriert wird, SHALL der Vitest-Smoke-Test failen, solange keine Erwartung pro Persona definiert ist.

#### Scenario: Neue Route ohne Matrix-Eintrag failt den Test
- **WHEN** ein Entwickler eine Route `r.Get("/api/new-resource", ...)` hinzufügt und `make test` läuft
- **THEN** failt `TestPermissionMatrix_Backend` mit einer klaren Fehlermeldung: „Route GET /api/new-resource ist nicht in der Permission-Matrix gepflegt"

---

### Requirement: Status quo — bekannte Designlöcher (§10)

Das System SHALL die folgenden Inkonsistenzen als bekannten Status quo führen. Sie werden in diesem Change dokumentiert, nicht behoben; jede braucht eine eigene Entscheidung. Erledigt und deshalb gestrichen sind die früheren Punkte „`vorstand_beisitzer` ohne Wirkung“ (er hat den Audience-Bypass im Dienst-Board und den Zugang zum Chat-Trainer-Zirkel), „`kassierer` ohne Wirkung“ (eigenes Router-Tier, `manage_club`/`manage_fees`, vier Nav-Einträge), „`/anfragen`-Mismatch“ (Backend liegt im Vorstand-Tier, das Frontend passt), „`DutyPage` schließt `vorstand` aus“ (Gate ist `manage_duties`) und „`MemberDatenschutzTab` schließt `admin` aus“ (Variable existiert nicht mehr).

1. **Vorstand liest Trainingstagebücher.** `trainingdiary.canReadMemberDiary`/`canSeeTeamDiary` und der Nav-Eintrag „Trainingstagebuch“ lassen `vorstand` durch; die Requirements „Trainingstagebuch-Lesezugriff folgt der Anwesenheitsstatistik“ in dieser Spec und `trainingstagebuch-sichtbarkeit` verlangen 403. Der Code-Kommentar begründet die Öffnung; die Specs sind nicht nachgezogen. Entscheidung offen: Spec öffnen oder Code schließen.
2. **Trainer sehen Videos aller Mannschaften** (`videos.CanViewVideo`); `video-management` bindet die Sicht an das Team. Bewusster Change (`video-sichtbarkeit-trainer-teamuebergreifend`), Spec nicht nachgezogen. `sportliche_leitung` darf umgekehrt überall hochladen, aber nur mit Team-Bezug ansehen.
3. **CSV-Importe für Hallen und Einladungen** liegen im v+t+sL- bzw. Vorstand-Tier; `venue-csv-import` und `csv-import` sagen admin-only. Ein reiner Trainer kann den Hallenbestand überschreiben.
4. **Übungsgruppen-Verwaltung** liegt im v+t+sL-Tier; `uebungsgruppen` sagt Vorstand + admin. Zusammen mit Punkt 5 kann jeder Trainer fremde Gruppen umbenennen oder leere löschen.
5. **Objektrechte-Lücken**: 14 `{id}`-Routen prüfen nur das Tier (`knownGaps` in `object_matrix_test.go`), darunter `PUT/DELETE /api/kader/{id}`, `PUT/DELETE /api/practice-groups/{id}`, `POST …/change-drafts/{draftId}/accept`, `PUT/DELETE /api/duty-slots/{id}`, `POST /api/games/{id}/regenerate`.
6. **Eigentümer und Eltern erreichen `GET /api/members/{id}` nicht** (Tier vorstand/kassierer); `sepa-mandat-upload` verspricht ihnen dort die Mandats-URL, der Handler-Zweig ist toter Code.
7. **Zwei Spiele-Sichtbarkeitsfilter** (`auth.GameVisibilityClause` mit Trainer-Bypass und erweitertem Kader; `policy.ScopeGamesQuery` mit Team-Scope für Trainer, ohne erweiterten Kader) existieren parallel.
8. **Frontend-Gates lesen `role`/`clubFunctions`** entgegen der Capability-Regel: alle `RoleRoute`s, Spielbericht-Freigabe (`medien`), VideoDetailPage, AdminUsersPage, MemberStammdatenTab.

#### Scenario: vorstand_beisitzer hat heute keinen Sondereffekt
- **WHEN** Persona `vorstand_beisitzer` einen Endpoint aufruft, der nicht öffentlich, Self-Service oder Dienst-Board ist
- **THEN** antwortet der Server wie für Persona `spieler`: die Funktion trägt kein eigenes Router-Tier, keine Capability und keinen Verwaltungs-Nav-Eintrag; ihre einzigen Mehrrechte sind der Audience-Bypass im Dienst-Board und der Zugang zum Chat-Trainer-Zirkel

#### Scenario: Offener Punkt bleibt sichtbar, bis er entschieden ist
- **WHEN** einer der Punkte 1–8 im Code behoben oder in der Domänen-Spec legitimiert wird
- **THEN** wird der Eintrag aus dieser Liste entfernt und die betroffene Domänen-Spec nachgezogen

### Requirement: Änderungsantrag-Routen erzwingen Mitglieds-Ownership

Die Self-Service-Routen `GET /api/members/{id}/change-drafts` und `POST /api/members/{id}/change-request` SHALL nur dann Mitgliedsdaten lesen oder schreiben, wenn der Aufrufer eine Beziehung zum Ziel-Mitglied `{id}` hat: Eigentümer (`member.user_id == claims.UserID`), Elternteil des Mitglieds (`family_links`), `admin`, `vorstand` oder `kassierer`. Für alle anderen Aufrufer SHALL der Server mit HTTP 403 antworten, BEVOR Antrags- oder Mitgliedsdaten (insbesondere der `old_value`-Snapshot) gelesen, zurückgegeben oder verändert werden.

| a | v | ve | vb | ka | t | te | s | se | sp | e |
|---|---|---|---|---|---|---|---|---|---|---|
| ✅ | ✅ (alle) | ✅ | ❌ (fremd) | ✅ (alle) | ❌ (fremd) | ❌ (fremd) | ❌ (fremd) | ❌ (fremd) | ❌ (fremd) | ✅ (eigenes Kind) |

> „fremd" = Member-ID gehört nicht zum Aufrufer/Kind. Eigentümer und Eltern erreichen ausschließlich das eigene bzw. das Kind-Mitglied; `vorstand`/`kassierer`/`admin` erreichen alle Mitglieder.

#### Scenario: Fremder Spieler liest Änderungsanträge eines anderen Mitglieds
- **WHEN** Persona `spieler` `GET /api/members/{id}/change-drafts` mit einer Member-ID aufruft, die nicht zu ihrem eigenen Account gehört
- **THEN** antwortet der Server mit 403 und liefert keine Antragsdaten und keinen `old_value`-Snapshot

#### Scenario: Eigentümer liest eigene Änderungsanträge
- **WHEN** der Eigentümer eines Mitglieds `GET /api/members/{id}/change-drafts` für die eigene Member-ID aufruft
- **THEN** antwortet der Server mit 200 und den eigenen Anträgen

#### Scenario: Elternteil liest Anträge des eigenen Kindes
- **WHEN** Persona `elternteil` `GET /api/members/{id}/change-drafts` für die Member-ID des eigenen Kindes aufruft
- **THEN** antwortet der Server NICHT mit 403

#### Scenario: Vorstand liest fremde Anträge
- **WHEN** Persona `vorstand` `GET /api/members/{id}/change-drafts` für ein beliebiges Mitglied aufruft
- **THEN** antwortet der Server mit 200

#### Scenario: Fremder Nutzer legt Änderungsantrag für anderes Mitglied an
- **WHEN** Persona `spieler` `POST /api/members/{id}/change-request` mit einer fremden Member-ID aufruft
- **THEN** antwortet der Server mit 403 und es wird kein Draft erzeugt, aktualisiert oder verdrängt

---

### Requirement: Bankdaten-Anträge nur durch Eigentümer oder Elternteil

Für `field_name='bankdaten'` SHALL `POST /api/members/{id}/change-request` ausschließlich Aufrufer akzeptieren, die Eigentümer oder Elternteil des Mitglieds sind (Selbstbedienungsmodell). Andere Aufrufer — auch `vorstand` und `kassierer`, deren Rolle die Genehmigung, nicht die Einreichung ist — SHALL mit HTTP 403 antworten. Dadurch kann kein fremder Aufrufer einen verschlüsselten Bankdaten-Envelope unter dem Namen eines anderen Mitglieds hinterlegen.

#### Scenario: Fremder unterschiebt Bankdaten-Envelope
- **WHEN** Persona `spieler` `POST /api/members/{id}/change-request` mit `field_name='bankdaten'` und einem Envelope `{bank_ciphertext, bank_dek_enc}` für eine fremde Member-ID sendet
- **THEN** antwortet der Server mit 403 und es wird kein `bankdaten`-Draft angelegt oder überschrieben

#### Scenario: Eigentümer reicht eigene Bankdaten ein
- **WHEN** der Eigentümer eines Mitglieds `POST .../change-request` mit `field_name='bankdaten'` und gültigem Envelope für die eigene Member-ID sendet
- **THEN** antwortet der Server mit 2xx und der `bankdaten`-Draft wird angelegt oder aktualisiert (UPSERT-Verhalten unverändert)

#### Scenario: Kassierer kann keinen Bankdaten-Antrag für ein Mitglied einreichen
- **WHEN** Persona `kassierer` `POST .../change-request` mit `field_name='bankdaten'` für ein fremdes Mitglied sendet
- **THEN** antwortet der Server mit 403 (Korrektur erfolgt über `PUT /api/members/{id}/bank-details`, nicht über den Antragsweg)

### Requirement: Upload-Auslieferung erfordert Authentifizierung

Das System SHALL Dateien unter `/api/uploads/*` nur an authentifizierte Aufrufer ausliefern. Da `<img>`-Requests keinen Bearer-Header senden, erfolgt die Authentifizierung über das HttpOnly-Refresh-Cookie (`auth.CookieMiddleware`, analog zu den SSE-Routen). Ohne gültiges Cookie SHALL der Server mit HTTP 401 antworten und die Datei NICHT ausliefern. Die Auslieferung SHALL `Referrer-Policy: no-referrer` und `Cache-Control: private, no-store` setzen, damit die (UUID-)URL nicht über Referrer oder Caches weiterleakt. Die UUID-Dateinamen bleiben als Defense-in-Depth erhalten.

> Bewusste Grenze: Es gibt keinen Pro-Foto-Sichtbarkeitscheck — jeder authentifizierte Nutzer kann ein Foto über seine (nicht erratbare) UUID-URL laden. Der behobene Befund war der *unauthentifizierte* Zugriff; per-Foto-Granularität wäre angesichts der bereits breiten Foto-Sichtbarkeit in Mitgliederlisten unverhältnismäßig.

#### Scenario: Authentifizierter Aufrufer erhält die Datei
- **WHEN** ein Aufrufer mit gültigem Refresh-Cookie `GET /api/uploads/<datei>` aufruft
- **THEN** wird die Datei mit 200 sowie `Referrer-Policy: no-referrer` und `Cache-Control: private, no-store` ausgeliefert

#### Scenario: Unauthentifizierter Zugriff wird abgelehnt
- **WHEN** `GET /api/uploads/<datei>` ohne gültiges Refresh-Cookie aufgerufen wird
- **THEN** antwortet der Server mit HTTP 401 und liefert die Datei NICHT aus

---

### Requirement: Trainingstagebuch-Schreibzugriff ist auf den Eigentümer beschränkt

Die Routen `POST /api/training-diary`, `PUT /api/training-diary/{id}`,
`DELETE /api/training-diary/{id}`, `POST /api/training-diary/{id}/proof` und
`DELETE /api/training-diary/{id}/proof` SHALL ausschließlich durch das Mitglied ausgeführt werden
können, dem der Eintrag gehört (`member.user_id == claims.UserID`). Keine Vereinsfunktion und
keine System-Rolle — auch nicht `admin`, `sportliche_leitung` oder der Trainer des Kaders —
verschafft Schreibzugriff auf ein fremdes Tagebuch.

`GET /api/training-diary` und `POST /api/training-diary` setzen zusätzlich voraus, dass der
Aufrufer überhaupt einen Mitglieds-Datensatz besitzt; Nutzer ohne verknüpftes Mitglied (etwa reine
Elternkonten) erhalten HTTP 403.

| a | v | ve | vb | ka | t | te | s | se | sp | e |
|---|---|---|---|---|---|---|---|---|---|---|
| ❌ (fremd) | ❌ | ❌ | ❌ | ❌ | ❌ (fremd) | ❌ | ❌ (fremd) | ❌ | ✅ (eigenes) | ❌ |

> Schreiben ist ausschließlich an die Eigentümerschaft gebunden. Eltern dürfen das Tagebuch ihres
> Kindes **lesen**, aber nicht befüllen — die Erfassung ist die Selbstauskunft des Spielers.

#### Scenario: Trainer ändert den Eintrag eines Spielers
- **WHEN** ein Trainer des Kaders `PUT /api/training-diary/{id}` auf den Eintrag eines seiner
  Spieler aufruft
- **THEN** antwortet der Server mit 403 und der Eintrag bleibt unverändert

#### Scenario: Trainer lädt einen Nachweis für einen Spieler hoch
- **WHEN** ein Trainer des Kaders `POST /api/training-diary/{id}/proof` auf einen fremden Eintrag
  aufruft
- **THEN** antwortet der Server mit 403 und es wird keine Datei geschrieben

#### Scenario: Nutzer ohne Mitglieds-Datensatz erfasst eine Einheit
- **WHEN** ein eingeloggter Nutzer ohne verknüpftes Mitglied `POST /api/training-diary` aufruft
- **THEN** antwortet der Server mit 403 und es wird kein Eintrag angelegt

#### Scenario: Unbekannte Eintrags-ID ist nicht per Statuscode enumerierbar
- **WHEN** ein beliebiger eingeloggter Nutzer `PUT /api/training-diary/{id}` mit einer nicht
  existierenden ID aufruft
- **THEN** antwortet der Server mit 404 und nicht mit 403

---

### Requirement: Trainingstagebuch-Lesezugriff folgt der Anwesenheitsstatistik

Die Routen `GET /api/members/{id}/training-diary`, `GET /api/training-diary/{id}/proof` und
`GET /api/teams/{id}/training-diary-stats` SHALL Lesezugriff ausschließlich gewähren an: das
Mitglied selbst, ein Elternteil über `family_links`, einen Trainer, der über
`trainer_memberships` × `kader` in der **aktiven Saison** eine Mannschaft betreut, in deren Stamm-
oder erweitertem Kader das Mitglied steht, sowie `sportliche_leitung` und `admin`.

`vorstand`, `vorstand_beisitzer` und `kassierer` SHALL **keinen** Zugriff erhalten — anders als bei
den Mitglieder- und Änderungsantrags-Routen begründet Vereinsverwaltung hier kein Leserecht. Das
Tagebuch ist persönlich.

Existiert die angefragte Member-ID nicht, SHALL der Server mit HTTP 403 antworten (nicht mit 500
und nicht mit 404) und dadurch nicht preisgeben, ob die ID vergeben ist.

| a | v | ve | vb | ka | t | te | s | se | sp | e |
|---|---|---|---|---|---|---|---|---|---|---|
| ✅ | ❌ | ❌ | ❌ | ❌ | ✅ (eigener Kader) | ✅ (eigener Kader) | ✅ (alle) | ✅ (alle) | ✅ (eigenes) | ✅ (eigenes Kind) |

#### Scenario: Mannschaftskamerad liest ein fremdes Tagebuch
- **WHEN** Persona `spieler` `GET /api/members/{id}/training-diary` mit der Member-ID eines
  Mitspielers aus demselben Kader aufruft
- **THEN** antwortet der Server mit 403

#### Scenario: Mannschaftskamerad ruft einen fremden Nachweis ab
- **WHEN** Persona `spieler` `GET /api/training-diary/{id}/proof` für den Eintrag eines
  Mitspielers aufruft
- **THEN** antwortet der Server mit 403 und liefert keine Bytes

#### Scenario: Vorstand liest ein fremdes Tagebuch
- **WHEN** Persona `vorstand` `GET /api/members/{id}/training-diary` für ein beliebiges Mitglied
  aufruft
- **THEN** antwortet der Server mit 403

#### Scenario: Trainer einer fremden Mannschaft
- **WHEN** ein Trainer `GET /api/teams/{id}/training-diary-stats` für eine Mannschaft aufruft, die
  er nicht betreut
- **THEN** antwortet der Server mit 403

#### Scenario: Kaderwechsel entzieht den Zugriff
- **WHEN** ein Mitglied aus dem Kader eines Trainers entfernt wird
- **THEN** antwortet der Server dem Trainer beim nächsten Abruf mit 403, ohne dass eine Nachpflege
  nötig ist

#### Scenario: Spieler ruft die Mannschaftsübersicht ab
- **WHEN** Persona `spieler` `GET /api/teams/{id}/training-diary-stats` für die eigene Mannschaft
  aufruft
- **THEN** antwortet der Server mit 403

---

### Requirement: Gelöschte Nachweise sind vom Fehlen unterscheidbar

`GET /api/training-diary/{id}/proof` SHALL für einen Berechtigten mit **HTTP 410** antworten, wenn
der Nachweis durch die Retention entfernt wurde (`proof_purged_at` gesetzt), und mit **HTTP 404**,
wenn nie einer hinterlegt war. Die Unterscheidung SHALL erst **nach** der Zugriffsprüfung erfolgen,
damit Unberechtigte daraus nichts über den Zustand des Eintrags ableiten können.

#### Scenario: Berechtigter ruft einen bereinigten Nachweis ab
- **WHEN** der Eigentümer den Nachweis eines Eintrags abruft, dessen `proof_purged_at` gesetzt ist
- **THEN** antwortet der Server mit 410

#### Scenario: Unberechtigter ruft einen bereinigten Nachweis ab
- **WHEN** ein fremder Spieler denselben Endpoint aufruft
- **THEN** antwortet der Server mit 403 und nicht mit 410

### Requirement: Objekt-Gates auf Detail- und Mutationsrouten

Routen, die ein einzelnes Objekt lesen oder verändern, MUST die Berechtigung am Objekt prüfen, nicht nur am Auth-Tier: `GET /api/training-sessions/{id}` folgt dem Kader-Zugriff; `POST /api/games/{id}/lineup` erlaubt nur Trainer eines beteiligten Teams (admin und sportliche Leitung vereinsweit); `POST /api/chat/conversations/{id}/read` und `DELETE /api/chat/conversations/{id}/members/me` verlangen aktive Mitgliedschaft; `POST /api/duty-assignments/{id}/fulfill` und `/cash-substitute` verlangen admin, trainer oder sportliche Leitung (das bestehende Recht, nun auch im Handler geprüft) und MUST mit 404 antworten, wenn die Zuweisung nicht existiert.

#### Scenario: Trainer eines fremden Teams speichert keine Aufstellung
- **WHEN** ein Trainer, dessen Kader an dem Spiel nicht beteiligt ist, `POST /api/games/{id}/lineup` aufruft
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Nicht-Mitglied kann keine Lesebestätigung setzen
- **WHEN** ein Nutzer ohne aktive Mitgliedschaft `POST /api/chat/conversations/{id}/read` aufruft
- **THEN** antwortet der Server mit HTTP 403 und schreibt keine `message_reads`-Zeile

#### Scenario: Nicht-Mitglied hinterlässt keine System-Nachricht
- **WHEN** ein Nutzer ohne aktive Mitgliedschaft `DELETE /api/chat/conversations/{id}/members/me` aufruft
- **THEN** antwortet der Server mit HTTP 403 und es entsteht keine Nachricht

#### Scenario: Spieler sieht fremde Trainingseinheit nicht
- **WHEN** ein Spieler ohne Kader-Zugriff `GET /api/training-sessions/{id}` aufruft
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Dienst-Erfüllung nur durch Berechtigte und nur für existierende Zuweisungen
- **WHEN** ein Spieler `POST /api/duty-assignments/{id}/fulfill` aufruft
- **THEN** antwortet der Server mit HTTP 403
- **WHEN** ein Trainer dieselbe Route für eine nicht existierende ID aufruft
- **THEN** antwortet der Server mit HTTP 404

### Requirement: Objektrechte werden mechanisch geprüft

Für jede Route mit einem Objekt-Parameter (`{id}`) MUST ein Test existieren, der ein fremdes Objekt anlegt und als nicht berechtigter Nutzer 403 oder 404 erwartet. Routen ohne einen solchen Test MUST den Test fehlschlagen lassen, sofern sie nicht mit Begründung in einer Allowlist bewusst offener Routen stehen. Ein verwaister Allowlist-Eintrag MUST ebenfalls fehlschlagen.

#### Scenario: Neue Objekt-Route ohne Fixture
- **WHEN** eine Route `GET /api/foo/{id}` hinzukommt, ohne Fixture-Erzeuger und ohne Allowlist-Eintrag
- **THEN** schlägt die Objekt-Matrix fehl und nennt die Route

#### Scenario: Fremdes Objekt bleibt verborgen
- **WHEN** Nutzer A ein Objekt von Nutzer B über eine `{id}`-Route anspricht
- **THEN** antwortet der Server mit 403 oder 404

### Requirement: Dienst-Rangliste im Authenticated-Tier mit Handler-Scope

`GET /api/duty-fairness/rangliste` SHALL im Authenticated-Tier liegen: jede
eingeloggte Persona erreicht die Route, ohne Token antwortet sie mit HTTP 401. Die
Einschränkung auf eigene Teams (HTTP 403 für ein Team ohne eigene Kader- oder
`family_links`-Verbindung) und die Namens-Maskierung SHALL der Handler selbst
durchsetzen — für die System-Rolle `admin` und die Vereinsfunktion `vorstand`
entfällt beides.

Der Sidebar-Eintrag „Dienst-Rangliste" (`/dienste/rangliste`) SHALL für jede
eingeloggte Persona sichtbar sein.

#### Scenario: Eingeloggte Persona erreicht die Rangliste
- **WHEN** eine beliebige Persona `GET /api/duty-fairness/rangliste` mit gültigem
  Token aufruft
- **THEN** antwortet das System nicht mit 401

#### Scenario: Aufruf ohne Token
- **WHEN** `GET /api/duty-fairness/rangliste` ohne Token aufgerufen wird
- **THEN** antwortet das System mit HTTP 401

#### Scenario: Sidebar zeigt die Rangliste für alle
- **WHEN** eine beliebige eingeloggte Persona die Navigation lädt
- **THEN** enthält sie den Eintrag „Dienst-Rangliste" mit Ziel `/dienste/rangliste`

### Requirement: Übungsgruppen-Sichtbarkeit im Authenticated-Tier mit Handler-Scope

`GET /api/practice-groups/my` SHALL im Authenticated-Tier liegen: jede eingeloggte
Persona erreicht die Route, ohne Token antwortet sie mit HTTP 401. Die Einschränkung
auf die eigenen Übungsgruppen (Trainer via `kader_trainers`, Spieler/Eltern via
`kader_members`/`kader_extended_members`/`family_links`) SHALL der Handler selbst
durchsetzen — für die System-Rolle `admin` und die Vereinsfunktionen `vorstand` und
`sportliche_leitung` entfällt die Einschränkung, sie sehen alle Übungsgruppen der
aktiven Saison.

#### Scenario: Eingeloggte Persona erreicht die Route
- **WHEN** eine beliebige Persona `GET /api/practice-groups/my` mit gültigem Token
  aufruft
- **THEN** antwortet das System nicht mit 401

#### Scenario: Aufruf ohne Token
- **WHEN** `GET /api/practice-groups/my` ohne Token aufgerufen wird
- **THEN** antwortet das System mit HTTP 401

### Requirement: Vorstand-Kassierer-Gate

Die Routen unter `RequireClubFunction("vorstand", "kassierer")` SHALL für `admin`, `vorstand`, `vorstand_elternteil` und `kassierer` mit 2xx antworten und für alle übrigen Personas mit 403. Das Tier bündelt die Finanz- und Mitglieder-Leseflächen der Finance-Gruppe; Zero-Knowledge-Bankdaten werden dabei nur als Envelope transportiert.

Betroffene Endpoints:
- Mitglieder (lesend): `GET /api/members/{id}`, `GET /api/members/{id}/parents`, `GET /api/members/export`
- Bankdaten: `PUT /api/members/{id}/bank-details`, `POST /api/upload/sepa-mandat/{id}`
- Verein: `GET/PUT /api/club`
- Beiträge: `GET/POST /api/fee-rates`, `DELETE /api/fee-rates/{id}`, `GET /api/fee-run/preview`, `POST /api/fee-run/export-data`, `POST /api/fee-run/confirm`, `GET /api/fee-run/protocol`
- Tresor: `GET/PUT /api/admin/encryption-config`, `PUT /api/admin/rotate-encryption`

| a | v | ve | vb | ka | t | te | s | se | sp | e | m |
|---|---|---|---|---|---|---|---|---|---|---|---|
| ✅ | ✅ | ✅ | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |

#### Scenario: kassierer liest ein Mitglied
- **WHEN** Persona `kassierer` `GET /api/members/{id}` aufruft
- **THEN** antwortet der Server NICHT mit 403

#### Scenario: vorstand_beisitzer wird geblockt
- **WHEN** Persona `vorstand_beisitzer` `GET /api/club` aufruft
- **THEN** antwortet der Server mit 403

#### Scenario: trainer erreicht den Beitragslauf nicht
- **WHEN** Persona `trainer` `GET /api/fee-run/preview` aufruft
- **THEN** antwortet der Server mit 403

### Requirement: Mitgliederlisten-Gate

`GET /api/members` SHALL unter `RequireClubFunction("vorstand", "kassierer", "trainer", "sportliche_leitung")` liegen: Mitgliederverwaltung und Kader-/Trainersuche teilen sich die Route. Der Umfang der Liste SHALL der Handler über `policy.ScopeMembersQuery` bestimmen: `admin`, `vorstand`, `kassierer` und `sportliche_leitung` sehen alle Mitglieder, `trainer` nur Mitglieder der Kader, die er in der aktiven Saison trainiert. Alle übrigen Personas SHALL 403 erhalten.

| a | v | ve | vb | ka | t | te | s | se | sp | e | m |
|---|---|---|---|---|---|---|---|---|---|---|---|
| ✅ | ✅ | ✅ | ❌ | ✅ | ✅ (Team) | ✅ (Team) | ✅ | ✅ | ❌ | ❌ | ❌ |

#### Scenario: trainer liest die Liste seiner Kader
- **WHEN** Persona `trainer` `GET /api/members` aufruft
- **THEN** antwortet der Server NICHT mit 403 und liefert nur Mitglieder seiner Kader

#### Scenario: spieler wird geblockt
- **WHEN** Persona `spieler` `GET /api/members` aufruft
- **THEN** antwortet der Server mit 403

### Requirement: Saisons-Lese-Gate

`GET /api/seasons` SHALL unter `RequireClubFunction("vorstand", "trainer", "sportliche_leitung", "kassierer")` liegen: Kaderpflege (Vorstand, Trainer, sportliche Leitung) und Beitragslauf (Kassierer) brauchen die Saisonliste. Die aktive Saison allein (`GET /api/seasons/active`) bleibt im Authenticated-Tier.

| a | v | ve | vb | ka | t | te | s | se | sp | e | m |
|---|---|---|---|---|---|---|---|---|---|---|---|
| ✅ | ✅ | ✅ | ❌ | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |

#### Scenario: kassierer liest die Saisonliste
- **WHEN** Persona `kassierer` `GET /api/seasons` aufruft
- **THEN** antwortet der Server NICHT mit 403

#### Scenario: elternteil liest nur die aktive Saison
- **WHEN** Persona `elternteil` `GET /api/seasons` und danach `GET /api/seasons/active` aufruft
- **THEN** antwortet der Server zuerst mit 403 und danach NICHT mit 401/403

### Requirement: Spielbericht-Freigeber-Gate

Die Routen unter `RequireClubFunction("medien", "vorstand")` SHALL für `admin`, `vorstand`, `vorstand_elternteil` und `medien` mit 2xx antworten und für alle übrigen Personas mit 403. Betroffen: `GET /api/match-reports/pending`, `POST /api/match-reports/{id}/publish`. Die zustandsabhängige Freigabe (nur `pending_review`/`publish_failed`) prüft der Handler zusätzlich.

| a | v | ve | vb | ka | t | te | s | se | sp | e | m |
|---|---|---|---|---|---|---|---|---|---|---|---|
| ✅ | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ |

#### Scenario: medien erreicht die Freigabe-Liste
- **WHEN** Persona `medien` `GET /api/match-reports/pending` aufruft
- **THEN** antwortet der Server NICHT mit 403

#### Scenario: trainer wird vom Freigeber-Gate geblockt
- **WHEN** Persona `trainer` `POST /api/match-reports/{id}/publish` aufruft
- **THEN** antwortet der Server mit 403

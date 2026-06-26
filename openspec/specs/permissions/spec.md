# permissions Specification

## Purpose

Definiert die verbindliche Autorisierungs-Matrix von TeamWERK: welche Persona
(Kombination aus System-Rolle, Vereinsfunktionen und Eltern-Status) welche
Backend-Routen erreichen darf und welche Frontend-Routen, Navigations-Items und
Page-internen Aktionen sichtbar sind. Dient als Quelle der Wahrheit für die
mechanischen Drift-Tests (`TestPermissionMatrix_Backend`, Vitest-Smoke-Tests).

## Requirements

### Requirement: Persona-Definition

Das System SHALL die folgenden 11 Personas als Test-Fixtures bereitstellen. Sie decken alle praktisch relevanten Kombinationen aus System-Rolle, Vereinsfunktion(en) und Eltern-Status ab, mit besonderem Fokus auf den im Verein häufigen Fall, dass funktionsführende Mitglieder gleichzeitig Eltern sind.

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

Kurzcodes für die folgenden Matrix-Tabellen: `a`=admin, `v`=vorstand, `ve`=vorstand_elternteil, `vb`=vorstand_beisitzer, `ka`=kassierer, `t`=trainer, `te`=trainer_elternteil, `s`=sportliche_leitung, `se`=sportliche_leitung_elternteil, `sp`=spieler, `e`=elternteil.

**Bewusst nicht abgedeckte Realfälle** (akzeptierte Test-Lücken):
- *Spieler-Trainer* (Trainer der auch spielt) — relevant für die Priorisierung der Duty-Soll-Berechnung; nicht in dieser Test-Matrix.
- *Spielendes Elternteil* (Mitglied, das selbst spielt UND Kind im Verein hat) — kein eigener Persona-Eintrag.

#### Scenario: Persona-Tokens werden konsistent für Backend- und Frontend-Tests verwendet
- **WHEN** ein Backend-Test eine Persona-ID anfragt und ein Frontend-Test dieselbe ID anfragt
- **THEN** liefern beide ein JWT mit identischen Werten für `role`, `club_functions`, `is_parent`

---

### Requirement: Public Endpoints sind ohne Auth zugänglich

Die Routen `POST /api/auth/login`, `POST /api/auth/refresh`, `POST /api/auth/logout`, `POST /api/auth/request-membership`, `POST /api/auth/register`, `GET /api/auth/token-info`, `POST /api/auth/forgot-password`, `POST /api/auth/reset-password`, `GET /api/profile/email/confirm`, `GET /api/uploads/*`, `GET /api/files/{id}/download`, `GET /api/members/{id}/sepa-mandat/download` SHALL ohne Bearer-Token erreichbar sein und dürfen NICHT mit 401 antworten, nur weil kein Token vorliegt.

#### Scenario: Login ohne Token
- **WHEN** ein Aufruf an `POST /api/auth/login` mit gültigem Body und ohne `Authorization`-Header gemacht wird
- **THEN** antwortet der Server NICHT mit 401 (200/400 je nach Body-Validität ist erlaubt)

---

### Requirement: Authenticated-Endpoints erfordern gültiges Bearer-Token

Alle Endpoints unterhalb der `auth.Middleware`-Group SHALL mit HTTP 401 antworten, wenn kein gültiger Access-Token vorliegt. Jede der 11 Personas mit gültigem Token SHALL diese Endpoints prinzipiell erreichen können — Filter auf Inhaltsebene werden in den jeweiligen Domain-Requirements behandelt.

Betroffene Endpoint-Gruppen (Auswahl, vollständige Liste im Matrix-Test):

- **Profil-Self:** `GET/PUT /api/profile/me`, `/vehicle`, `/account`, `/phones`, `/visibility`, `/reminder-preference`, `/absence-visibility`, `/notification-preferences`, `POST /api/profile/password`, `POST /api/profile/email`
- **Kind-Profil:** `GET/PUT /api/profile/kind/{memberId}/...`, `POST/DELETE /api/profile/kind/{memberId}/photo|phones`
- **Dashboard:** `GET /api/dashboard`
- **Dienste (Self-Service):** `GET /api/duty-board`, `POST/DELETE /api/duty-board/{slotId}/claim`, `GET /api/duty-accounts`, `GET /api/duty-slots`, `GET /api/duty-slots/{id}/assignments`
- **Mitfahrgelegenheiten:** `GET/POST /api/mitfahrgelegenheiten`, `DELETE /api/mitfahrgelegenheiten/{id}`, `POST /api/mitfahrt-paarungen` (+ confirm/reject)
- **Push:** `GET /api/push/vapid-public-key`, `POST/DELETE /api/push/subscribe`
- **Dokumente:** `GET /api/folders`, `POST /api/folders`, `GET /api/folders/{id}/contents`, … (Pro-Folder-Permission filtert auf Inhaltsebene)
- **Games-Read + RSVP:** `GET /api/games`, `/games/{id}`, `/games/my`, `POST /api/games/{id}/respond`, `GET /api/games/{id}/responses|participants`, `POST /api/games/{id}/lineup`
- **Trainings-Read + RSVP:** `GET /api/training-sessions`, `/training-sessions/{id}`, `POST /api/training-sessions/{id}/respond`, `GET /api/training-sessions/{id}/attendances`
- **Teams:** `GET /api/teams`, `/teams/names`, `/teams/my`, `/teams/{id}/roster`
- **Chat:** alle `/api/chat/*`-Konversation- und Broadcast-Endpoints (außer `POST /api/chat/broadcasts`, siehe Broadcast-Requirement)
- **Absences:** alle `/api/absences*`-Endpoints (Ownership-Check im Handler)

#### Scenario: 401 ohne Bearer-Token
- **WHEN** `GET /api/duty-board` ohne `Authorization`-Header aufgerufen wird
- **THEN** antwortet der Server mit 401

#### Scenario: Jede Persona darf Self-Service-Endpoint aufrufen
- **WHEN** eine beliebige Persona einen Aufruf an `GET /api/dashboard` mit gültigem Token sendet
- **THEN** antwortet der Server mit 200 (Inhaltsfilterung ist Sache des Handlers)

---

### Requirement: Trainer-und-Sportliche-Leitung-Gate

Die Routen unter `RequireClubFunction("trainer", "sportliche_leitung")` SHALL nur für `admin`, `trainer`, `trainer_elternteil`, `sportliche_leitung`, `sportliche_leitung_elternteil` mit 2xx antworten. Für `vorstand`, `vorstand_elternteil`, `vorstand_beisitzer`, `kassierer`, `spieler`, `elternteil` SHALL der Server mit 403 antworten.

Betroffene Endpoints:

- `GET /api/venues`
- `GET/POST/PUT/DELETE /api/training-series(/{id})`
- `POST/PUT/DELETE /api/training-sessions(/{id})`
- `POST /api/training-sessions/{id}/attendances`
- `POST/PUT/DELETE /api/duty-slots(/{id})`
- `POST /api/duty-assignments/{id}/fulfill`
- `POST /api/duty-assignments/{id}/cash-substitute`
- `GET/POST/DELETE /api/membership-requests(/{id}/approve|reject)`
- `POST /api/auth/invite`

| a | v | ve | vb | ka | t | te | s | se | sp | e |
|---|---|---|---|---|---|---|---|---|---|---|
| ✅ | ❌ | ❌ | ❌ | ❌ | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ |

#### Scenario: vorstand wird vom Trainer-Gate geblockt
- **WHEN** Persona `vorstand` `POST /api/duty-slots` aufruft
- **THEN** antwortet der Server mit 403

#### Scenario: trainer_elternteil wird vom Trainer-Gate durchgelassen
- **WHEN** Persona `trainer_elternteil` `POST /api/training-sessions` aufruft
- **THEN** antwortet der Server NICHT mit 403 (Erfolgs- oder Validation-Status, abhängig vom Body)

---

### Requirement: Vorstand-Trainer-Sportliche-Leitung-Gate

Die Routen unter `RequireClubFunction("vorstand", "trainer", "sportliche_leitung")` SHALL für `admin`, `vorstand`, `vorstand_elternteil`, `trainer`, `trainer_elternteil`, `sportliche_leitung`, `sportliche_leitung_elternteil` mit 2xx antworten und für `vorstand_beisitzer`, `kassierer`, `spieler`, `elternteil` mit 403.

Betroffene Endpoints (write):
- `POST /api/venues`, `POST /api/venues/import`, `DELETE /api/venues`, `PUT/DELETE /api/venues/{id}`
- `POST/PUT/DELETE /api/games(/{id})`
- `POST /api/games/{id}/regenerate`, `POST /api/games/regenerate-day`
- `POST /api/members/{id}/change-drafts/{draftId}/accept`
- `DELETE /api/members/{id}/change-drafts/{draftId}`
- `GET /api/age-class-rules`

Betroffene Endpoints (read-only):
- `GET /api/duty-types`, `GET /api/duty-templates(/{id})`, `GET /api/duty-templates/{id}/preview`
- `GET /api/seasons`
- `GET /api/kader`, `POST /api/kader`, `GET/PUT/DELETE /api/kader/{id}`
- `GET /api/kader/{id}/member-suggestions`, `GET /api/kader/{id}/extended-member-suggestions`
- `PATCH /api/kader/{id}/games-per-season`
- `POST /api/kader/copy-from-season`, `POST /api/kader/auto-assign`

| a | v | ve | vb | ka | t | te | s | se | sp | e |
|---|---|---|---|---|---|---|---|---|---|---|
| ✅ | ✅ | ✅ | ❌ | ❌ | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ |

#### Scenario: vorstand_beisitzer wird vom kombinierten Gate geblockt
- **WHEN** Persona `vorstand_beisitzer` `GET /api/kader` aufruft
- **THEN** antwortet der Server mit 403 (Designloch — siehe §10)

#### Scenario: vorstand_elternteil hat denselben Zugriff wie vorstand
- **WHEN** Persona `vorstand_elternteil` `POST /api/games` mit gültigem Body aufruft
- **THEN** antwortet der Server NICHT mit 403

---

### Requirement: Vorstand-Gate

Die Routen unter `RequireClubFunction("vorstand")` SHALL ausschließlich für `admin`, `vorstand` und `vorstand_elternteil` mit 2xx antworten. Für alle anderen Personas (inklusive `vorstand_beisitzer`, `kassierer`, `trainer`, `trainer_elternteil`, `sportliche_leitung`, `sportliche_leitung_elternteil`, `spieler`, `elternteil`) SHALL der Server mit 403 antworten.

Betroffene Endpoints (Auswahl):
- **Mitglieder:** `GET /api/members(/export|/{id})`, `POST /api/members(/import)`, `PUT /api/members/{id}(/status|/user)`, `DELETE /api/members/{id}`, `POST /api/members/{id}/proxy-account|welcome-email`, `GET /api/members/{id}/parents`, `POST /api/users/{id}/create-member`, `POST/DELETE /api/family-links`
- **Verein/Saisons:** `GET/PUT /api/club`, `POST/PUT/DELETE /api/seasons(/{id})`, `PUT /api/seasons/{id}/activate|/duty-targets`
- **Teams:** `POST /api/teams`, `PUT /api/teams/{id}`
- **Nutzer:** `GET/POST/PUT/DELETE /api/users(/{id})`, `PUT /api/users/{id}/role`
- **Einladungen:** `GET/DELETE /api/invitations(/{id})`, `POST /api/invitations/{id}/send`, `POST /api/invitations/import-csv`, `PUT /api/invitations/{id}/member`
- **Diensttypen + Konten:** `POST/PUT/DELETE /api/duty-types(/{id})`, `GET /api/duty-accounts/export`
- **Templates:** `POST/PUT/DELETE /api/duty-templates(/{id})`
- **Uploads:** `POST/DELETE /api/upload/member-photo/{id}`, `POST /api/upload/sepa-mandat/{id}`
- **Config:** `PUT /api/age-class-rules/{ageClass}`

| a | v | ve | vb | ka | t | te | s | se | sp | e |
|---|---|---|---|---|---|---|---|---|---|---|
| ✅ | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |

#### Scenario: trainer wird vom Vorstand-Gate geblockt
- **WHEN** Persona `trainer` `GET /api/members` aufruft
- **THEN** antwortet der Server mit 403

#### Scenario: admin bypasst Vorstand-Gate
- **WHEN** Persona `admin` `POST /api/teams` mit gültigem Body aufruft
- **THEN** antwortet der Server NICHT mit 403

---

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

Die folgenden Frontend-Routen aus `web/src/App.tsx` SHALL pro Persona entweder ihre Page rendern (✅) oder per `<Navigate to="/" replace>` umleiten (➜).

| Route | a | v | ve | vb | ka | t | te | s | se | sp | e |
|---|---|---|---|---|---|---|---|---|---|---|---|
| `/` (Dashboard) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `/profil` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `/profil/kind/:memberId` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `/dokumente(/...)` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `/dienste` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `/mitfahrgelegenheiten` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `/kalender(/:gameId)` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `/termine(/:type/:id)` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `/mein-team` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `/chat` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `/mitglieder(/:id)` | ✅ | ✅ | ✅ | ➜ | ➜ | ➜ | ➜ | ➜ | ➜ | ➜ | ➜ |
| `/nutzer` | ✅ | ✅ | ✅ | ➜ | ➜ | ➜ | ➜ | ➜ | ➜ | ➜ | ➜ |
| `/anfragen` | ✅ | ✅ | ✅ | ➜ | ➜ | ➜ | ➜ | ➜ | ➜ | ➜ | ➜ |
| `/einstellungen` | ✅ | ✅ | ✅ | ➜ | ➜ | ➜ | ➜ | ➜ | ➜ | ➜ | ➜ |
| `/kader` | ✅ | ✅ | ✅ | ➜ | ➜ | ✅ | ✅ | ✅ | ✅ | ➜ | ➜ |
| `/diensttypen` | ✅ | ✅ | ✅ | ➜ | ➜ | ➜ | ➜ | ➜ | ➜ | ➜ | ➜ |
| `/dienstplan-vorlagen(/:id)` | ✅ | ✅ | ✅ | ➜ | ➜ | ➜ | ➜ | ➜ | ➜ | ➜ | ➜ |
| `/veranstaltungsorte` | ✅ | ✅ | ✅ | ➜ | ➜ | ➜ | ➜ | ➜ | ➜ | ➜ | ➜ |

#### Scenario: spieler wird von /mitglieder umgeleitet
- **WHEN** Persona `spieler` mit initialer URL `/mitglieder` rendert
- **THEN** wird `<Navigate to="/" replace>` aktiv und die Dashboard-Page sichtbar

#### Scenario: trainer darf /kader sehen
- **WHEN** Persona `trainer` mit initialer URL `/kader` rendert
- **THEN** wird `AdminKaderPage` gerendert (kein Redirect)

#### Scenario: vorstand_elternteil hat dieselben RoleRoute-Rechte wie vorstand
- **WHEN** Persona `vorstand_elternteil` mit initialer URL `/mitglieder` rendert
- **THEN** wird `MembersPage` gerendert (kein Redirect)

---

### Requirement: Sidebar-Navigations-Items

`AppShell.navModules` SHALL pro Persona die folgenden Items zeigen (alle nicht-aufgelisteten Items sind ausgeblendet):

**Modul „Nutzer"**
- „Dashboard" — alle Personas
- „Mein Profil" — alle Personas **außer** `admin`
- Kind-Sublinks (dynamisch) — Personas mit `children` aus `/api/profile/me` (typisch Eltern-Varianten)

**Modul „Spielbetrieb"**
- „Kalender" — alle
- „Termine" — alle

**Modul „Verein"**
- „Mein Team", „Dokumente", „Dienste", „Mitfahrten", „Nachrichten" — alle

**Modul „Verwaltung"**

| Item | a | v | ve | vb | ka | t | te | s | se | sp | e |
|---|---|---|---|---|---|---|---|---|---|---|---|
| Nutzerverwaltung | ✅ | ✅ | ✅ | – | – | – | – | – | – | – | – |
| Mitglieder | ✅ | ✅ | ✅ | – | – | – | – | – | – | – | – |
| Kader | ✅ | ✅ | ✅ | – | – | ✅ | ✅ | ✅ | ✅ | – | – |
| Diensttypen | ✅ | ✅ | ✅ | – | – | – | – | – | – | – | – |
| Dienstplan-Vorlagen | ✅ | ✅ | ✅ | – | – | – | – | – | – | – | – |
| Veranstaltungsorte | ✅ | ✅ | ✅ | – | – | – | – | – | – | – | – |
| Einstellungen | ✅ | ✅ | ✅ | – | – | – | – | – | – | – | – |

Wenn alle Items eines Moduls für eine Persona ausgeblendet sind, SHALL auch der Modul-Header (z. B. „VERWALTUNG") nicht gerendert werden.

#### Scenario: spieler sieht kein Verwaltungs-Modul
- **WHEN** Persona `spieler` rendert AppShell
- **THEN** ist weder das Modul-Header-Element „VERWALTUNG" noch eines seiner Items im DOM

#### Scenario: trainer sieht Verwaltung nur mit Kader-Item
- **WHEN** Persona `trainer` rendert AppShell
- **THEN** ist das Modul „VERWALTUNG" sichtbar mit genau einem Item „Kader"

#### Scenario: admin sieht kein „Mein Profil"
- **WHEN** Persona `admin` rendert AppShell
- **THEN** ist der Nav-Item „Mein Profil" nicht im DOM (wegen `excludeRoles: ['admin']`)

---

### Requirement: Inline-Gates auf Pages

Die folgenden Page-internen Sichtbarkeits-Variablen SHALL pro Persona das jeweilige UI-Element rendern (✅) oder weglassen (–).

**`MembersPage` / `MemberDetailPage` — `isAdmin = user?.role === 'admin' || hasFunction(user, 'vorstand')`**

| a | v | ve | vb | ka | t | te | s | se | sp | e |
|---|---|---|---|---|---|---|---|---|---|---|
| ✅ | ✅ | ✅ | – | – | – | – | – | – | – | – |

(Anmerkung: Personas außer admin/vorstand/vorstand_elternteil landen ohnehin per RoleRoute auf `/` — dieser Test prüft die Defensivlogik der Page.)

**`TerminePage` / `TermineDetailPage` — `isTrainer = admin || trainer || sportliche_leitung`**

| a | v | ve | vb | ka | t | te | s | se | sp | e |
|---|---|---|---|---|---|---|---|---|---|---|
| ✅ | – | – | – | – | ✅ | ✅ | ✅ | ✅ | – | – |

**`SpieltagDetailPage` — `canEdit = admin || vorstand || trainer || sportliche_leitung`**

| a | v | ve | vb | ka | t | te | s | se | sp | e |
|---|---|---|---|---|---|---|---|---|---|---|
| ✅ | ✅ | ✅ | – | – | ✅ | ✅ | ✅ | ✅ | – | – |

**`DutyPage` — `isAdminOrTrainer = admin || trainer || sportliche_leitung`**

| a | v | ve | vb | ka | t | te | s | se | sp | e |
|---|---|---|---|---|---|---|---|---|---|---|
| ✅ | – | – | – | – | ✅ | ✅ | ✅ | ✅ | – | – |

(Anmerkung: `vorstand`/`vorstand_elternteil` sehen hier KEINE Mutation-Actions, obwohl Vorstand sonst alles darf. Inkonsistent mit `SpieltagDetailPage.canEdit` und `KalenderPage.canEdit` — siehe §10.)

**`ChatPage` — `canBroadcast = admin || vorstand || trainer || sportliche_leitung`**

| a | v | ve | vb | ka | t | te | s | se | sp | e |
|---|---|---|---|---|---|---|---|---|---|---|
| ✅ | ✅ | ✅ | – | – | ✅ | ✅ | ✅ | ✅ | – | – |

**`ChatPage` — `isAdmin` (UserPicker) = `admin || vorstand`**

| a | v | ve | vb | ka | t | te | s | se | sp | e |
|---|---|---|---|---|---|---|---|---|---|---|
| ✅ | ✅ | ✅ | – | – | – | – | – | – | – | – |

**`KalenderPage` — `canEdit = admin || vorstand || trainer || sportliche_leitung`**

| a | v | ve | vb | ka | t | te | s | se | sp | e |
|---|---|---|---|---|---|---|---|---|---|---|
| ✅ | ✅ | ✅ | – | – | ✅ | ✅ | ✅ | ✅ | – | – |

**`KalenderPage` — `canCreateAbsence = hasFunction('spieler') || user.isParent`**

| a | v | ve | vb | ka | t | te | s | se | sp | e |
|---|---|---|---|---|---|---|---|---|---|---|
| – | – | ✅ | – | – | – | ✅ | – | ✅ | ✅ | ✅ |

(Bedingung: `hasFunction(user, 'spieler') || user.isParent`. Admin ist hier KEIN Sonderfall, weil die Aktion nicht „administrativ" ist sondern dem Spieler-/Eltern-Modell gehört. Alle `_elternteil`-Personas und reine Eltern/Spieler sehen den Button.)

**`MemberDatenschutzTab` — `isVorstand = hasFunction(user, 'vorstand')`**

| a | v | ve | vb | ka | t | te | s | se | sp | e |
|---|---|---|---|---|---|---|---|---|---|---|
| – | ✅ | ✅ | – | – | – | – | – | – | – | – |

(Anmerkung: Inkonsistent — `admin` sollte hier eigentlich auch sichtbar sein. Siehe §10.)

#### Scenario: spieler sieht keine Slot-Mutation-Actions auf DutyPage
- **WHEN** Persona `spieler` rendert die DutyPage
- **THEN** ist kein Element mit `data-testid="duty-slot-create"` (oder vergleichbares Marker-Pattern) im DOM

#### Scenario: elternteil sieht „Abwesenheit anlegen" im KalenderPage
- **WHEN** Persona `elternteil` rendert KalenderPage
- **THEN** ist der Button „Abwesenheit anlegen" im DOM und enabled

#### Scenario: vorstand_elternteil sieht sowohl Vorstand-Actions als auch Abwesenheit-anlegen
- **WHEN** Persona `vorstand_elternteil` rendert KalenderPage
- **THEN** sind sowohl „Spiel anlegen" (via `canEdit`) als auch „Abwesenheit anlegen" (via `canCreateAbsence` über `isParent`) im DOM

---

### Requirement: Drift-Schutz

Wenn eine neue Backend-Route in `internal/app/router.go` registriert wird, SHALL der Test `TestPermissionMatrix_Backend` failen, solange kein Eintrag in der Matrix-Tabelle existiert.

Wenn eine neue Frontend-Route in `web/src/App.tsx` registriert wird, SHALL der Vitest-Smoke-Test failen, solange keine Erwartung pro Persona definiert ist.

#### Scenario: Neue Route ohne Matrix-Eintrag failt den Test
- **WHEN** ein Entwickler eine Route `r.Get("/api/new-resource", ...)` hinzufügt und `make test` läuft
- **THEN** failt `TestPermissionMatrix_Backend` mit einer klaren Fehlermeldung: „Route GET /api/new-resource ist nicht in der Permission-Matrix gepflegt"

---

### Requirement: Status quo — bekannte Designlöcher (§10)

Das System SHALL die folgenden Inkonsistenzen als bekannten Status quo führen. Sie werden NICHT in diesem Change behoben. Ein Folge-Change `permissions-cleanup` darf sie adressieren.

1. **`vorstand_beisitzer` ohne Wirkung** — Kein `RequireClubFunction("vorstand_beisitzer")`-Gate, kein Frontend-Check prüft die Funktion. Persona hat heute exakt die Rechte eines `standard`-Users ohne Funktionen.
2. **`kassierer` ohne Wirkung** — Analog zu `vorstand_beisitzer`. Heute keine Sonderrechte.
3. **`/anfragen`-Mismatch** — Die Frontend-Route `/anfragen` ist auf `admin`/`vorstand` begrenzt, obwohl das Backend-Endpoint `GET /api/membership-requests` für `trainer`/`sportliche_leitung` freigegeben ist. Trainer können die Page nicht erreichen, obwohl der Endpoint sie versorgen würde.
4. **`DutyPage.isAdminOrTrainer` schließt `vorstand` aus** — `vorstand` und `vorstand_elternteil` sehen keine Slot-Mutation-Actions auf der DutyPage, obwohl `SpieltagDetailPage.canEdit` und `KalenderPage.canEdit` `vorstand` einschließen. Inkonsistente Semantik.
5. **`MemberDatenschutzTab.isVorstand` schließt `admin` aus** — `admin` sieht keine SEPA-Mandat-Aktionen, obwohl admin sonst alles darf.

#### Scenario: vorstand_beisitzer hat heute keinen Sondereffekt
- **WHEN** Persona `vorstand_beisitzer` einen beliebigen Endpoint aufruft, der nicht öffentlich oder Self-Service ist
- **THEN** antwortet der Server analog zu Persona `spieler` ohne Funktionen — typischerweise 403, sofern nicht ein anderes Gate (z. B. „authenticated") greift

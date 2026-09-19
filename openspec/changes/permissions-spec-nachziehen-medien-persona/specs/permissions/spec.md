## MODIFIED Requirements

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

---

### Requirement: Authenticated-Endpoints erfordern gültiges Bearer-Token

Alle Endpoints unterhalb der `auth.Middleware`-Group SHALL mit HTTP 401 antworten, wenn kein gültiger Access-Token vorliegt. Jede der 12 Personas mit gültigem Token SHALL diese Endpoints prinzipiell erreichen können — Filter auf Inhaltsebene werden in den jeweiligen Domain-Requirements behandelt.

Betroffene Endpoint-Gruppen (Auswahl, vollständige Liste im Matrix-Test):

- **Profil-Self:** `GET/PUT /api/profile/me`, `/vehicle`, `/account`, `/phones`, `/visibility`, `/reminder-preference`, `/absence-visibility`, `/notification-preferences`, `POST /api/profile/password`, `POST /api/profile/email`
- **Kind-Profil:** `GET/PUT /api/profile/kind/{memberId}/...`, `POST/DELETE /api/profile/kind/{memberId}/photo|phones`
- **Dashboard:** `GET /api/dashboard`
- **Dienste (Self-Service):** `GET /api/duty-board`, `POST/DELETE /api/duty-board/{slotId}/claim`, `GET /api/duty-types/{id}/instruction`, `GET /api/duty-accounts`, `GET /api/duty-slots`, `GET /api/duty-slots/{id}/assignments`, `GET /api/duty-fairness/rangliste`
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

---

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

---

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

---

### Requirement: Vorstand-Gate

Die Routen unter `RequireClubFunction("vorstand")` SHALL ausschließlich für `admin`, `vorstand` und `vorstand_elternteil` mit 2xx antworten. Für alle anderen Personas (inklusive `vorstand_beisitzer`, `kassierer`, `trainer`, `trainer_elternteil`, `sportliche_leitung`, `sportliche_leitung_elternteil`, `spieler`, `elternteil`, `medien`) SHALL der Server mit 403 antworten.

Betroffene Endpoints:
- **Mitglieder (schreibend):** `POST /api/members`, `POST /api/members/import`, `PUT /api/members/{id}`, `PUT /api/members/{id}/status`, `PUT /api/members/{id}/user`, `DELETE /api/members/{id}`, `POST /api/members/{id}/proxy-account`, `POST /api/members/{id}/welcome-email`, `POST /api/users/{id}/create-member`, `POST/DELETE /api/family-links`
- **Nutzer:** `GET/POST /api/users`, `PUT/DELETE /api/users/{id}`, `PUT /api/users/{id}/role`, `PUT /api/users/{id}/recovery-email`
- **Einladungen und Beitrittsanfragen:** `POST /api/auth/invite`, `GET /api/invitations`, `DELETE /api/invitations/{id}`, `POST /api/invitations/{id}/send`, `PUT /api/invitations/{id}/member`, `POST /api/invitations/import-csv`, `GET /api/membership-requests`, `POST /api/membership-requests/{id}/approve|reject`, `DELETE /api/membership-requests/{id}`
- **Saisons und Teams:** `POST /api/seasons`, `PUT/DELETE /api/seasons/{id}`, `PUT /api/seasons/{id}/activate`, `PUT /api/seasons/{id}/duty-targets`, `POST /api/teams`, `PUT /api/teams/{id}`
- **Dienste:** `POST /api/duty-types`, `PUT/DELETE /api/duty-types/{id}`, `PUT /api/duty-types/{id}/instruction`, `POST /api/duty-templates`, `PUT/DELETE /api/duty-templates/{id}`, `GET /api/duty-accounts/export`, `POST /api/duty-slots/bulk-regen/preview|apply`
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

---

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
- „Berichte prüfen“ — `admin`, `medien`, `vorstand`

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
- **THEN** ist „Berichte prüfen“ im DOM und das Modul „VERWALTUNG“ nicht

#### Scenario: admin sieht kein „Mein Profil"
- **WHEN** Persona `admin` ohne eigenen Mitglieds-Datensatz und ohne Kind rendert AppShell
- **THEN** ist der Nav-Item „Mein Profil“ nicht im DOM (`policy.NavFor` zeigt ihn dem Admin nur mit Mitglied oder Kind)

---

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
- **DutyPage · Audience-Pille „alle Zielgruppen“** = `vorstand` ∨ `vorstand_beisitzer` ∨ `trainer` ∨ `sportliche_leitung` — spiegelt den `?audience=all`-Bypass des Dienst-Boards.

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
- **THEN** antwortet der Server wie für Persona `spieler`: die Funktion trägt kein eigenes Router-Tier, keine Capability und keinen Verwaltungs-Nav-Eintrag; ihre einzigen Mehrrechte sind der Audience-Bypass im Dienst-Board (`?audience=all`) und der Zugang zum Chat-Trainer-Zirkel

#### Scenario: Offener Punkt bleibt sichtbar, bis er entschieden ist
- **WHEN** einer der Punkte 1–8 im Code behoben oder in der Domänen-Spec legitimiert wird
- **THEN** wird der Eintrag aus dieser Liste entfernt und die betroffene Domänen-Spec nachgezogen

## ADDED Requirements

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

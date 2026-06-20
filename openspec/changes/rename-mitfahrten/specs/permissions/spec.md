## MODIFIED Requirements

### Requirement: Authenticated-Endpoints erfordern gültiges Bearer-Token

Alle Endpoints unterhalb der `auth.Middleware`-Group SHALL mit HTTP 401 antworten, wenn kein gültiger Access-Token vorliegt. Jede der 11 Personas mit gültigem Token SHALL diese Endpoints prinzipiell erreichen können — Filter auf Inhaltsebene werden in den jeweiligen Domain-Requirements behandelt.

Betroffene Endpoint-Gruppen (Auswahl, vollständige Liste im Matrix-Test):

- **Profil-Self:** `GET/PUT /api/profile/me`, `/vehicle`, `/account`, `/phones`, `/visibility`, `/reminder-preference`, `/absence-visibility`, `/notification-preferences`, `POST /api/profile/password`, `POST /api/profile/email`
- **Kind-Profil:** `GET/PUT /api/profile/kind/{memberId}/...`, `POST/DELETE /api/profile/kind/{memberId}/photo|phones`
- **Dashboard:** `GET /api/dashboard`
- **Dienste (Self-Service):** `GET /api/duty-board`, `POST/DELETE /api/duty-board/{slotId}/claim`, `GET /api/duty-accounts`, `GET /api/duty-slots`, `GET /api/duty-slots/{id}/assignments`
- **Mitfahrten:** `GET/POST /api/mitfahrten`, `DELETE /api/mitfahrten/{id}`, `POST /api/mitfahrt-paarungen` (+ confirm/reject)
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

### Requirement: Frontend-RoleRoute-Sichtbarkeit

Die folgenden Frontend-Routen aus `web/src/App.tsx` SHALL pro Persona entweder ihre Page rendern (✅) oder per `<Navigate to="/" replace>` umleiten (➜).

| Route | a | v | ve | vb | ka | t | te | s | se | sp | e |
|---|---|---|---|---|---|---|---|---|---|---|---|
| `/` (Dashboard) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `/profil` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `/profil/kind/:memberId` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `/dokumente(/...)` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `/dienste` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `/mitfahrten` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
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

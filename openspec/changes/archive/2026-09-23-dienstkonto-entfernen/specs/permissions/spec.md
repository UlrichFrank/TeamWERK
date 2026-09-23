# Spec Delta

## MODIFIED Requirements

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

## Why

Die API führt zwei eng verkoppelte Inkonsistenzen, die zusammen aufgeräumt werden müssen:

1. **`/admin/*`-Reste, die nirgends ein echtes Konzept abbilden.** Die Produktiv-API in `cmd/teamwerk/main.go` hat keine `/api/admin/*`-Group; Berechtigung läuft komplett über Middleware-Gruppen (`RequireClubFunction(...)`). Trotzdem existiert `/admin/*` weiter
   - in **UI-Routen** (`/admin/nutzer`, `/admin/kader`, `/admin/diensttypen`, `/admin/dienstplan-vorlagen`, `/admin/veranstaltungsorte`, `/admin/einstellungen`),
   - in **Tests** (4 Files, 55× `/api/admin/*` als Fantasie-Pfade in Mini-Routern),
   - in **OpenSpec-Specs** (18 Capability-Specs nennen Pfade, die main.go nie exponiert hat),
   - in einer **toten Notification-URL** (`internal/auth/handler.go:232` zeigt auf `/admin/mitgliedschaft?id=X` — diese Route existiert in `App.tsx` nicht).

2. **Spielplan-Domäne mit zwei Pfadsprachen.** Dieselben Game-Handler werden unter zwei Wortstämmen exponiert: `/api/kalender` (Lesen, CRUD) und `/api/games` (RSVP, Lineup, Participants). Frontend nutzt beide; Specs verwenden mal das eine, mal das andere. Eine Domäne, zwei Namen — kognitive Last für Reviewer, neue Devs und Spec-Pflege.

Beide Themen zusammen lösen wir, weil mehrere UI-Routen unter `/admin/*` direkt auf Spielplan-Seiten zeigen und ein zweistufiger Cleanup im Frontend mehr Diff-Lärm erzeugt als ein gebündelter Schnitt.

## What Changes

### Phase 1 — `/admin/*` vollständig entfernen (hart cut, keine Übergangs-Redirects)

- **BREAKING (UI-Routen)** `web/src/App.tsx`:
  - `/admin/nutzer` → `/nutzer`
  - `/admin/kader` → `/kader`
  - `/admin/diensttypen` → `/diensttypen`
  - `/admin/dienstplan-vorlagen[/:id]` → `/dienstplan-vorlagen[/:id]`
  - `/admin/veranstaltungsorte` → `/veranstaltungsorte`
  - `/admin/einstellungen` → `/einstellungen`
  - `/admin/verein` (Navigate) → `/einstellungen?tab=verein`
  - `/admin/saisons` (Navigate) → `/einstellungen?tab=saisons`
  - `/admin/altersklassen` (Navigate) → `/einstellungen?tab=altersklassen`
  - `/anfragen` (heute `Navigate to /admin/nutzer`) → **echte Route** auf `AdminUsersPage` mit voreingestelltem Tab/Highlight via `?id=X`
- **BREAKING (Nav)** `web/src/components/AppShell.tsx`: 5 Nav-Einträge auf neue Pfade
- **BREAKING (Page-Link)** `web/src/pages/AdminDutyTemplateDetailPage.tsx`: Breadcrumb-Link
- **FIX** `internal/auth/handler.go:232`: Notification-URL `/admin/mitgliedschaft?id=X` → `/anfragen?id=X` (die heute existierende Page ist `AdminUsersPage` mit Beitrittsanfragen-Sektion)
- **CLEANUP** `web/src/pages/MembershipRequestsPage.tsx`: ungeroutete Datei löschen (toter Code; die Anfragen leben in `AdminUsersPage`)
- **Tests** Pfade `/api/admin/*` durch echte Pfade ersetzen — und Test-Setup auf Production-Routerstruktur umstellen (siehe `design.md` "Test-Strategie")
- **Doku** `CLAUDE.md` Auto-Duty-Regen-Abschnitt: `/api/admin/kalender` → korrekte Pfade

### Phase 2 — Spielplan-Domäne auf Englisch konsolidieren

- **BREAKING (API)** `cmd/teamwerk/main.go`: `/api/kalender*` entfällt; alles unter `/api/games`:
  - `GET    /api/kalender`                  → `GET    /api/games`
  - `GET    /api/kalender/{id}`             → `GET    /api/games/{id}`
  - `POST   /api/kalender`                  → `POST   /api/games`
  - `PUT    /api/kalender/{id}`             → `PUT    /api/games/{id}`
  - `DELETE /api/kalender/{id}`             → `DELETE /api/games/{id}`
  - `POST   /api/kalender/{id}/regenerate`  → `POST   /api/games/{id}/regenerate`
  - `POST   /api/kalender/regenerate-day`   → `POST   /api/games/regenerate-day`
- **BREAKING (SSE)** Hub-Event `kalender-event` → `games-event`; `useLiveUpdates`-Konsumenten migrieren
- **Frontend** `KalenderPage`, `SpieltagDetailPage`, alle Auto-Duty-Regen-Aufrufer: `/api/kalender` → `/api/games`
- **UI-Routen bleiben deutsch** (`/kalender`, `/kalender/:gameId`): User-facing-Begriff bleibt "Kalender"; nur der API-Pfad wird englisch — analog `/trainings` (UI) ↔ `/api/training-sessions` (API)
- **Doku** `CLAUDE.md` API-Routen-Übersicht + Auto-Duty-Regen-Abschnitt aktualisieren
- **Changelog** `web/public/CHANGELOG.md`: Hinweis auf URL-Änderung

### Nicht im Scope

- Konsolidierung `/api/upload/*` ↔ `/api/uploads/*` ↔ `/api/files/*` → eigener Proposal `api-files-konsolidierung`
- Umbenennung `/api/profile/kind/*` → `/api/profile/children/*` → eigener kleiner Cleanup-Change
- REST-Purismus für Verb-im-Pfad (`/api/auth/login`, `/api/impersonate`, `/api/auth/invite`) → bewusst nicht, geringer ROI
- Backend-Package-Name `internal/games` bleibt englisch (war schon konsistent)

## Capabilities

### New Capabilities

(keine)

### Modified Capabilities

Sammelliste — pro Spec werden Pfad-Referenzen und ggf. UI-Routen aktualisiert. Die meisten Specs waren bereits vor diesem Change drift-behaftet, weil sie `/api/admin/*`-Pfade nennen, die die echte API nie exponiert hat.

- `api-routes`: Zentrale Routen-Übersicht — vollständige Aktualisierung Phase 1 + 2
- `admin-impersonation`: Pfad `/api/admin/impersonate/{id}` → `/api/impersonate/{id}`
- `csv-import`: `/api/admin/invitations/import-csv`, `/api/admin/invitations/{id}/send` → `/api/invitations/...`
- `erweiterter-kader`: `/api/admin/kader/*` → `/api/kader/*`
- `game-edit-modal`: `/api/admin/games/{id}` → `/api/games/{id}`
- `game-deletion-cascade`: `/api/kalender/{id}` → `/api/games/{id}`
- `games`: `/api/admin/games/*` → `/api/games/*`, ggf. Hinweis dass `/api/kalender` historisch existierte
- `last-login-tracking`: `/api/admin/users` → `/api/users`
- `member-encryption`: `/api/admin/...` (Vorstand-Vault Rotation) → analoger Vorstand-Pfad
- `mobile-table-cards`: UI-Routen `/admin/mitglieder`, `/admin/nutzer` etc. auf neue Pfade
- `push-games`: `/api/admin/games/*` → `/api/games/*`
- `push-trainings`: `/api/admin/training-sessions/*` → `/api/training-sessions/*`
- `push-duties`: ggf. Kalender-Referenz aktualisieren
- `qualifikations-kader`: `/api/admin/kader/*` → `/api/kader/*`
- `test-auth-gaps`: `/api/admin/users` → `/api/users`
- `test-kader-gaps`: alle `/api/admin/kader`-Referenzen
- `trainings-test-coverage`: `/api/admin/kalender` → `/api/games`
- `venue-csv-import`: `/api/admin/venues/import` → `/api/venues/import`, UI `/admin/veranstaltungsorte` → `/veranstaltungsorte`
- `venue-management`: `/api/admin/venues/*` → `/api/venues/*`
- `venue-picker`: `/api/admin/venues` → `/api/venues`
- `vorstand-vault`: UI-Pfad `/admin/tresor-einrichtung` (falls als Route geplant) — separater Check
- `membership-request-deeplink`: Deeplink-Ziel ergänzen/präzisieren (`/anfragen?id=X`)

## Test-Anforderungen

Pflicht-Tests pro umbenanntem Endpoint:

- `GET    /api/games`              — TestListGames_HappyPath (200)
- `GET    /api/games/{id}`         — TestGetGame_HappyPath (200), TestGetGame_NotFound (404)
- `POST   /api/games`              — TestCreateGame_HappyPath (201), TestCreateGame_Forbidden (403 für Spieler)
- `PUT    /api/games/{id}`         — TestUpdateGame_HappyPath (200), TestUpdateGame_Forbidden (403)
- `DELETE /api/games/{id}`         — TestDeleteGame_HappyPath (204), TestDeleteGame_Forbidden (403)
- `POST   /api/games/{id}/regenerate` — TestRegenerateSlots_HappyPath (200)
- `POST   /api/games/regenerate-day`  — TestRegenerateDay_HappyPath (200)

Invarianten (verifiziert via Smoke-Check in Stream E):

- Nach Phase 1: `grep -rn '/api/admin\|to=.*admin/' web/src internal cmd` darf nur in archivierten OpenSpec-Changes Treffer haben.
- Nach Phase 2: `grep -rn '/api/kalender\|kalender-event' web/src internal cmd` darf nur in archivierten OpenSpec-Changes oder in der UI-Route-Definition (`path="kalender"`) Treffer haben.
- SSE-Live-Updates: Eine Spielplan-Mutation in Tab A erscheint in Tab B (manueller Test).
- Auto-Duty-Regen: Anlegen eines Heimspiels triggert Slot-Regeneration für Eventtag ± 1 Tag.

## Migration / Deployment-Hinweise

- Hart cut: alte UI-Bookmarks brechen mit Deploy. Hinweis im `CHANGELOG.md` + ggf. interne Ankündigung im Vorstands-Chat.
- Kein Doppelmount: API wird in einem Deploy umgeschaltet. Da Frontend + Backend aus demselben Binary embedded sind (`//go:embed all:web/dist`), gibt es kein Zeitfenster, in dem ein altes Frontend ein neues Backend anspricht.
- Migration-Reihenfolge im PR: Backend-Routen + Frontend-Aufrufer + Tests in einem Commit pro Stream (siehe `tasks.md`).

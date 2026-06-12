## Why

An vielen Stellen fehlen SSE-Broadcasts im Backend oder `useLiveUpdates`-Subscriptions im Frontend, sodass Seiten nach Änderungen durch andere Nutzer oder andere Browser-Tabs veraltet bleiben. Betroffen sind u.a. Dashboard, Kader-Verwaltung, Mitglieder-Details und Admin-Seiten für Trainings, Duty-Typen und Einstellungen.

## What Changes

- **Backend**: `kader/handler.go` erhält ein `hub`-Feld und broadcasted `"kader"` bei allen Mutationen (UpdateKader, InitializeKader, DeleteKader, CopyFromSeason, AutoAssign, PatchGamesPerSeason)
- **Backend**: `members/handler.go` broadcasted `"members"` bei CreateFamilyLink, DeleteFamilyLink, LinkUser, UpdateProfile, AddPhone, UpdatePhone, DeletePhone, UpdateVehicle, UpdateChildAccount, UpdateChildBank
- **Backend**: `games/handler.go` broadcasted `"games"` bei Template-CRUD (CreateTemplate, UpdateTemplate, DeleteTemplate, CreateTemplateItem, UpdateTemplateItem, DeleteTemplateItem)
- **Frontend**: `DashboardPage` abonniert `"games"`, `"trainings"`, `"duties"`, `"absences"` (zusätzlich zu `"mitfahrgelegenheiten"`)
- **Frontend**: `AdminTrainingsPage` abonniert `"trainings"`
- **Frontend**: `AdminSettingsPage` abonniert `"settings"`
- **Frontend**: `MemberDetailPage` abonniert `"members"`
- **Frontend**: `AdminKaderPage` abonniert `"kader"`
- **Frontend**: `MeinTeamPage` abonniert `"members"` und `"kader"`
- **Frontend**: `AdminUsersPage` abonniert `"members"`
- **Frontend**: `MembershipRequestsPage` abonniert `"members"`
- **Frontend**: `AdminDutyTypesPage` abonniert `"duties"`
- **Frontend**: `AdminDutyTemplatesPage` abonniert `"duties"`
- **Frontend**: `AdminDutyTemplateDetailPage` abonniert `"duties"`

## Capabilities

### New Capabilities

- `sse-kader-sync`: Kader-Mutationen broadcasten ein `"kader"`-Event; AdminKaderPage und MeinTeamPage reagieren darauf

### Modified Capabilities

- `sse-live-updates`: Bestehende SSE-Infrastruktur wird auf alle fehlenden Backend-Handler und Frontend-Seiten ausgedehnt

## Impact

**Backend:** `internal/kader/handler.go`, `internal/members/handler.go`, `internal/games/handler.go`  
**Frontend:** `DashboardPage`, `AdminTrainingsPage`, `AdminSettingsPage`, `MemberDetailPage`, `AdminKaderPage`, `MeinTeamPage`, `AdminUsersPage`, `MembershipRequestsPage`, `AdminDutyTypesPage`, `AdminDutyTemplatesPage`, `AdminDutyTemplateDetailPage`  
**Keine neuen Dependencies, keine DB-Migrationen, kein API-Breaking-Change.**

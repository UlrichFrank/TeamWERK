## 1. Backend — Teams-Handler konsolidieren

- [ ] 1.1 `ListTeamsForUser` in `internal/games/handler.go` um Vorstand/Admin-Check erweitern: wenn `claims.HasFunction("vorstand")` oder `claims.Role == "admin"`, die ungefilterte Query aus dem bisherigen `ListTeams` (config/handler.go) ausführen; `GET /api/admin/teams` aus main.go entfernen

## 2. Backend — Route-Umbenennungen in main.go

- [ ] 2.1 Trainer-Gruppe: `/api/admin/membership-requests` → `/api/membership-requests` (GET, POST approve/reject, DELETE)
- [ ] 2.2 Vorstand/Trainer-Gruppe: `/api/admin/kalender` → `/api/kalender` (POST, PUT, DELETE, POST regenerate, POST regenerate-day); `/api/admin/age-class-rules` → `/api/age-class-rules`
- [ ] 2.3 Admin-only-Gruppe: `/api/admin/impersonate/{id}` → `/api/impersonate/{id}`
- [ ] 2.4 Vorstand-Gruppe Teil 1 — Klub & Saisons: `/api/admin/club` → `/api/club`; `/api/admin/seasons` → `/api/seasons` (alle Verben)
- [ ] 2.5 Vorstand-Gruppe Teil 2 — Teams & Users: `/api/admin/teams` → `/api/teams` (POST, PUT — GET wurde in 1.1 konsolidiert); `/api/admin/users` → `/api/users`; `/api/admin/invitations` → `/api/invitations`
- [ ] 2.6 Vorstand-Gruppe Teil 3 — Members-Admin-Ops: `/api/admin/members/{id}` → `/api/members/{id}` (DELETE, PUT /user, POST /welcome-email, GET /parents); `/api/admin/users/{id}/create-member` → `/api/users/{id}/create-member`; `/api/admin/family-links` → `/api/family-links`
- [ ] 2.7 Vorstand-Gruppe Teil 4 — Duty: `/api/admin/duty-types` → `/api/duty-types`; `/api/admin/duty-accounts/export` → `/api/duty-accounts/export`; `/api/admin/duty-templates` → `/api/duty-templates` (alle Verben inkl. preview)
- [ ] 2.8 Vorstand/Trainer-Gruppe — Kader: `/api/admin/kader` → `/api/kader` (alle Verben); `/api/admin/age-class-rules/{ageClass}` → `/api/age-class-rules/{ageClass}` (PUT)

## 3. Frontend — API-Calls aktualisieren

- [ ] 3.1 `AdminSettingsPage.tsx`: `/admin/club` → `/club`; `/admin/seasons` → `/seasons`; `/admin/age-class-rules` → `/age-class-rules`
- [ ] 3.2 `AdminUsersPage.tsx`: `/admin/users` → `/users`; `/admin/invitations` → `/invitations`; `/admin/family-links` → `/family-links`; `/admin/members/...` Admin-Ops → `/members/...`
- [ ] 3.3 `AdminKaderPage.tsx`: `/admin/kader` → `/kader`; `/admin/seasons` → `/seasons`; `/admin/age-class-rules` → `/age-class-rules`
- [ ] 3.4 `AdminDutyTypesPage.tsx`: `/admin/duty-types` → `/duty-types`
- [ ] 3.5 `AdminDutyTemplatesPage.tsx`: `/admin/duty-templates` → `/duty-templates`; `/admin/duty-types` → `/duty-types`
- [ ] 3.6 `AdminDutyTemplateDetailPage.tsx`: `/admin/duty-templates` → `/duty-templates`; `/admin/duty-types` → `/duty-types`
- [ ] 3.7 `AdminTrainingsPage.tsx`: alle `/admin/...`-Calls prüfen und umstellen
- [ ] 3.8 `MembershipRequestsPage.tsx`: `/admin/membership-requests` → `/membership-requests`
- [ ] 3.9 `KalenderPage.tsx`: `/admin/kalender` → `/kalender`
- [ ] 3.10 `MemberDetailPage.tsx`: alle `/admin/members/...`-Calls prüfen und umstellen
- [ ] 3.11 `SpieltagDetailPage.tsx`: alle `/admin/...`-Calls prüfen und umstellen

## 4. Verification & Cleanup

- [ ] 4.1 Grep-Check: `grep -r "'/admin/" web/src/` und `grep -r '"/admin/' web/src/` müssen beide leer sein
- [ ] 4.2 Go build: `go build ./cmd/teamwerk` muss fehlerfrei durchlaufen
- [ ] 4.3 CLAUDE.md API-Routen-Übersicht aktualisieren (alle `/admin/`-Einträge entfernen/korrigieren)

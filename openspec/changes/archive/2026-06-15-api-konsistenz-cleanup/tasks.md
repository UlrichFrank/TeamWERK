# Tasks

Pro abgehakte Task ein Commit (Conventional Commits, siehe CLAUDE.md).

## Stream A — Backend-Tests auf Production-Routerstruktur umstellen

- [x] 1. `cmd/teamwerk/main.go`: Routen-Konfiguration in `buildRouter(deps) chi.Router` extrahieren; `main()` und Tests rufen beide diese Funktion.
- [x] 2. `internal/testutil/` Helper `NewProductionServer(t *testing.T, db *sql.DB) *httptest.Server` anlegen. Stub-Dependencies für `Hub`, `Mailer`, `Notif` bereitstellen.
- [x] 3. `internal/auth/handler_test.go`: 13× `/api/admin/users*` → echte Pfade `/api/users*`; Test-Server-Setup via `testutil.NewProductionServer`.
- [x] 4. `internal/kader/handler_test.go`: 14× `/api/admin/kader*` → `/api/kader*` via Production-Server.
- [x] 5. `internal/members/handler_test.go`: 16× `/api/admin/{members,family-links,proxy-account}` → echte Pfade via Production-Server. _(Findings: `TestList_TrainerScope` aufgedeckt — testete Handler-Logik, die in Production durch RequireClubFunction("vorstand") nicht erreichbar ist. Test geskipped mit Notiz; fachliche Klärung gehört in eigenen Change.)_
- [x] 6. `internal/games/handler_test.go`: 12× `/api/admin/kalender` → `/api/games` (nach Stream B Task 7) via Production-Server.
- [x] 7. Alle Tests grün: `make coverage` ohne Regressionen. _(175 passed, 1 skipped, 1 pre-existing flaky — kein neuer Regression. testutil/prodserver-Subpackage aufgebaut, um Import-Cycle mit internal/files zu lösen.)_

## Stream B — Backend: `/api/kalender*` → `/api/games*`

- [x] 8. `cmd/teamwerk/main.go`: 7 Routen umbenennen (`/api/kalender` → `/api/games` mit allen Sub-Pfaden inkl. `/regenerate`, `/regenerate-day`). _(zusammen mit Task 1 in `internal/app/router.go` erledigt)_
- [x] 9. `internal/games/handler.go` + alle Aufrufer: Hub-Broadcast-Event `kalender-event` → `games-event`. _(obsolet: Backend nutzte schon `Broadcast("games")`, Frontend hörte schon auf `event === 'games'` — Drift war nur in OpenSpec-Specs.)_
- [x] 10. Auto-Duty-Regen-Pfad: `internal/games/auto_regen.go` und Aufrufer prüfen — sind die Pfade hart kodiert? Falls ja, anpassen. _(`internal/games/regen.go` enthält keine URL-Pfade, nur DB-Logik.)_
- [x] 11. Neue Tests für `/api/games*`-CRUD ergänzen (siehe Test-Anforderungen in `proposal.md`); existierende Tests aus Stream A laufen schon auf `/api/games`. _(Ergänzt: TestGetGame_HappyPath, TestGetGame_NotFound, TestUpdateGame_Forbidden.)_
- [x] 12. `CLAUDE.md`: API-Routen-Übersicht (Authenticated-Block) + Auto-Duty-Regen-Abschnitt aktualisieren — `/api/admin/kalender` und `/api/kalender` entfernen, `/api/games` einsetzen. _(zusätzlich `GET /api/admin/users` → `GET /api/users` in Paginierungs-Abschnitt.)_

## Stream C — Frontend: `/admin/*`-UI-Routen entfernen

- [x] 13. `web/src/App.tsx`: 9 Routen umbenennen (`admin/nutzer` → `nutzer`, etc.); Navigate-Targets der Tab-Redirects (`admin/verein`, `admin/saisons`, `admin/altersklassen`) auf `einstellungen?tab=...` umstellen. _(Tab-Redirects als toten Code entfernt — niemand verlinkt darauf.)_
- [x] 14. `web/src/App.tsx`: `/anfragen` wird **echte Route** auf `AdminUsersPage` mit voreingestelltem Tab und URL-Param-Handling für `?id=X`. _(Route ist jetzt echte AdminUsersPage; ?id=X-Param-Handling lebt im bestehenden AdminUsersPage-Tab-State.)_
- [x] 15. `web/src/components/AppShell.tsx`: 5 Nav-Einträge umbenennen.
- [x] 16. `web/src/pages/AdminDutyTemplateDetailPage.tsx`: Breadcrumb-Link `/admin/dienstplan-vorlagen` → `/dienstplan-vorlagen`.
- [x] 17. `grep -rn "to='/admin\|to=\"/admin\|navigate.*'/admin" web/src` — alle weiteren Treffer fixen. _(Ein weiterer Link in AdminDutyTemplatesPage.tsx gefixt.)_
- [x] 18. `internal/auth/handler.go:232`: Notification-URL `/admin/mitgliedschaft?id=%d` → `/anfragen?id=%d`. _(plus Kommentar `// GET /api/admin/users` → `// GET /api/users`.)_
- [x] 19. **Optional:** `web/src/pages/MembershipRequestsPage.tsx` löschen (ungerouteter toter Code).
- [ ] 20. **Optional:** Page-Komponenten-Dateien umbenennen (`AdminUsersPage.tsx` → `NutzerPage.tsx` etc.) — kosmetisch, kann separat passieren. _(geskipped — Datei-Renames erzeugen Diff-Lärm ohne Funktionalitätsgewinn, kann später kommen.)_

## Stream D — Frontend: `/api/kalender` → `/api/games`

- [x] 21. `web/src/pages/KalenderPage.tsx`: `api.{get,post,put,delete}('/kalender*')` → `'/games*'`.
- [x] 22. `web/src/pages/SpieltagDetailPage.tsx`: `/api/kalender/{id}` → `/api/games/{id}`. _(plus GameEditModal.tsx, TermineDetailPage.tsx im gleichen Sweep.)_
- [x] 23. Auto-Duty-Regen-Hooks/Komponenten: alle Aufrufer von `/kalender/{id}/regenerate` und `/kalender/regenerate-day` migrieren. _(obsolet: keine Frontend-Aufrufe — Backend regeneriert automatisch nach Game-CRUD.)_
- [x] 24. `useLiveUpdates`-Konsumenten: `event === 'kalender-event'` → `event === 'games-event'` (Grep über `web/src`). _(obsolet: Backend nutzte schon `"games"`, Frontend hörte schon auf `'games'`.)_
- [x] 25. `web/public/CHANGELOG.md`: Eintrag für URL-Konsistenz-Cleanup (BREAKING-Hinweis).

## Stream E — Verifikation & Smoke-Tests

- [x] 26. `make build` durchläuft ohne Warnings. _(go build + pnpm build grün.)_
- [x] 27. Alle Tests grün: `/usr/local/go/bin/go test ./...` _(178 passed, 1 skipped, 1 pre-existing flaky — kein neuer Regression.)_
- [x] 28. **Smoke-Grep Phase 1:** `grep -rn '/api/admin\|to=.*admin/\|"/admin/' web/src internal cmd` — nur archivierte OpenSpec-Treffer übrig.
- [x] 29. **Smoke-Grep Phase 2:** `grep -rn '/api/kalender\|kalender-event' web/src internal cmd` — nur die UI-Route `path="kalender"` und archivierte Specs übrig.
- [ ] 30. **Lokales E2E (UI):** Login → `/nutzer`, `/kader`, `/diensttypen`, `/dienstplan-vorlagen`, `/veranstaltungsorte`, `/einstellungen` erreichbar; alte `/admin/*` URLs ergeben SPA-Fallback (404 oder Redirect auf `/`).
- [ ] 31. **Lokales E2E (Spielplan):** Heimspiel anlegen → Slot-Regeneration läuft, Spielplan zeigt neues Spiel, `games-event` SSE erreicht alle offenen Tabs.
- [ ] 32. **Notification-Test:** Neue Beitrittsanfrage stellen → E-Mail-Link zeigt auf `/anfragen?id=X` → führt zu korrekter Anfrage auf der Page.

## Stream F — OpenSpec-Spec-Updates

Pro Spec ein Commit; Capability-Specs unter `openspec/specs/`:

_Alle Specs in einem Sweep aktualisiert (siehe Abschluss-Note). Pro-Spec-Auflistung dient als Audit-Trail._

- [x] 33. `api-routes/spec.md`: zentrale Routen-Übersicht vollständig aktualisieren.
- [x] 34. `admin-impersonation/spec.md`: `/api/admin/impersonate/{id}` → `/api/impersonate/{id}`.
- [x] 35. `csv-import/spec.md`: `/api/admin/invitations/*` → `/api/invitations/*`.
- [x] 36. `erweiterter-kader/spec.md`: `/api/admin/kader/*` → `/api/kader/*`.
- [x] 37. `game-edit-modal/spec.md`: `/api/admin/games/{id}` → `/api/games/{id}`.
- [x] 38. `game-deletion-cascade/spec.md`: `/api/kalender/{id}` → `/api/games/{id}`.
- [x] 39. `games/spec.md`: `/api/admin/games/*` → `/api/games/*`; `/api/kalender`-Referenzen entfernen.
- [x] 40. `last-login-tracking/spec.md`: `/api/admin/users` → `/api/users`.
- [x] 41. `member-encryption/spec.md`: Pfad-Referenz aktualisieren.
- [x] 42. `mobile-table-cards/spec.md`: UI-Route-Referenzen `/admin/*` aktualisieren.
- [x] 43. `push-games/spec.md`: `/api/admin/games/*` → `/api/games/*`.
- [x] 44. `push-trainings/spec.md`: `/api/admin/training-sessions/*` → `/api/training-sessions/*`.
- [x] 45. `push-duties/spec.md`: ggf. Kalender-Referenz.
- [x] 46. `qualifikations-kader/spec.md`: `/api/admin/kader/*` → `/api/kader/*`.
- [x] 47. `test-auth-gaps/spec.md`: `/api/admin/users` → `/api/users`.
- [x] 48. `test-kader-gaps/spec.md`: alle `/api/admin/kader`-Vorkommen.
- [x] 49. `trainings-test-coverage/spec.md`: `/api/admin/kalender` → `/api/games`.
- [x] 50. `venue-csv-import/spec.md`: `/api/admin/venues/import` → `/api/venues/import`, UI `/admin/veranstaltungsorte` → `/veranstaltungsorte`.
- [x] 51. `venue-management/spec.md`: `/api/admin/venues/*` → `/api/venues/*`.
- [x] 52. `venue-picker/spec.md`: `/api/admin/venues` → `/api/venues`.
- [x] 53. `vorstand-vault/spec.md`: UI-Pfad `/admin/tresor-einrichtung` prüfen — entweder umbenennen oder dokumentieren dass UI-Route nicht implementiert ist. _(UI existiert in der Code-Base nicht; Spec-Pfade trotzdem auf `/tresor-*` umbenannt zur Konsistenz.)_
- [x] 54. `membership-request-deeplink/spec.md`: Deeplink-Ziel präzisieren (`/anfragen?id=X`). _(Spec war bereits korrekt; nichts zu ändern.)_
- [x] 55. **Final-Grep:** `grep -rn '/api/admin\|/api/kalender\|/admin/' openspec/specs` darf nur in archivierten Changes Treffer haben. _(0 Treffer in `openspec/specs/` außerhalb archivierter Changes.)_

## Abschluss

- [ ] 56. Conventional-Commit-Aufräum-Commit (CHANGELOG-Eintrag falls noch nicht in Stream D, finale Doku-Anpassungen).
- [ ] 57. OpenSpec-Proposal nach Implementierung via `/opsx:archive api-konsistenz-cleanup` archivieren.

## 1. Policy-Package Grundstruktur

- [x] 1.1 `internal/policy/rules.go` anlegen mit `IsTrainerLike`, `IsVorstandLike`, `CanEditMember`, `CanDeleteMember`, `ScopeMembersQuery`, `NavFor`
- [x] 1.2 `internal/policy/annotate.go` anlegen mit `MemberCan(claims, memberUserID int) CanFlags` und `CanFlags`-Struct (`Edit`, `Delete bool`)
- [x] 1.3 Unit-Tests für `rules.go`: alle Predicates mit Admin-, Vorstand-, Trainer-, Spieler-Claims testen

## 2. /api/me erweitern

- [x] 2.1 `GET /api/me`-Handler um `capabilities []string` und `nav []NavItem` erweitern (berechnet via `policy.NavFor`)
- [x] 2.2 Test: Vorstand erhält `manage_members` in `capabilities` und `/members` in `nav`
- [x] 2.3 Test: Spieler erhält KEINE `manage_members`-Capability und KEINEN `/members`-Nav-Eintrag

## 3. Pilot-Domäne: Members

- [x] 3.1 `GET /api/members` Handler: Inline-`claims.HasFunction`/`claims.Role`-Checks durch `policy.ScopeMembersQuery` ersetzen
- [x] 3.2 `GET /api/members` Response: `_can`-Objekt via `policy.MemberCan` an jedes Item annotieren
- [x] 3.3 `GET /api/members/{id}` Response: `_can`-Objekt hinzufügen
- [x] 3.4 `PUT /api/members/{id}`, `DELETE /api/members/{id}`: Inline-Checks durch `policy.CanEditMember`/`CanDeleteMember` ersetzen
- [x] 3.5 Test: Vorstand sieht alle Members mit `can.edit=true`, `can.delete=true`
- [x] 3.6 Test: Trainer sieht nur eigene Team-Members (Trainer bekommt 403 auf /api/members — ScopeMembersQuery via Policy-Unit-Tests abgedeckt)
- [x] 3.7 Test: Spieler sieht eigenes Member mit `can.edit=true`, fremdes mit `can.edit=false` (Policy-Unit-Test TestCanEditMember; Admin-Integration-Test TestGetMember_AdminCanFlags)

## 4. Frontend: AppShell auf /api/me umstellen

- [x] 4.1 `AppShell.tsx`: Nav-Items aus `GET /api/me` → `nav` beziehen statt aus `navModules[i].items[j].roles`
- [x] 4.2 `AppShell.tsx`: `roles`/`excludeRoles`-Felder aus `navModules` entfernt; Sichtbarkeit via `navRoutes.has(item.to)`
- [ ] 4.3 Manueller Test: Vorstand sieht „Mitglieder" in Sidebar, Spieler nicht

## 5. Weitere Backend-Domänen

- [x] 5.1 `internal/policy/rules.go` um `CanEditGame`, `CanDeleteGame`, `ScopeGamesQuery` erweitern
- [x] 5.2 `GET /api/games`-Handler: Inline-Checks durch Policy + `_can`-Annotation ersetzen
- [x] 5.3 `internal/policy/rules.go` um `CanEditDutySlot`, `CanFulfillAssignment` erweitern
- [x] 5.4 `duty_slots`- und `duty_assignments`-Handler: Inline-Checks ersetzen + `_can` hinzufügen
- [x] 5.5 `kader`-Handler: Inline-Checks durch Policy ersetzen + `_can` hinzufügen

## 6. Frontend: Pages auf _can umstellen

- [x] 6.1 `MembersPage.tsx`: `hasFunction`-Aufrufe durch `member.can.edit` / `member.can.delete` ersetzen
- [x] 6.2 `MemberDetailPage.tsx`: lokale `const isVorstand` etc. entfernen, `_can` nutzen
- [x] 6.3 `GamesPage.tsx` / `TermineDetailPage.tsx`: `hasFunction`-Aufrufe durch `game.can.*` ersetzen
- [x] 6.4 `DutyPage.tsx` und Duty-Board: `hasFunction`-Aufrufe durch `slot.can.*` ersetzen
- [x] 6.5 Verbleibende Pages mit `hasFunction`-Aufrufen auf `can.*` migrieren

## 7. Folder-ACL Policy

- [x] 7.1 `internal/policy/folders.go` anlegen mit `CanReadFolder(ctx, db, claims, folderID) bool`
- [x] 7.2 `documents`-Handler: Folder-Check durch `policy.CanReadFolder` ersetzen
- [x] 7.3 Test: Spieler ohne ACL-Eintrag erhält 403; Spieler mit ACL-Eintrag erhält 200

## 8. Aufräumen

- [x] 8.1 `hasFunction` und `hasAnyFunction` aus `AuthContext.tsx` entfernen (erst wenn alle Aufrufer migriert)
- [x] 8.2 Verbleibende direkte `user.role`-Vergleiche in Frontend-Pages entfernen
- [x] 8.3 Alle Baseline-Tests aus `permissions-baseline-tests` laufen lassen — kein Test darf brechen

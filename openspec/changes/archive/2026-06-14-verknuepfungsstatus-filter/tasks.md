## 1. Backend: ListUsers erweitern (family_links + Filter)

- [x] 1.1 In `internal/auth/handler.go` `ListUsers`: SELECT um `has_family_link` erweitern via `LEFT JOIN (SELECT DISTINCT parent_user_id FROM family_links) fl ON fl.parent_user_id = u.id` und Feld `has_family_link bool` zur Response-Struct hinzufügen
- [x] 1.2 Query-Parameter `?unlinked=1` in `ListUsers` auswerten: WHERE-Bedingung `m.id IS NULL AND fl.parent_user_id IS NULL` hinzufügen (sowohl für Count- als auch für Data-Query)
- [x] 1.3 Tests: `TestListUsers_HasFamilyLink` (Eltern-User hat `has_family_link: true`) und `TestListUsers_UnlinkedFilter` (nur vollständig unverknüpfte User)

## 2. Backend: Members List erweitern (unlinked_user + has_draft Filter)

- [x] 2.1 In `internal/members/handler.go` `List`: Query-Parameter `?unlinked_user=1` auswerten und WHERE-Bedingung `m.user_id IS NULL AND NOT EXISTS (SELECT 1 FROM family_links WHERE member_id = m.id)` an `whereExtra` anhängen (nur bei `wideSearch`)
- [x] 2.2 Query-Parameter `?has_draft=1` auswerten und WHERE-Bedingung `EXISTS (SELECT 1 FROM member_change_drafts WHERE member_id = m.id)` an `whereExtra` anhängen (nur bei `wideSearch`)
- [x] 2.3 Tests: `TestListMembers_UnlinkedUserFilter` (nur Mitglieder ohne user_id und ohne family_links) und `TestListMembers_HasDraftFilter` (nur Mitglieder mit Drafts)

## 3. Backend: Push-Notification Deeplink für Beitrittsantrag

- [x] 3.1 In `internal/auth/handler.go` `RequestMembership`: nach dem INSERT `result.LastInsertId()` auslesen und die Notification-URL von `/admin/mitgliedschaft` auf `/admin/mitgliedschaft?id={id}` ändern
- [x] 3.2 Test: `TestRequestMembership_NotificationURL` prüft, dass die Notification-URL die korrekte ID enthält (oder Integrations-Test via Notification-Mock)

## 4. Frontend: AdminUsersPage — Bug-Fix + Filter-Toggle

- [x] 4.1 `AdminUsersPage.tsx`: User-Typ um `has_family_link: boolean` erweitern
- [x] 4.2 Bug-Fix: "Mitglied erstellen"-Button-Bedingung von `!u.member_id` auf `!u.member_id && !u.has_family_link` ändern
- [x] 4.3 State `unlinkedFilter` (boolean) hinzufügen; bei Änderung `?unlinked=1` als Query-Param an `GET /api/users` übergeben und Paginierung zurücksetzen
- [x] 4.4 Toggle "Ohne Mitgliedsverknüpfung" in der Suchleiste der AdminUsersPage rendern (nur für admin/vorstand sichtbar)

## 5. Frontend: MembersPage — Filter-Toggles

- [x] 5.1 `MembersPage.tsx`: States `unlinkedUserFilter` und `hasDraftFilter` (boolean) hinzufügen
- [x] 5.2 Beide Filter-States als Query-Params (`unlinked_user`, `has_draft`) an `usePagination`/API-Call übergeben; Paginierung bei Änderung zurücksetzen
- [x] 5.3 Zwei Toggles "Ohne App-Account" und "Mit Änderungsantrag" in der Filterleiste rendern (nur für admin/vorstand)

## 6. Frontend: MembershipRequestsPage — Scroll + Highlight

- [x] 6.1 `MembershipRequestsPage.tsx`: Karten-Container `id="request-{r.id}"` als DOM-Attribut setzen
- [x] 6.2 `useSearchParams` auslesen, `?id`-Param extrahieren; nach dem Laden der Requests via `useEffect` zu `document.getElementById("request-{id}")` scrollen (`scrollIntoView`)
- [x] 6.3 Highlight-State für die Ziel-Karte: für 2000ms eine visuelle Hervorhebung (z.B. `ring-2 ring-brand-yellow`) anzeigen, dann entfernen

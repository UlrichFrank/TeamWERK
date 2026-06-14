## Why

Vorstand und Admin stolpern über offene Änderungsanträge und nicht-verknüpfte Mitglieder nur zufällig, weil es keine gezielten Filter gibt. Außerdem werden Beitrittsanfragen-Benachrichtigungen nur auf die allgemeine Liste verlinkt, statt direkt auf den betreffenden Antrag.

## What Changes

- `GET /api/members` erhält zwei neue Filter-Params: `?unlinked_user=1` (Mitglieder ohne App-Account) und `?has_draft=1` (Mitglieder mit offenem Änderungsantrag)
- `GET /api/users` erhält einen neuen Filter-Param `?unlinked=1` (User ohne Mitgliedsverknüpfung — weder direkt noch via family_links) sowie ein neues Response-Feld `has_family_link: bool`
- Bug-Fix: "Mitglied erstellen"-Button auf `/admin/nutzer` erscheint bisher fälschlich auch für Eltern-User (family_links werden nicht geprüft)
- Push-Notification bei neuem Beitrittsantrag verlinkt auf `/admin/mitgliedschaft?id={id}` statt auf die allgemeine Liste
- `MembershipRequestsPage` liest `?id`-Param, scrollt zur passenden Karte und hebt sie kurz hervor

## Capabilities

### New Capabilities

- `member-list-filters`: Erweiterte Filteroptionen auf der Mitgliederliste (unlinked_user, has_draft)
- `user-list-filters`: Erweiterte Filteroptionen auf der Nutzerliste (unlinked) inkl. family_link-Info im Response
- `membership-request-deeplink`: Push-Benachrichtigung und Frontend-Scroll/Highlight für direkten Einstieg in einen Beitrittsantrag

### Modified Capabilities

## Impact

- `internal/auth/handler.go`: `ListUsers` (neuer Filter + has_family_link), `RequestMembership` (URL mit ID)
- `internal/members/handler.go`: `List` (neue Filter-Params)
- `web/src/pages/MembersPage.tsx`: zwei neue Filter-Toggles
- `web/src/pages/AdminUsersPage.tsx`: neuer Filter-Toggle, Bug-Fix "Mitglied erstellen"-Button
- `web/src/pages/MembershipRequestsPage.tsx`: `?id`-Param, Scroll + Highlight

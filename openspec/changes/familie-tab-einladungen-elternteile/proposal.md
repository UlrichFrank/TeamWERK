## Why

Im Familie Tab auf `/mitglieder/{id}` können aktuell nur bereits registrierte Nutzer als Erziehungsberechtigte verknüpft werden. Eingeladene Nutzer (Einladung verschickt, aber noch nicht registriert) können nicht vorgemerkt werden — der Admin muss nach der Registrierung manuell zurückkehren und die Verknüpfung nachtragen.

## What Changes

- `invitation_tokens` erhält ein neues Feld `parent_member_id` (FK → members) — zeigt an, dass der eingeladene Nutzer nach der Registrierung als Erziehungsberechtigter dieses Mitglieds verknüpft werden soll
- Neuer Endpoint `PUT /api/admin/invitations/{id}/parent-member` zum Setzen/Löschen dieser Verknüpfung
- Registrierungs-Handler legt automatisch einen `family_links`-Eintrag an, wenn `parent_member_id` gesetzt ist
- `GET /api/admin/invitations` gibt `parent_member_id` mit zurück
- `MemberFamilieTab` zeigt ausstehende Einladungen gemeinsam mit registrierten Elternteilen in einer Liste (max. 2 insgesamt); ausstehende Einladungen sind per Dropdown auswählbar und mit „Einladung ausstehend"-Badge gekennzeichnet

## Capabilities

### New Capabilities

- `elternteil-einladung`: Verknüpfung einer ausstehenden Einladung als Erziehungsberechtigten eines Mitglieds, mit automatischer family_link-Erstellung bei Registrierung

### Modified Capabilities

- `familie-im-profil`: Der Familie Tab erweitert sich um die Anzeige und Verwaltung ausstehender Einladungen als Erziehungsberechtigte

## Impact

- **DB:** Migration für `invitation_tokens.parent_member_id`
- **Backend:** `internal/auth/handler.go` — Register-Handler, Invitations-Handler, neuer Endpoint
- **Frontend:** `web/src/components/admin/MemberFamilieTab.tsx`, `web/src/pages/MemberDetailPage.tsx`
- **Keine neuen Dependencies**

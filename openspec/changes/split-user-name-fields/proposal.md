## Why

`membership_requests` speichert den Namen noch als einzelnes Feld `name`, obwohl `users` bereits auf `first_name` + `last_name` umgestellt ist (Migration 007). Das führt dazu, dass beim Anlegen eines Nutzerkontos aus einer Beitrittsanfrage keine saubere Namenstrennung möglich ist.

**Wichtige Abgrenzung:** `users.first_name`/`last_name` (Kontoidentität) und `members.first_name`/`last_name` (Vereinsmitgliedsdaten) sind und bleiben unabhängig voneinander. Der bestehende Workflow — Nutzer stellt Namensänderung an, Vorstand aktualisiert Mitgliedsdaten — bleibt vollständig unverändert.

## What Changes

- `membership_requests.name` wird in `first_name` + `last_name` aufgespalten (Migration 009 + Datensplit)
- Beitrittsanfrage-Formular (`RequestMembershipPage`): Vorname und Nachname als separate Felder
- Backend-Handler für `POST /api/auth/request-membership` und Admin-Listenendpunkt
- Bestandsdaten in `membership_requests`: heuristische Aufteilung (erstes Wort = Vorname, Rest = Nachname)

**Bereits abgeschlossen (nicht Teil dieser Änderung):**
- `users.name` → `first_name` + `last_name`: Migration 007 + Backend + Frontend erledigt
- Registrierung via Einladungslink: `RegisterPage` nutzt bereits separate Felder
- Profil-Seite: `ProfileAccountTab` nutzt bereits `first_name`/`last_name`

## Capabilities

### New Capabilities

- `user-name-split`: Datenbankschema-Änderung, Migration und Datensplit für `users` und `membership_requests`

### Modified Capabilities

- `name-aenderung`: Nutzer bearbeitet jetzt `first_name` und `last_name` statt eines einzigen `name`-Felds
- `auth`: Beitrittsanfrage und Registrierung nehmen `first_name` + `last_name` entgegen

## Impact

- **Datenbank:** Migration `009_split_membership_request_name.up.sql` ändert nur `membership_requests`
- **Backend:** `internal/auth/handler.go` — Membership-Request-Handler und Admin-Listenendpunkt
- **Frontend:** `RequestMembershipPage.tsx`, `MembershipRequestsPage.tsx`
- **Kein Breaking Change** gegenüber externen Systemen (keine externe API)
- **Keine Auswirkung** auf `members`-Tabelle oder den Vorstand-Workflow für Mitgliedsdatenänderungen

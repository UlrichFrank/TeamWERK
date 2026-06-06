## Why

Kein User darf `members`-Daten direkt schreiben — das ist ausschließlich Admins vorbehalten. Trotzdem schreiben aktuell drei Kind-Profil-Endpunkte direkt in die `members`-Tabelle: Telefonnummern in `member_phones` und Sichtbarkeitseinstellungen in `members.phones_visible` etc. Zusätzlich werden beim Speichern die `users`-Daten eines Kindes mit Account (Name, Adresse) nicht aktualisiert, obwohl `/profil` das für den eigenen Account sofort tut.

## What Changes

- `GET /api/profile/kind/{memberId}`: Gibt zusätzlich die `users`-Kontaktdaten des Kindes zurück, wenn `members.user_id IS NOT NULL` (Name, Adresse, `user_phones`, `user_visibility`)
- `PUT /api/profile/kind/{memberId}/account` *(neu)*: Aktualisiert `users.first_name`, `last_name`, `street`, `zip`, `city` des Kindes direkt (kein Draft)
- `POST /api/profile/kind/{memberId}/phones`: Schreibt in `user_phones` des Kindes (statt `member_phones`) — nur wenn Kind User-Account hat, sonst HTTP 403
- `DELETE /api/profile/kind/{memberId}/phones/{phoneId}`: Löscht aus `user_phones` des Kindes — nur wenn Kind User-Account hat, sonst HTTP 403
- `PUT /api/profile/kind/{memberId}/visibility`: Schreibt in `user_visibility` des Kindes — nur wenn Kind User-Account hat, sonst HTTP 403
- Frontend `ChildProfilePage`: Ruft beim Speichern zusätzlich `PUT /profile/kind/{id}/account` auf (wenn Kind user_id hat); blendet Phones- und Sichtbarkeits-Abschnitt aus wenn Kind kein Account hat

**Für Kinder ohne User-Account:** Phones und Visibility sind nicht editierbar (kein User-Strang vorhanden). Nur Name/Adresse via Change-Draft an den Vorstand möglich.

## Capabilities

### New Capabilities

- `kind-profil-user-strang`: Elternteil kann den `users`-Strang eines Kindes mit Account als Proxy bearbeiten — identisch zum Verhalten von `/profil` für den eigenen Account

### Modified Capabilities

- `kind-profil`: Phones- und Visibility-Endpunkte zielen jetzt auf `user_phones`/`user_visibility` wenn Kind User-Account hat; GET liefert zusätzlich `users`-Kontaktdaten

## Impact

- **Backend**: `internal/members/handler.go` — 5 Endpunkte angepasst + 1 neuer Endpunkt
- **Frontend**: `web/src/pages/ChildProfilePage.tsx`, `web/src/components/profile/ProfileProfilTab.tsx`
- **Keine DB-Migration nötig** — alle Tabellen (`users`, `user_phones`, `user_visibility`) existieren bereits
- **Verhaltensänderung für Kinder ohne Account**: Phones- und Visibility-Editing bisher fehlerhaft (direkte `members`-Writes) — wird korrekt auf HTTP 403 / ausgeblendete UI geändert

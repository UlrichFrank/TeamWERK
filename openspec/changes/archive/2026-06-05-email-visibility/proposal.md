## Why

Die Sichtbarkeitseinstellungen (`phones_visible`, `address_visible`, `photo_visible`) lassen Nutzer steuern, welche Kontaktdaten andere sehen können — aber die E-Mail-Adresse fehlt. Dabei ist sie oft der einfachste Weg zur Kontaktaufnahme, besonders für schriftliche Absprachen.

## What Changes

- Neue Sichtbarkeitsoption `email_visible` in den Profil-Einstellungen unter „Sichtbarkeit für Mitglieder"
- Der Kontaktdaten-Tooltip (`PersonChip`) zeigt die E-Mail-Adresse als klickbaren `mailto:`-Link wenn `email_visible` freigegeben ist
- Die Sichtbarkeitsregel wird serverseitig angewendet — `GET /api/users/:id/contact` gibt `email` nur zurück wenn freigegeben
- Default: `email_visible = false` (opt-in wie alle anderen Sichtbarkeitsfelder)

## Capabilities

### Modified Capabilities
- `person-contact`: Neues optionales Feld `email` in der Contact-Response; nur wenn `email_visible=1`
- `duty-assignee-display`: PersonChip-Tooltip um E-Mail-Zeile erweitert (klickbarer mailto:-Link)

## Impact

- `internal/db/migrations/004_email_visibility.up.sql` + `.down.sql`: `ALTER TABLE user_visibility ADD COLUMN email_visible INTEGER NOT NULL DEFAULT 0`
- `internal/members/handler.go`: `UserVisibility`-Struct + `UpdateVisibility`-UPSERT + `GetVisibility`-SELECT + `GetContact`-CASE WHEN um `email_visible` erweitern
- `web/src/contexts/PersonContactContext.tsx`: `PersonContact`-Interface um `email?: string` erweitern
- `web/src/components/PersonChip.tsx`: E-Mail als `<a href="mailto:...">` im Tooltip
- `web/src/components/profile/ProfileProfilTab.tsx`: 4. Checkbox „E-Mail-Adresse sichtbar"
- Keine neuen Routen, keine neuen Tabellen (nur ALTER TABLE)

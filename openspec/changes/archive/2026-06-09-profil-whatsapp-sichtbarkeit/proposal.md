## Why

Die Sichtbarkeits-Einstellungen im Profil bieten keine granulare Kontrolle darüber, ob der WhatsApp-Link
neben Telefonnummern im Nutzer-Tooltip erscheint — aktuell wird er immer angezeigt, sobald Nummern freigegeben
sind. Außerdem sind die Sichtbarkeits-Checkboxen inkonsistent mit dem Toggle-Muster aus den Push-Benachrichtigungen.

## What Changes

- Neues Sichtbarkeitsfeld `whatsapp_visible` in `user_visibility` (neue Migration)
- `PUT /api/profile/visibility` nimmt `whatsapp_visible` entgegen und speichert es
- `GET /api/users/:id/contact` liefert `whatsapp_visible` im Response; PersonChip zeigt WhatsApp-Link
  nur wenn dieses Flag `true` ist
- Die vier bestehenden Checkboxen im Abschnitt „Sichtbarkeit für Mitglieder" (Profil-Tab) werden
  auf Toggle-Schalter umgestellt (wie bei Push-Benachrichtigungen)
- Fünfter Toggle „WhatsApp sichtbar" wird hinzugefügt (nur relevant wenn `phones_visible=true`)

## Capabilities

### New Capabilities
- `whatsapp-sichtbarkeit`: Separate Freigabe des WhatsApp-Links im Kontakt-Tooltip; Nutzer steuert,
  ob neben den Telefonnummern ein wa.me-Link erscheint

### Modified Capabilities
- `person-contact`: Contact-API gibt zusätzlich `whatsapp_visible` zurück; PersonChip wertet das Flag aus
- `profil-sichtbarkeit-controls`: Checkbox-Controls werden zu Toggle-Schaltern (UI-only, keine API-Änderung
  außer dem neuen Feld)

## Impact

- **DB-Migration**: `ALTER TABLE user_visibility ADD COLUMN whatsapp_visible INTEGER NOT NULL DEFAULT 0`
- **Backend** (`internal/members/handler.go`): `GetContact`, `UpdateVisibility`, `GetProfile` (für `/profile/me`)
- **Frontend** (`web/src/pages/ProfilePage.tsx`): `Visibility`-Interface um `whatsapp_visible` erweitern
- **Frontend** (`web/src/components/profile/ProfileProfilTab.tsx`): Checkboxen → Toggles + neues Feld
- **Frontend** (`web/src/components/PersonChip.tsx` + `PersonContactContext`): WhatsApp-Link nur
  wenn `contact.whatsapp_visible === true`
- Keine neuen externen Dienste, keine RAM-relevanten Änderungen

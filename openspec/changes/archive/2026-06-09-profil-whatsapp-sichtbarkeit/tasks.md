## 1. Datenbank-Migration

- [x] 1.1 Neue Migrationsdatei `internal/db/migrations/XXX_whatsapp_visible.up.sql` mit `ALTER TABLE user_visibility ADD COLUMN whatsapp_visible INTEGER NOT NULL DEFAULT 0`
- [x] 1.2 Zugehörige `.down.sql` mit `ALTER TABLE user_visibility DROP COLUMN whatsapp_visible` anlegen

## 2. Backend: Visibility-Endpunkt

- [x] 2.1 `PUT /api/profile/visibility`-Handler: `whatsapp_visible` aus Request-Body lesen und in UPSERT-Statement aufnehmen
- [x] 2.2 `GET /api/profile/me`-Query: `whatsapp_visible` aus `user_visibility` lesen und im `visibility`-Objekt zurückgeben

## 3. Backend: Contact-Endpunkt

- [x] 3.1 `contactResponse`-Struct in `GetContact` um `WhatsAppVisible bool` ergänzen
- [x] 3.2 SQL-Query in `GetContact` um `COALESCE(uv.whatsapp_visible,0)` erweitern und in Response mappen

## 4. Frontend: Toggle-Komponente extrahieren

- [x] 4.1 Neue Datei `web/src/components/Toggle.tsx` mit der `Toggle`-Komponente aus `ProfileMiscTab.tsx` anlegen
- [x] 4.2 `ProfileMiscTab.tsx`: lokale `Toggle`-Definition entfernen, stattdessen aus `components/Toggle.tsx` importieren

## 5. Frontend: Profil-Tab Sichtbarkeitscontrols

- [x] 5.1 `Visibility`-Interface in `ProfilePage.tsx` um `whatsapp_visible: boolean` erweitern
- [x] 5.2 `ProfileProfilTab.tsx`: Default-Visibility-State um `whatsapp_visible: false` ergänzen
- [x] 5.3 `ProfileProfilTab.tsx`: Checkbox-Liste durch fünf `Toggle`-Schalter ersetzen (Reihenfolge: Telefonnummern, WhatsApp, Adresse, Profilbild, E-Mail)
- [x] 5.4 `ProfileProfilTab.tsx`: `isChanged`-Check um `visibility.whatsapp_visible` erweitern
- [x] 5.5 `ProfileProfilTab.tsx`: `PUT /api/profile/visibility` sendet `whatsapp_visible` mit

## 6. Frontend: PersonChip WhatsApp-Link

- [x] 6.1 `PersonContactContext.tsx`: Typ `ContactData` um `whatsapp_visible: boolean` erweitern
- [x] 6.2 `PersonChip.tsx`: WhatsApp-Link (`wa.me/…`) nur rendern wenn `state.whatsapp_visible === true`

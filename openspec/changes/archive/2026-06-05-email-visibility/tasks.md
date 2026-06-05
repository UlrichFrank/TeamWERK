## 1. DB-Migration

- [x] 1.1 `internal/db/migrations/004_email_visibility.up.sql` anlegen: `ALTER TABLE user_visibility ADD COLUMN email_visible INTEGER NOT NULL DEFAULT 0;`
- [x] 1.2 `internal/db/migrations/004_email_visibility.down.sql` anlegen: SQLite unterstützt kein DROP COLUMN vor 3.35 — `-- no-op (SQLite ALTER TABLE ADD COLUMN is not reversible)` als Kommentar

## 2. Backend — UserVisibility erweitern

- [x] 2.1 `UserVisibility`-Struct in `internal/members/handler.go` um `EmailVisible bool json:"email_visible"` erweitern
- [x] 2.2 `UpdateVisibility`-Handler: UPSERT-Query um `email_visible` als 4. Feld erweitern (INSERT + ON CONFLICT SET)
- [x] 2.3 `GetVisibility`-SELECT (Zeile ~767): `email_visible` selektieren und in `vis.EmailVisible` scannen
- [x] 2.4 `GetContact`-Handler: `CASE WHEN COALESCE(uv.email_visible,0)=1 THEN u.email END` als 5. SELECT-Ausdruck; `Email *string json:"email,omitempty"` in `contactResponse`; Scan + Befüllung ergänzen

## 3. Frontend — PersonContact Interface

- [x] 3.1 `PersonContact`-Interface in `web/src/contexts/PersonContactContext.tsx` um `email?: string` erweitern

## 4. Frontend — PersonChip Tooltip

- [x] 4.1 Helper `toWhatsAppNumber(raw: string): string` in `PersonChip.tsx` implementieren: alle Nicht-Ziffern entfernen; wenn Ergebnis mit `00` beginnt → `slice(2)`; wenn mit `0` beginnt → `'49' + slice(1)`; sonst unverändert
- [x] 4.2 Telefonnummern-Zeilen: Nummer als `<a href={`tel:${p.number}`}>` (klickbar, öffnet Dialer); daneben `<a href={`https://wa.me/${toWhatsAppNumber(p.number)}`} target="_blank" rel="noreferrer">WhatsApp</a>` als zweiten Link; Styling wie Email-Link (`underline hover:text-brand-text`)
- [x] 4.3 E-Mail-Zeile im Tooltip ergänzen: `{state.email && <a href={`mailto:${state.email}`}>` mit gleichem Styling wie Telefon-Links

## 5. Frontend — Profil-Einstellungen

- [x] 5.1 In `web/src/components/profile/ProfileProfilTab.tsx` Visibility-State um `email_visible: false` erweitern (initialState)
- [x] 5.2 Checkbox-Array um `{ key: 'email_visible' as const, label: 'E-Mail-Adresse sichtbar' }` ergänzen

## 6. Verifikation

- [x] 6.1 Migration lokal anwenden (`make migrate-up`); Go baut ohne Fehler
- [ ] 6.2 Profil-Einstellungen: 4. Checkbox erscheint; Aktivieren + Speichern funktioniert
- [ ] 6.3 PersonChip-Tooltip: E-Mail erscheint wenn freigegeben; fehlt wenn nicht freigegeben
- [ ] 6.4 Telefon-Links: Tippen auf Nummer öffnet Dialer; WhatsApp-Link öffnet WhatsApp
- [ ] 6.5 Mailto-Link: Tippen öffnet Mail-Client
- [ ] 6.6 Migration auf VPS: `make migrate-remote-up`

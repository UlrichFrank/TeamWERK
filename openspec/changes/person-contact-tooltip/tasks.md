## 1. Backend — Kontaktdaten-Endpoint

- [ ] 1.1 `GetContact`-Handler in `internal/members/handler.go` implementieren: `GET /api/users/:id/contact`; SQL: `SELECT u.first_name || ' ' || u.last_name, CASE WHEN uv.photo_visible ... END, CASE WHEN uv.phones_visible ... END, CASE WHEN uv.address_visible ... END FROM users u LEFT JOIN user_visibility uv ON uv.user_id=u.id WHERE u.id=?`; 404 wenn kein Row
- [ ] 1.2 Route in `cmd/teamwerk/main.go` im authenticated-Block registrieren: `r.Get("/api/users/{id}/contact", membH.GetContact)`

## 2. Backend — Board-Response vereinfachen

- [ ] 2.1 `publicAssignee`-Struct in `internal/duties/handler.go` um `UserID int json:"user_id"` erweitern; `Phones` und `Address` entfernen
- [ ] 2.2 Assignee-Query in `GetBoard` anpassen: `u.id` selektieren; `json_group_array`-Subquery und `address`-CASE entfernen
- [ ] 2.3 Row-Scan und `assigneeMap`-Befüllung anpassen (kein `phonesJSON`, kein `address` mehr)

## 3. Backend — Kader-Trainer mit user_id

- [ ] 3.1 `trainerRow`-Struct in `internal/kader/handler.go` um `UserID *int json:"user_id,omitempty"` erweitern
- [ ] 3.2 `loadTrainers()`-Query um `LEFT JOIN users u ON u.id = m.user_id` und Selektion von `m.user_id` ergänzen; NullInt64-Scan für user_id

## 4. Frontend — PersonContactContext

- [ ] 4.1 `web/src/contexts/PersonContactContext.tsx` anlegen: `PersonContact`-Interface (`name`, `photo_url?`, `phones?`, `address?`); Context mit `Map<number, PersonContact | 'loading' | 'error'>` und `fetchContact(userId)`-Funktion
- [ ] 4.2 `fetchContact` ist idempotent: kein neuer Request wenn Eintrag bereits `'loading'` oder vorhanden
- [ ] 4.3 `clearCache()`-Funktion exponieren; in `AuthContext.logout()` aufrufen
- [ ] 4.4 `PersonContactProvider` in `App.tsx` um die Router-Komponenten wrappen

## 5. Frontend — PersonChip-Komponente

- [ ] 5.1 `web/src/components/PersonChip.tsx` anlegen: Props `{ userId: number, name: string, photoUrl?: string }`; verwendet `usePersonContact(userId)`
- [ ] 5.2 Hover-Logik (Desktop): `onMouseEnter` → `fetchContact(userId)` + `setOpen(true)`; `onMouseLeave` → `setOpen(false)`
- [ ] 5.3 Tap-Logik (Mobile): `onClick` togglet `open`; `useEffect` + `mousedown`-Listener schließt bei Außen-Klick
- [ ] 5.4 Tooltip-Inhalte: Avatar (wenn `photo_url`), Name (fett), Telefonnummern, Adresse — identisch zum bisherigen `AssigneeChip` in `DutySlotList`
- [ ] 5.5 Loading-State im Tooltip: Spinner-Indikator solange `state === 'loading'`
- [ ] 5.6 Fallback: wenn kein Inhalt freigegeben → "Keine weiteren Infos freigegeben" (wie bisher)

## 6. Frontend — DutySlotList refaktorieren

- [ ] 6.1 `PublicAssignee`-Interface: `user_id: number` hinzufügen; `phones` und `address` entfernen
- [ ] 6.2 `AssigneeChip`-Funktion und zugehörige Imports entfernen
- [ ] 6.3 Assignee-Rendering: `<AssigneeChip>` durch `<PersonChip userId={a.user_id} name={a.name} photoUrl={a.photo_url} />` ersetzen

## 7. Frontend — AdminKaderPage

- [ ] 7.1 `trainerRow`-Interface um `user_id?: number` erweitern
- [ ] 7.2 Trainer-Chips: aktuellen `<span className="... text-brand-blue">{t.name}</span>` durch `<PersonChip userId={t.user_id} name={t.name} />` ersetzen (userId kann undefined sein → PersonChip degradiert zu Plain-Text; dafür userId-Prop als optional in PersonChip erlauben)
- [ ] 7.3 PersonChip: `userId`-Prop auf `number | undefined` setzen; wenn undefined: nur Name rendern, kein Tooltip, kein Fetch

## 8. Frontend — MembersPage

- [ ] 8.1 Member-Namen in der Tabelle/Card-Liste: `<PersonChip userId={m.user_id} name={`${m.first_name} ${m.last_name}`} photoUrl={m.user_photo_url} />` statt Plain-Text (user_id und user_photo_url sind bereits im API-Response)

## 9. Verifikation

- [ ] 9.1 Dev-Server starten; Duty-Board aufrufen: Assignee-Chips mit Hover-Tooltip funktionieren; Browser-Network-Tab zeigt `/api/users/:id/contact`-Request nur beim ersten Hover
- [ ] 9.2 Gleiche Person mehrfach sichtbar (z.B. zwei Slots): zweiter Hover löst keinen zweiten Request aus (Cache prüfen)
- [ ] 9.3 Logout und re-Login: Tooltip-Daten werden neu gefetcht (kein Cache-Leak)
- [ ] 9.4 AdminKader: Trainer-Chips mit Tooltip testen
- [ ] 9.5 MembersPage: Member mit Account zeigt Tooltip; Member ohne Account zeigt Plain-Text
- [ ] 9.6 Mobile: Tap öffnet Tooltip; Außen-Tap schließt ihn

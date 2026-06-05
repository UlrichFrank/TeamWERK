## 1. Datenbank-Migration

- [ ] 1.1 Migration `00N_invitation_parent_member_id.up.sql` anlegen: `ALTER TABLE invitation_tokens ADD COLUMN parent_member_id INTEGER REFERENCES members(id) ON DELETE SET NULL`
- [ ] 1.2 Dazugehörige `.down.sql` anlegen: `ALTER TABLE invitation_tokens DROP COLUMN parent_member_id`
- [ ] 1.3 Migration lokal testen: `make migrate-up` und `make migrate-down`

## 2. Backend — Invitations-Endpoint erweitern

- [ ] 2.1 `invitation`-Struct in `internal/auth/handler.go` um `ParentMemberID *int` (`json:"parent_member_id"`) erweitern
- [ ] 2.2 `GET /api/admin/invitations`-Query um `parent_member_id` erweitern (SELECT + Scan)
- [ ] 2.3 Neuen Handler `UpdateInvitationParentMember` implementieren: `PUT /api/admin/invitations/{id}/parent-member` — setzt/löscht `parent_member_id`
- [ ] 2.4 Route in `cmd/teamwerk/main.go` registrieren (Admin-Gruppe)

## 3. Backend — Registrierung erweitern

- [ ] 3.1 In `Register`-Handler: `parent_member_id` aus der `invitation_tokens`-Row lesen (neue Variable `parentMemberID sql.NullInt64`)
- [ ] 3.2 Nach User-Erstellung: wenn `parentMemberID.Valid` → `INSERT OR IGNORE INTO family_links (parent_user_id, member_id) VALUES (?, ?)` ausführen

## 4. Frontend — MemberDetailPage

- [ ] 4.1 `PendingInvitation`-Interface in `MemberDetailPage.tsx` um `parent_member_id?: number | null` erweitern
- [ ] 4.2 `invitations` an `MemberFamilieTab` weitergeben (neuer Prop)
- [ ] 4.3 Handler `handleLinkParentInvitation(invitationId: number | null)` in `MemberDetailPage.tsx` implementieren: ruft `PUT /api/admin/invitations/{id}/parent-member` auf und aktualisiert `invitations`-State

## 5. Frontend — MemberFamilieTab

- [ ] 5.1 Props erweitern: `invitations: PendingInvitation[]` und `onLinkParentInvitation: (invitationId: number | null) => Promise<void>` hinzufügen
- [ ] 5.2 Gemeinsame Liste aufbauen: `linkedParents` (registriert) + Einladungen mit `parent_member_id === memberId` (pending) zusammenführen
- [ ] 5.3 Pending-Einträge in der Liste mit E-Mail + Badge „Einladung ausstehend" anzeigen (statt Name); „Entfernen" ruft `onLinkParentInvitation(null)` für die Einladungs-ID auf
- [ ] 5.4 Zweiten Dropdown für ausstehende Einladungen hinzufügen (gefiltert: noch nicht als parent_member_id dieses Mitglieds gesetzt, noch nicht registriert); nur sichtbar wenn < 2 Einträge gesamt
- [ ] 5.5 „Hinzufügen"-Button für Einladungs-Dropdown: ruft `onLinkParentInvitation(invitationId)` auf
- [ ] 5.6 Styling: brand-Token-Klassen verwenden, kein `bg-gray-*` o.ä.; Badge als `text-xs italic text-brand-text-muted`

# Tasks: Profile Restructure

## Phase 1: Backend erweitern

- [x] `extractFieldValue` in `drafts.go` um `email`, `dsgvo`, `sepa_mandat` erweitern
- [x] `applyDraftToMember` in `drafts.go` um `email`, `dsgvo`, `sepa_mandat` erweitern
- [x] `/api/members` GET: Feld `has_pending_drafts bool` pro Member zurückgeben (Subquery auf member_change_drafts)

## Phase 2: ProfileMemberTab — Neuimplementierung

- [x] Stammdaten-Block: Echte Daten aus `ownMember` anzeigen (Vorname, Nachname, Geb.-Datum, Passnummer, Rückennummer, Position, Status) — alles read-only
- [x] Falls Name-Draft vorhanden: Unter Stammdaten anzeigen → `Angefordert: [Vorname] [Nachname]` + `[Abbrechen]`-Button
- [x] Name-ändern-Block: 2 editierbare Felder (Vorname, Nachname) + `[Änderung anfordern]`-Button → `POST /members/{id}/change-request { field_name: "name", new_value: {...} }`
- [x] IBAN-Block: Echtes IBAN-Feld (editable) + `[Änderung anfordern]` → `POST /members/{id}/change-request { field_name: "iban", new_value: "..." }`
- [x] Falls IBAN-Draft vorhanden: `Angefordert: [IBAN]` + `[Abbrechen]`-Button
- [x] Abbrechen-Button ruft `DELETE /members/{id}/change-drafts/{draftId}` auf und aktualisiert Drafts-Liste
- [x] `ownMember` Prop um `iban?: string` erweitern (kommt aus `/profile/me` → `own_member`)

## Phase 3: MemberDetailPage — Draft-Integration

- [x] State `drafts: ChangeDraft[]` hinzufügen
- [x] Beim Laden: `GET /members/{id}/change-drafts` aufrufen und `drafts` setzen
- [x] Handler `handleDraftAccept(draftId: number)`: `POST /members/{id}/change-drafts/{draftId}/accept` → Drafts neu laden → Member-Daten neu laden
- [x] Handler `handleDraftReject(draftId: number)`: `DELETE /members/{id}/change-drafts/{draftId}` → Drafts neu laden
- [x] Drafts + onDraftAccept + onDraftReject an MemberStammdatenTab, MemberKontaktTab, MemberDatenschutzTab weitergeben

## Phase 4: Tab-Komponenten — Draft-Anzeige

### MemberStammdatenTab
- [x] Name-Draft: Wenn `drafts.find(d => d.field_name === 'name')`, zeige unter Vorname/Nachname: `Angefordert: [Vorname] [Nachname]  [✓ Annehmen] [✗ Ablehnen]`
- [x] `onDraftAccept(draft.id)` und `onDraftReject(draft.id)` wirklich aufrufen

### MemberKontaktTab
- [x] Adresse-Draft: Wenn Draft, zeige `Angefordert: [Straße], [PLZ] [Ort]  [✓] [✗]`
- [x] IBAN-Draft: Wenn Draft, zeige `Angefordert: [IBAN]  [✓] [✗]`
- [x] Props `drafts`, `onDraftAccept`, `onDraftReject` empfangen und verwenden

### MemberDatenschutzTab
- [x] DSGVO-Draft: Wenn Draft, zeige den angeforderten Status + `[✓] [✗]`
- [x] SEPA-Draft: Wenn Draft, zeige angeforderten Status + `[✓] [✗]`
- [x] Props `drafts`, `onDraftAccept`, `onDraftReject` empfangen und verwenden

## Phase 5: MembersPage — ⏳-Indikator

- [x] Member-Interface um `has_pending_drafts?: boolean` erweitern
- [x] In der Tabellen-/Card-Ansicht: ⏳-Icon wenn `has_pending_drafts === true`
- [x] Nur für Admin-User sichtbar

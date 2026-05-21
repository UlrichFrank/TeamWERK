# Tasks: Profile Restructure

## Phase 1: Backend erweitern

- [ ] `extractFieldValue` in `drafts.go` um `email`, `dsgvo`, `sepa_mandat` erweitern
- [ ] `applyDraftToMember` in `drafts.go` um `email`, `dsgvo`, `sepa_mandat` erweitern
- [ ] `/api/members` GET: Feld `has_pending_drafts bool` pro Member zurückgeben (Subquery auf member_change_drafts)

## Phase 2: ProfileMemberTab — Neuimplementierung

- [ ] Stammdaten-Block: Echte Daten aus `ownMember` anzeigen (Vorname, Nachname, Geb.-Datum, Passnummer, Rückennummer, Position, Status) — alles read-only
- [ ] Falls Name-Draft vorhanden: Unter Stammdaten anzeigen → `Angefordert: [Vorname] [Nachname]` + `[Abbrechen]`-Button
- [ ] Name-ändern-Block: 2 editierbare Felder (Vorname, Nachname) + `[Änderung anfordern]`-Button → `POST /members/{id}/change-request { field_name: "name", new_value: {...} }`
- [ ] IBAN-Block: Echtes IBAN-Feld (editable) + `[Änderung anfordern]` → `POST /members/{id}/change-request { field_name: "iban", new_value: "..." }`
- [ ] Falls IBAN-Draft vorhanden: `Angefordert: [IBAN]` + `[Abbrechen]`-Button
- [ ] Abbrechen-Button ruft `DELETE /members/{id}/change-drafts/{draftId}` auf und aktualisiert Drafts-Liste
- [ ] `ownMember` Prop um `iban?: string` erweitern (kommt aus `/profile/me` → `own_member`)

## Phase 3: MemberDetailPage — Draft-Integration

- [ ] State `drafts: ChangeDraft[]` hinzufügen
- [ ] Beim Laden: `GET /members/{id}/change-drafts` aufrufen und `drafts` setzen
- [ ] Handler `handleDraftAccept(draftId: number)`: `POST /members/{id}/change-drafts/{draftId}/accept` → Drafts neu laden → Member-Daten neu laden
- [ ] Handler `handleDraftReject(draftId: number)`: `DELETE /members/{id}/change-drafts/{draftId}` → Drafts neu laden
- [ ] Drafts + onDraftAccept + onDraftReject an MemberStammdatenTab, MemberKontaktTab, MemberDatenschutzTab weitergeben

## Phase 4: Tab-Komponenten — Draft-Anzeige

### MemberStammdatenTab
- [ ] Name-Draft: Wenn `drafts.find(d => d.field_name === 'name')`, zeige unter Vorname/Nachname: `Angefordert: [Vorname] [Nachname]  [✓ Annehmen] [✗ Ablehnen]`
- [ ] `onDraftAccept(draft.id)` und `onDraftReject(draft.id)` wirklich aufrufen

### MemberKontaktTab
- [ ] Adresse-Draft: Wenn Draft, zeige `Angefordert: [Straße], [PLZ] [Ort]  [✓] [✗]`
- [ ] IBAN-Draft: Wenn Draft, zeige `Angefordert: [IBAN]  [✓] [✗]`
- [ ] Props `drafts`, `onDraftAccept`, `onDraftReject` empfangen und verwenden

### MemberDatenschutzTab
- [ ] DSGVO-Draft: Wenn Draft, zeige den angeforderten Status + `[✓] [✗]`
- [ ] SEPA-Draft: Wenn Draft, zeige angeforderten Status + `[✓] [✗]`
- [ ] Props `drafts`, `onDraftAccept`, `onDraftReject` empfangen und verwenden

## Phase 5: MembersPage — ⏳-Indikator

- [ ] Member-Interface um `has_pending_drafts?: boolean` erweitern
- [ ] In der Tabellen-/Card-Ansicht: ⏳-Icon wenn `has_pending_drafts === true`
- [ ] Nur für Admin-User sichtbar

## 1. Backend: bankdaten-Draft

- [x] 1.1 In `internal/members/drafts.go` → `extractFieldValue`: neuen `case "bankdaten"` ergänzen, der `{iban, account_holder}` aus dem Member-Struct zurückgibt
- [x] 1.2 In `internal/members/drafts.go` → `applyDraftToMember`: neuen `case "bankdaten"` ergänzen, der `UPDATE members SET iban=?, account_holder=? WHERE id=?` ausführt
- [x] 1.3 In `internal/members/drafts_handlers.go` → `allowedFields`: `"iban"` und `"account_holder"` entfernen, `"bankdaten"` hinzufügen

## 2. Backend: Typ-Indikatoren in Mitgliederliste

- [x] 2.1 In `internal/members/handler.go` → `Member`-Struct: `HasPendingDrafts bool` ersetzen durch `HasPendingProfilDraft bool` und `HasPendingBankDraft bool` (beide mit `json:"...,omitempty"`)
- [x] 2.2 In `internal/members/handler.go` → Listenhandler: Draft-Query aufteilen in zwei separate Abfragen — eine für `field_name='profil'`, eine für `field_name='bankdaten'` — und die entsprechenden Flags setzen

## 3. Frontend: ProfileBankTab

- [x] 3.1 In `web/src/components/profile/ProfileBankTab.tsx`: `loadDrafts` auf `bankdaten`-Draft umstellen (sucht nach `field_name === 'bankdaten'`, liest `new_value.iban` und `new_value.account_holder`)
- [x] 3.2 In `ProfileBankTab.tsx` → `handleSave`: statt zwei separater Requests einen einzigen `{ field_name: 'bankdaten', new_value: { iban: raw, account_holder: accountHolder } }` senden
- [x] 3.3 In `ProfileBankTab.tsx`: `handleCancel` auf den `bankdaten`-Draft anpassen (löscht den kombinierten Draft)
- [x] 3.4 In `ProfileBankTab.tsx`: `ibanDraft`/`ahDraft` durch einen einzigen `bankdatenDraft` ersetzen; Banner zeigt IBAN und Kontoinhaber aus `bankdatenDraft.new_value`

## 4. Frontend: MemberKontaktTab (Admin)

- [x] 4.1 In `web/src/components/admin/MemberKontaktTab.tsx`: `ibanDraft` und `accountHolderDraft` durch einen einzigen `bankdatenDraft` (field_name === 'bankdaten') ersetzen
- [x] 4.2 In `MemberKontaktTab.tsx`: eine kombinierte Draft-Karte rendern, die IBAN (`bankdatenDraft.new_value.iban`) und Kontoinhaber (`bankdatenDraft.new_value.account_holder`) zusammen zeigt, mit einem gemeinsamen Annehmen/Ablehnen-Button

## 5. Frontend: MembersPage

- [x] 5.1 In `web/src/pages/MembersPage.tsx`: Member-Interface um `has_pending_profil_draft?: boolean` und `has_pending_bank_draft?: boolean` erweitern, `has_pending_drafts` entfernen
- [x] 5.2 In `MembersPage.tsx`: `⏳`-Anzeige ersetzen durch `<User size={14} />` (wenn `has_pending_profil_draft`) und `<CreditCard size={14} />` (wenn `has_pending_bank_draft`) aus `lucide-react`; beide können gleichzeitig erscheinen

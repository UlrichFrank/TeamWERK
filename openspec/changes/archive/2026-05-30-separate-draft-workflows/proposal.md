## Why

Bankdaten-Änderungen (IBAN, Kontoinhaber) werden aktuell als zwei separate Drafts (`iban`, `account_holder`) gespeichert, obwohl sie logisch zusammengehören und gemeinsam von einem Admin genehmigt werden sollten. Zusätzlich zeigt die Mitgliederliste nur ein generisches „hat ausstehende Änderungen"-Signal, ohne zu unterscheiden ob es sich um Persönliche Daten oder Bankdaten handelt.

## What Changes

- **BREAKING** `field_name='iban'` und `field_name='account_holder'` entfallen als separate Draft-Typen; werden durch `field_name='bankdaten'` (enthält `{iban, account_holder}`) ersetzt
- Backend: neuer `case "bankdaten"` in `extractFieldValue` und `applyDraftToMember`
- Backend `GET /api/members`: `has_pending_drafts: bool` → zwei separate Felder `has_pending_profil_draft: bool` und `has_pending_bank_draft: bool`
- Frontend `ProfileBankTab`: sendet einen einzigen `bankdaten`-Draft statt zwei Einzelrequests
- Frontend `MemberKontaktTab` (Admin): zeigt eine kombinierte Bankdaten-Karte statt separate IBAN/Kontoinhaber-Karten
- Frontend `MembersPage`: zeigt `<User>`-Icon (Persönliche Daten) und/oder `<CreditCard>`-Icon (Bankdaten) statt generischem `⏳`-Zeichen

## Capabilities

### New Capabilities

- `bankdaten-draft`: Kombinierter Draft-Typ der IBAN und Kontoinhaber atomar zusammenfasst — ein Admin-Klick genehmigt beide Felder
- `draft-type-indicators`: Separate visuelle Indikatoren in der Mitgliederliste unterscheiden zwischen ausstehenden Persönliche-Daten- und Bankdaten-Änderungen

### Modified Capabilities

Keine bestehenden Spezifikationen betroffen (bisher keine Spec-Dateien für diesen Bereich).

## Impact

- `internal/members/drafts.go`: `extractFieldValue`, `applyDraftToMember`
- `internal/members/drafts_handlers.go`: `allowedFields`-Map
- `internal/members/handler.go`: Member-Struct, Mitglieder-Listenquery
- `web/src/components/profile/ProfileBankTab.tsx`
- `web/src/components/admin/MemberKontaktTab.tsx`
- `web/src/pages/MembersPage.tsx`
- Bestehende `iban`/`account_holder`-Drafts in der DB: müssen vom Admin manuell abgelehnt werden (keine automatische Migration)

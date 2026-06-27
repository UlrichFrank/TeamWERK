# Tasks

Ein Commit pro Task (Conventional Commits).

## 1. Backend

- [x] 1.1 `Member`-Struct in `internal/members/handler.go` um `HasBankData bool \`json:"has_bank_data,omitempty"\`` ergänzen.
- [x] 1.2 `getMember()` in `internal/members/drafts.go` erweitern: `COALESCE(m.sepa_mandat,0)`, `m.sepa_mandat_date` und `(SELECT COUNT(*) FROM member_sensitive WHERE member_id=m.id)>0` abfragen; Scan-Variablen + Zuweisung ergänzen.
- [x] 1.3 Tests: `TestGetProfile_HasBankDataFlag`, `TestGetProfile_SepaMandatFields`, `TestGetProfile_NoBankData` in `internal/members/handler_test.go` (oder neuer Testfile).

## 2. Frontend — ProfilePage Interface

- [x] 2.1 `Member`-Interface in `web/src/pages/ProfilePage.tsx` um `has_bank_data?: boolean`, `sepa_mandat?: boolean`, `sepa_mandat_date?: string` erweitern.

## 3. Frontend — ProfileBankTab

- [x] 3.1 Statusanzeige oben in der Karte: „Bankverbindung: ✓ hinterlegt / – nicht hinterlegt" (aus `has_bank_data`) und „SEPA-Mandat: ✓ hinterlegt (Datum) / – nicht hinterlegt" (aus `sepa_mandat` + `sepa_mandat_date`).
- [x] 3.2 `handleSave` fixen: `encryptBankData({ iban, account_holder })` vor dem POST aufrufen, `{ bank_ciphertext, bank_dek_enc }` als `new_value` senden. Import aus `../../lib/bankCrypto`.
- [x] 3.3 Pending-Draft-Hinweis: „Änderungsanfrage ausstehend" (ohne Details, kein Klartext aus `new_value` lesen) + „Zurückziehen"-Button bleibt.

## 4. Frontend — MemberKontaktTab

- [x] 4.1 Lokalen State `decryptedDraft: { iban: string; account_holder: string } | null` hinzufügen; `useEffect([privateKey, bankdatenDraft])` ruft `decryptBankData(bankdatenDraft.new_value as BankEnvelope, privateKey)` auf und setzt State.
- [x] 4.2 Vault gesperrt + Draft vorhanden: Hinweis rendern „Bankdaten-Antrag liegt vor — Tresor entsperren um einzusehen und anzunehmen (Menü „Tresor")."; nur „Ablehnen"-Button.
- [x] 4.3 Vault entsperrt + Draft vorhanden: `decryptedDraft.account_holder` und `decryptedDraft.iban` anzeigen (statt `bankdatenDraft.new_value?.account_holder`); beide Buttons aktiv.

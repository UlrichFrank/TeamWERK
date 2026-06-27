## Why

Nach der Zero-Knowledge-Migration (Modell B) gibt es zwei UX-Lücken, die zu
stiller Fehlfunktion bzw. schlechter Nutzerführung führen:

**Problem 1 — Profil-Seite (`/profil`, Tab „Bankdaten"):**
Der Member sieht auf seiner eigenen Profilseite nicht, ob eine Bankverbindung
oder ein SEPA-Mandat hinterlegt ist. Das `getMember`-Query fragt diese Felder
gar nicht ab; `clearMemberBank` entfernt den Ciphertext zusätzlich (korrekt,
da Mitglieder die eigenen Bankdaten nicht lesen dürfen). Außerdem sendet
`ProfileBankTab` bei einer Änderungsanfrage noch das Altformat `{iban,
account_holder}` — das Backend erwartet seit Modell B jedoch `{bank_ciphertext,
bank_dek_enc}`, sodass der AcceptDraft-Schritt lautlos nichts tut (Bug C).

**Problem 2 — Mitglieder-Detail (`/mitglieder/:id`, Tab „Kontakt"):**
Liegt ein Bankdaten-Antrag vor, versucht `MemberKontaktTab` `new_value?.iban`
und `new_value?.account_holder` anzuzeigen — die in Modell B jedoch nicht mehr
existieren (`new_value` ist jetzt ein verschlüsselter Envelope). Die
Antragsanzeige ist daher immer leer, und „Annehmen" ist ohne Tresor-Entsperren
möglich, obwohl der Vorstand/Kassierer so keine Möglichkeit hat zu prüfen,
was er annimmt (Bug D + fehlende Vault-Gate).

## What Changes

**Backend:**
- `getMember` (in `drafts.go`, genutzt von `GetProfile`) wird um `sepa_mandat`,
  `sepa_mandat_date` und `has_bank_data` (bool, ob `member_sensitive`-Row
  vorhanden) erweitert. `clearMemberBank` bleibt unverändert (entfernt weiterhin
  Ciphertext); `has_bank_data` ist ein berechnetes Statusfeld, kein Ciphertext.

**Frontend — ProfileBankTab:**
- Zeigt Statusindikatoren: „Bankverbindung: ✓ hinterlegt / – nicht hinterlegt"
  und „SEPA-Mandat: ✓ hinterlegt (DD.MM.YYYY) / – nicht hinterlegt".
- Änderungsformular (Kontoinhaber + IBAN) bleibt erhalten, ruft aber jetzt
  `encryptBankData()` auf und sendet `{bank_ciphertext, bank_dek_enc}` (Fix Bug C).
- Liegt bereits ein offener Antrag vor: Hinweis „Änderungsanfrage ausstehend"
  ohne Details (Member kann seinen eigenen Ciphertext nicht lesen) + Button
  „Zurückziehen".

**Frontend — MemberKontaktTab:**
- Vault gesperrt + Bank-Draft vorhanden: Hinweis „Bankdaten-Antrag liegt vor —
  Tresor entsperren um einzusehen und anzunehmen." Nur „Ablehnen" aktiv;
  „Annehmen" ist nicht sichtbar/aktiv (Variante II: Tresor zwingend).
- Vault entsperrt + Bank-Draft vorhanden: `decryptBankData(draft.new_value,
  privateKey)` → entschlüsseltes Kontoinhaber/IBAN anzeigen → „Annehmen" aktiv.
- Fix Bug D: `new_value` wird als `BankEnvelope` interpretiert, nicht als
  Klartext-Objekt.

**Kein Migrations-SQL**, keine neuen Routen, keine Breaking Changes an
bestehenden API-Contracts.

## Scope

In scope:
- `getMember` Erweiterung (sepa_mandat, sepa_mandat_date, has_bank_data)
- `ProfileBankTab`: Statusanzeige + encryptBankData-Fix
- `MemberKontaktTab`: Vault-Gate + Draft-Entschlüsselung für Anzeige
- `ProfilePage.tsx`: Member-Interface um neue Felder erweitern

Out of scope:
- Änderung der SEPA-Mandat-Upload-Logik
- Änderung des Vault-Setup-Flows (TresorPage)
- Anzeige entschlüsselter Bankdaten im Profil (Member liest eigene Daten nicht)

## Test-Anforderungen

| Route / Komponente | Testname | Erwartetes Ergebnis |
|---|---|---|
| `GET /api/profile/me` | `TestGetProfile_HasBankDataFlag` | `has_bank_data: true` wenn `member_sensitive`-Row existiert |
| `GET /api/profile/me` | `TestGetProfile_SepaMandatFields` | `sepa_mandat`, `sepa_mandat_date` korrekt befüllt |
| `GET /api/profile/me` | `TestGetProfile_NoBankData` | `has_bank_data: false` wenn keine `member_sensitive`-Row |
| `POST /api/members/:id/change-request` (bankdaten) | `TestCreateBankChangeRequest_EnvelopeFormat` | Backend akzeptiert `{bank_ciphertext, bank_dek_enc}` und legt Draft korrekt an |

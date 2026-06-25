# Tasks

Reihenfolge ist migrationskritisch: erst neue Schreib-/Lesepfade (Envelope) +
Tresor-Einrichtung, dann clientseitige Erstverschlüsselung des `v1:`-Bestands, **zuletzt**
Entfernen des serverseitigen Decrypts und `FIELD_ENCRYPTION_KEY`. Ein Commit pro Task
(Conventional Commits).

## 1. Krypto-Core & Schema

- [x] 1.1 `web/src/lib/crypto.ts` aus `origin/encryption` portieren (PBKDF2 600k → AES-KW,
  AES-GCM-256, wrap/unwrap, Key-Check, Salt). Unit-Tests mit bekannten Testvektoren.
- [x] 1.2 Krypto-Core um **binäre Blobs** erweitern (`encryptBytes`/`decryptBytes` für
  SEPA-Mandat-PDFs, Magic-Header analog). Roundtrip-Test.
- [x] 1.3 Migration `internal/db/migrations/008_member_sensitive.{up,down}.sql`:
  `member_sensitive(member_id PK FK→members ON DELETE CASCADE, ciphertext, dek_enc_vorstand)`
  + `ALTER TABLE clubs ADD COLUMN vorstand_kdf_salt`, `vorstand_key_check`. Nächste freie
  Nummer prüfen.

## 2. Tresor-Einrichtung & Rotation

- [x] 2.1 Backend `GET/PUT /api/admin/encryption-config` (Salt + Key-Check speichern; 409 bei
  bereits vorhandener Config; Gate vorstand/kassierer/admin). Tests: 204, 409, 400, 403.
- [x] 2.2 Backend `PUT /api/admin/rotate-encryption` (neuer Salt + Key-Check + Batch
  re-gewrappter DEKs atomar schreiben). Tests: 204 (Bestand bleibt lesbar), Rotation.
- [x] 2.3 Frontend `VaultContext` (sessionStorage `vk`, 30-min-Inaktivität, Key-Caching;
  hält nur den AES-KW-Wrapping-Key) — **neu** gegen heutige Standards.
- [x] 2.4 Frontend `TresorPage` (`/tresor`): Einrichtung + Entsperren inkl. expliziter
  **Datenverlust-Warnung** (kein Recovery, Zwei-Personen-Regel); Route + Nav (RoleRoute +
  policy.NavItem + AppShell).
- [ ] 2.5 Frontend Rotation: alle DEKs laden, mit neuer Passphrase re-wrappen, Batch posten;
  `vk` aktualisieren. **Mit Sektion 3 umgesetzt** — braucht DEK-Bestand + DEK-Listen-Endpoint
  (existiert erst, wenn Bankdaten verschlüsselt geschrieben werden).

## 3. Bankdaten-Schreib-/Lesepfade auf Envelope umstellen

- [ ] 3.1 `internal/members` (`bank_crypto.go`, `handler.go`): `PUT /members/{id}/bank-details`
  nimmt Ciphertext + `dek_enc_vorstand` entgegen, lehnt Klartext/fehlenden Wrap mit 400 ab;
  Lesepfade liefern nur Blob + Wrap. Server-`crypto.Decrypt`-Aufrufe entfernen. Tests:
  200 (Envelope), 400 (Klartext), 403 (unberechtigt).
- [ ] 3.2 `internal/members/drafts.go`: `member_change_drafts(field_name='bankdaten')` als
  Group-Blob (kein Server-Decrypt). Tests Happy/Fehlerfall.
- [ ] 3.3 `internal/config` (Vereins-SEPA-Stammdaten): `clubs.iban/bic/glaeubiger_id/
  kontoinhaber` als **ein** Group-Blob speichern/ausliefern (kein Server-Decrypt). Tests.
- [ ] 3.4 `internal/upload`: SEPA-Mandat-PDF als clientseitig verschlüsselter Blob
  hochladen/ausliefern (keine Server-`DecryptBytes`). Tests.
- [ ] 3.5 Frontend Bankdaten-Eingabe (Profil/Mitglied/Verein/Mandat-Upload): vor dem Senden
  clientseitig verschlüsseln; entsperrter Tresor erforderlich (sonst Dialog).

## 4. Fee-Run clientseitig

- [ ] 4.1 `sepa/iban.go`-Logik nach TS portieren (`web/src/lib/iban.ts`) inkl. mod97;
  Tests mit denselben Vektoren wie Go.
- [ ] 4.2 Clientseitiger `pain.008.001.08`-Builder in TS; Goldfile-Test gegen die heutige
  Go-XML-Ausgabe (fachliche Parität: ein PmtInf, RCUR, ReqdColltnDt, Verwendungszweck).
- [ ] 4.3 Backend `POST /api/fee-run/export-data` (Ciphertext + Group-Wraps + Beträge +
  Verwendungszweck-Bausteine; **keine** Klartext-IBAN). Alter `POST /export` (Server-XML)
  entfällt. Tests: 200 (nur Ciphertext), 403.
- [ ] 4.4 Frontend Fee-Run-Seite: Blobs entschlüsseln, IBANs validieren, `iban_fehlt`/
  `iban_ungueltig` clientseitig ergänzen, XML lokal zum Download. `confirm`/`protocol`
  bleiben unverändert (keine IBANs).
- [ ] 4.5 `internal/beitragslauf` server-seitigen XML-Builder + IBAN-Decrypt entfernen;
  `preview` liefert weiterhin Nicht-IBAN-Ausschlüsse.

## 5. Serverseitigen Decrypt abbauen

- [ ] 5.1 `internal/policy/bankdata.go` (`CanDecryptBankData`) + alle Aufrufer entfernen;
  Architektur-/Build-Tests grün.
- [ ] 5.2 `internal/crypto`: Bank-/SEPA-Decrypt-Pfade aus regulären Routen entfernen; nur
  noch Migrations-Brücke (Abschnitt 6) nutzt den Schlüssel.

## 6. Migration des Bestands

- [ ] 6.1 Temporärer, gegateter Migrations-Endpoint, der `v1:`-/Klartext-Bestand über die
  Brücke (`FIELD_ENCRYPTION_KEY`) entschlüsselt und dem Kassierer-Browser über TLS
  ausliefert (nur bei gesetztem Brücken-Schlüssel verfügbar). Tests: Gate, „nur wenn Bridge".
- [ ] 6.2 Frontend Migrations-Seite: Bestand laden, clientseitig zu Envelope verschlüsseln,
  hochladen; idempotent (bereits migrierte überspringen), Fortschrittsanzeige.
- [ ] 6.3 Nach Vollmigration: Migrations-Endpoint + `FIELD_ENCRYPTION_KEY` entfernen;
  Startup-Check anpassen (Server startet ohne Schlüssel). Test: Start ohne Key OK.

## 7. SSE, Doku, Verifikation

- [ ] 7.1 Mutations-Routen (encryption-config, rotate, bank-details, club, mandat) rufen
  `h.hub.Broadcast(...)`; betroffene Seiten abonnieren `useLiveUpdates`.
- [ ] 7.2 `docs/agent/03-go.md` + `10-deployment.md` aktualisieren (Zero-Knowledge-Modell,
  Tresor-Einrichtung/Rotation, kein `FIELD_ENCRYPTION_KEY` mehr nach Migration,
  Bedrohungsmodell-Grenze, kein Recovery).
- [ ] 7.3 `/verify-change` + `openspec validate --strict`; Proposal nach Apply archivieren.

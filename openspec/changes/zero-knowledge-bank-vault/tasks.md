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
- [x] 2.5 Frontend Passphrase-Rotation in `TresorPage` (Modell B, O(1)): privaten Schlüssel
  mit neuer Passphrase neu verschlüsseln (`rewrapPrivateKeyForRotation`) und `rotate-encryption`
  posten — kein DEK-Listen-Endpoint nötig (DEKs/Public-Key unverändert). Keypair-Rotation (O(n))
  vom Backend unterstützt, UI dafür bei Bedarf später.

## 3. Bankdaten-Schreib-/Lesepfade auf Envelope umstellen

- [x] 3.1 `internal/members`: `PUT /members/{id}/bank-details` nimmt Envelope
  (`bank_ciphertext` + `bank_dek_enc`) → `member_sensitive` (Upsert/Delete), lehnt
  Klartext-IBAN mit 400 ab; `GET /members/{id}` liefert den Envelope nur an
  vorstand/kassierer/admin (kein Server-Decrypt). Profil/Kind-Profil liefern G2-konform
  KEINE Bankdaten mehr (`clearMemberBank`). Tests: Envelope-Speicherung, Klartext-400,
  Trainer-403, Eigentümer/Eltern lesen nichts. Permission-Matrix ergänzt.
- [x] 3.2 `internal/members/drafts.go` (Backend): `bankdaten`-Draft trägt den clientseitigen
  Envelope; Server ver-/entschlüsselt nicht mehr (kein crypto.Encrypt/Decrypt), `old_value`
  = null, Reveal reicht den Envelope durch, Annehmen schreibt nach `member_sensitive`. Test:
  Draft→Accept→member_sensitive. **Offen (Browser):** ProfileBankTab/MemberKontaktTab
  (Envelope erzeugen/anzeigen).
- [x] 3.3 (Backend) `internal/config`: Vereins-SEPA als **ein** Envelope
  (`clubs.sepa_ciphertext/sepa_dek_enc`); UpdateClub/GetClub speichern/liefern den Envelope,
  lehnen Klartext-SEPA mit 400 ab; kein Server-Decrypt (encClubField/decClubField + Regex-
  Validierung entfernt → clientseitig). Test umgestellt. **Frontend:** VereinTab ver-/
  entschlüsselt SEPA (Tresor entsperrt), sperrt SEPA-Felder ohne Tresor (tsc/lint grün;
  Browser-Verifikation offen).
- [x] 3.4 `internal/upload`: SEPA-Mandat-PDF clientseitig verschlüsselt. Upload nimmt
  Ciphertext-Blob (`crypto.IsClientEncryptedBytes`-Magic) + gewrappten `dek_enc`,
  speichert beides roh; Download-Token liefert `dek_enc`, Download streamt den Blob (kein
  Server-Decrypt). Migration `members.sepa_mandat_dek_enc`. Frontend (`MemberKontaktTab`):
  `encryptFile`/`decryptFile`, Öffnen braucht entsperrten Tresor. Go-Test umgestellt;
  tsc/lint grün; Browser-Verifikation offen.
- [~] 3.5 Frontend Bankdaten-Eingabe — **MemberDetailPage + MemberKontaktTab (PDF) erledigt**;
  Beitritts-Formular **entfällt** (RequestMembership erfasst keine Bankdaten).
  **CSV-Bereinigung erledigt:** Der serverseitige CSV-Import schreibt keine Bankdaten mehr
  (kein toter-Spalten-Schreibpfad); IBAN/Kontoinhaber-Spalten lösen nur einen Hinweis aus,
  Bankdaten werden pro Mitglied im Tresor erfasst. **Bewusst zurückgestellt:** voller
  client-seitiger CSV-Bank-Bulk-Import (G5, Browser parst CSV → verschlüsseln → Matching →
  Envelope-Upload) als fokussierter Folgeschritt + Browser-Verifikation.
  Frontend-Crypto-Flows brauchen Browser-Verifikation.

## 4. Fee-Run clientseitig

- [x] 4.1 IBAN-Logik nach TS portiert (`web/src/lib/sepa.ts`: normalizeIBAN/isValidIBAN, mod97);
  Test spiegelt die Go-Vektoren (`sepa.test.ts`).
- [x] 4.2 Clientseitiger `pain.008.001.08`-Builder (`web/src/lib/sepaXml.ts`); Ausgabe
  **byte-identisch** zur Go-Implementierung verifiziert (diff gegen sampleInput); Tests in
  `sepaXml.test.ts` (ein PmtInf, RCUR, ReqdColltnDt, Verwendungszweck, IBANs).
- [x] 4.3 Backend `POST /api/fee-run/export-data` (nur Ciphertext + Wraps + nicht-geheime
  Felder; **keine** Klartext-IBAN). Alter `POST /export` entfernt. Tests: 200 (nur Envelope),
  400 (ohne Vereins-SEPA).
- [x] 4.4 Frontend Fee-Run-Seite (`BeitragslaufPage`): XML-Download holt `export-data`,
  entschlüsselt Vereins-SEPA + Mitglieds-Envelopes mit dem Tresor-Schlüssel, validiert IBANs
  clientseitig (ungültige übersprungen + gemeldet), `buildPainXML` → lokaler Download;
  erfordert entsperrten Tresor. `confirm`/`protocol` unverändert. tsc/lint grün;
  **Browser-Verifikation offen**.
- [x] 4.5 `internal/beitragslauf` Server-XML-Builder (`xml.go`) + IBAN-/Club-Decrypt entfernt;
  `preview` liefert weiterhin Nicht-IBAN-Ausschlüsse (+ `iban_fehlt` bei fehlendem Envelope).

## 5. Serverseitigen Decrypt abbauen

- [x] 5.1 `policy.CanDecryptBankData` + `bankdata.go`/-test entfernt; einziger Aufrufer
  (Draft-Reveal) gated inline auf die Finance-Gruppe. Tote `iban`/`account_holder`-Draft-
  Cases + `encBankField` (schrieben in die tote `members.iban`) entfernt. Build/Test/Lint grün.
- [x] 5.2 Keine regulären Routen entschlüsseln mehr Bank-/SEPA-PII server-seitig (members,
  config, beitragslauf, drafts auf Envelope umgestellt). `internal/crypto`
  (`Decrypt`/`DecryptBytes`, `FIELD_ENCRYPTION_KEY`) bleibt **nur** für die Migrations-Brücke
  (Sektion 6) + Auslieferung von Legacy-server-verschlüsselten Mandat-Dateien bis zur Migration.

## 6. Migration des Bestands (zwei Deploys, minimales Brücken-Fenster)

**Strategie (siehe Design D7):** Das sicherheitskritische, irreversible Fenster ist die Zeit,
in der der Server gleichzeitig den Brücken-Schlüssel (`FIELD_ENCRYPTION_KEY`) **und** einen
`v1:`-Klartext über TLS ausliefernden Endpoint hält. Um es zeitlich minimal und maximal
automatisiert zu halten, wird die **nicht-destruktive Startup-Toleranz** (Server startet auch
ohne Schlüssel) in den **ersten** Deploy (Branch A) **vorgezogen**. Der kritische Moment ist
dann kein Build+Deploy-Zyklus mehr, sondern eine sekundenschnelle, skriptbare Ops-Aktion
(`make zk-finalize-remote`: Schlüssel aus `env` entfernen + Restart). Der eigentliche
Code-Abbau (Branch B) folgt als reine Hygiene jederzeit später.

```
1. Branch A deployen (make deploy). FIELD_ENCRYPTION_KEY bleibt gesetzt.   [reversibel]
2. Tresor-Inhaber: /admin/migration im Browser → Migration läuft (Minuten).
3. make zk-finalize-remote → prüft complete, entfernt Key, Restart.        [kritisch, ~Sek.]
4. Branch B deployen — Endpoint/Brücke/Legacy-Spalten weg.                 [Hygiene, jederzeit]
```

### Branch A — `feat/zk-migrate-bestand` (von `feat/zero-knowledge-bank-vault`, reversibel)

- [x] 6.1 **Startup-Toleranz vorziehen** (`refactor(crypto)`): `crypto.HasKey()`;
  `cmd/teamwerk/main.go` startet mit **und ohne** `FIELD_ENCRYPTION_KEY` (`slog.Warn` statt
  `fatal` bei fehlendem Key — Brücke/Migration dann deaktiviert; `fatal` nur noch bei
  **gesetztem, aber ungültigem** Key). Sicher, da alle regulären Routen envelope-only sind.
  Test: Start ohne Key OK; Start mit ungültigem Key bricht weiter ab.
- [ ] 6.2 **Gegateter Brücken-Endpoint** — neues Package `internal/migration` (importiert nur
  `database/sql` + `internal/crypto` + Upload-Dir → arch-test-konform; in `arch_test.go`
  klassifizieren). Routen in der Finance-Gruppe (`router.go`, vorstand/kassierer):
  - `GET /api/admin/migrate-legacy/status` → `{bridge_available, pending_members, pending_club,
    pending_mandates, complete}` (`bridge_available = crypto.HasKey()`).
  - `GET /api/admin/migrate-legacy/data` → entschlüsselt über die Brücke **nur noch
    nicht-migrierte** Datensätze (`members.iban/account_holder`, `clubs.*`-SEPA falls noch
    `v1:`, Mandat-PDF-Bytes); **404, wenn `!HasKey()`** („nur wenn Bridge").
  - `POST /api/admin/migrate-legacy/upload` → nimmt Envelopes; **pro Datensatz in einer
    Transaktion**: Envelope-Spalte(n) schreiben **und** Legacy-`v1:`-Spalte nullen (Mandat:
    Datei auf Client-Magic `TWENC1\n` umschreiben + `members.sepa_mandat_dek_enc` setzen) →
    idempotent + self-disabling; Mandat-Blob via `crypto.IsClientEncryptedBytes` erzwungen.
    `h.hub.Broadcast("members")`/`"settings"`.
  - Tests: Trainer-403 (Gate); `data`/`upload` 404 wenn `!HasKey()` („nur wenn Bridge");
    `data` liefert entschlüsselten `v1:`-Klartext; `upload` schreibt Envelope **und** nullt die
    Legacy-Spalte; Re-Run idempotent (`status.complete`, leeres `data`).
- [ ] 6.3 **Frontend-Migrationsseite** (`/admin/migration`, RoleRoute vorstand/kassierer,
  `policy.NavItem` + AppShell-Nav, `useLiveUpdates`): **erfordert entsperrten Tresor**
  (Safety-Gate — Passphrase muss vor dem Brücken-Abbau nachweislich funktionieren). Flow:
  `status` → `data` → je Datensatz clientseitig Envelope (`bankCrypto.ts`/`crypto.ts`,
  Wrap an Group-Public-Key; Mandate via `encryptFile`) → Batch-`upload`; **Fortschrittsanzeige**;
  idempotent (Re-Run lädt nur den Rest); Auto-Fertig bei `status.complete`. tsc/lint grün,
  Browser-Verifikation offen (wie übrige ZK-Flows).
- [ ] 6.4 **Ops-Automation** `make zk-finalize-remote`: ruft `…/migrate-legacy/status` und
  **bricht ab, wenn nicht `complete`**; entfernt dann die `FIELD_ENCRYPTION_KEY`-Zeile aus
  `/etc/teamwerk/env` und `systemctl restart teamwerk` (Muster wie bestehende `*-remote`-
  Targets). **DB-Backup als Vorbedingung** (irreversibel ab dem Spalten-Nullen in 6.2).

### Branch B — `feat/zk-remove-bridge` (von Branch A, Hygiene NACH erfolgter Migration)

- [ ] 6.5 **Brücke + Endpoint abbauen**: `internal/migration` + Routen + main.go-Verdrahtung
  entfernen; `internal/crypto` auf `IsClientEncryptedBytes`/`clientFileMagic` reduzieren
  (`Decrypt`/`DecryptBytes`/`Encrypt`/`EncryptBytes`/`InitFromEnv`/`activeKey` weg);
  `crypto.InitFromEnv()`-Aufruf aus `main.go` streichen; Legacy-Mandat-Download auf reines
  Streamen umstellen. Test: Server startet ohne `FIELD_ENCRYPTION_KEY`.
- [ ] 6.6 **Legacy-Spalten droppen**: Migration `009_drop_legacy_bank_columns.{up,down}.sql`
  (nächste freie Nummer nach 008) — `members.iban/account_holder`,
  `clubs.glaeubiger_id/iban/bic/kontoinhaber` droppen; `down` legt sie als nullable `TEXT`
  wieder an (Daten **nicht** wiederherstellbar — dokumentieren).

## 7. SSE, Doku, Verifikation

- [x] 7.1 Mutations-Routen rufen `h.hub.Broadcast(...)`: encryption-config/rotate/club →
  `"settings"` (TresorPage/VereinTab abonnieren), bank-details/child-bank/mandat →
  `"members"`. Verifiziert.
- [x] 7.2 `docs/agent/03-go.md`, `10-deployment.md`, `06-gotchas.md` auf das Zero-Knowledge-
  Modell aktualisiert (Keypair/Tresor, kein Server-Decrypt, Bedrohungsmodell-Grenze, kein
  Recovery, Migration als geplanter Rollout-Schritt, `export-data`/clientseitiger Fee-Run).
- [ ] 7.3 `openspec validate --strict` grün; `/verify-change` + Archivierung **nach** Apply/
  Migration (Sektion 6).

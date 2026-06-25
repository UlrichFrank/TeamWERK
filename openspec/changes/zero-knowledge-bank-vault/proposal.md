## Why

Aktuell (ausgerollter Zustand aus `encrypt-bank-sepa-at-rest`) liegt die Bank-/SEPA-PII
zwar AES-256-GCM-verschlüsselt at-rest (`"v1:"`), aber der **Server hält den
Schlüssel** (`FIELD_ENCRYPTION_KEY`) und entschlüsselt zentral autorisiert
(`policy.CanDecryptBankData`). Damit kann jeder, der den laufenden Prozess oder die
Umgebung sieht — und insbesondere die App-Rolle `admin` (umgeht alle Checks) — den
Klartext lesen. Gegen ein **gestohlenes DB-Backup / einen Disk-Snapshot / einen
neugierigen Hoster** schützt das nur, solange der Schlüssel nicht im selben Zugriff
liegt; gegen den **App-Admin** schützt es gar nicht.

Ziel ist ein **Zero-Knowledge-Modell „at rest"**: Bank-/SEPA-Daten werden
**clientseitig** verschlüsselt, der Server speichert **nur Ciphertext + gewrappte
Schlüssel** und besitzt **keinen** Entschlüsselungsschlüssel mehr. Lesen dürfen nur
der **Eigentümer** (das verknüpfte Mitglied) sowie **Vorstand + Kassierer** über
einen gemeinsamen Finance-Group-Key — **niemand sonst, auch nicht `admin` und nicht
der Server**.

**Bewusste Modellgrenze (ehrlich dokumentiert):** Da der Server die React/JS-App
ausliefert, ist er per Definition Teil der Trust-Base. Dieses Modell schützt gegen
**passive** Kompromittierung (Backup-Diebstahl, Disk-Snapshot, neugieriger Hoster,
lesender Admin), **nicht** gegen einen **aktiv bösartigen Server** (ausgeliefertes
Trojaner-JS) oder ein **kompromittiertes Endgerät / Phishing** des Kassierers. Das
entspricht der Grenze, die ProtonMail/Bitwarden-Web-Vaults tragen. Ein gegen den
Server-Betreiber kryptografisch unangreifbares Modell (Native-App/Browser-Extension
aus einem Store) wurde geprüft und als unverhältnismäßig verworfen (siehe `design.md`,
„Path B"), zumal der Verein den VPS **selbst** betreibt.

## What Changes

- **Envelope-Encryption statt App-Schlüssel:** Pro Mitglied ein zufälliger
  Data-Key (DEK); die Bankdaten werden mit dem DEK AES-GCM-verschlüsselt. Der DEK
  wird mit dem **Finance-Gruppenschlüssel** gewrappt (`dek_enc_vorstand`). Die
  DEK-Schicht entkoppelt die Passphrase-Rotation vom Datenbestand (Rotation = DEKs
  neu wrappen, kein Blob-Neuencrypt).
- **Geteilte Finance-Gruppen-Passphrase (entschieden):** Eine gemeinsame
  Tresor-Passphrase für alle `vorstand`/`kassierer`, **separat vom Login-Passwort**,
  die den Browser nie verlässt. `vorstand_key = PBKDF2(passphrase, salt)` (WebCrypto,
  600 000 Iter.). Server speichert nur Salt + Key-Check-Wert. **Kein** Pro-Person-
  Keypair, **keine** Split-Key-Derivation (für Path A unnötig, siehe `design.md`).
- **Kein Eigentümer-Selbstlesen (entschieden):** Es liest **ausschließlich** die
  Finance-Gruppe. Mitglieder/Eltern geben Bankdaten ein (group-wrap) und können sie
  **nicht** zurücklesen. Damit ist die Krypto vollständig vom Login/Passwort-Reset
  entkoppelt (kein `dek_enc_member`, kein Change-Password-Re-Wrap). Eltern verlieren
  das heutige Lese-Recht (`…∨ Elternteil`); auch der Eigentümer liest nicht mehr selbst.
- **Rollenwechsel = Passphrase-Rotation:** Aufnahme/Entzug erfolgt über Neu-Vergabe
  der geteilten Passphrase + clientseitiges Re-Wrap aller DEKs durch einen aktuellen
  Halter (der Server kann das nicht). **Kein serverseitiges Recovery** — Passphrase-
  Verlust = Datenverlust (bewusst, Zwei-Personen-Regel + Warnung bei Einrichtung).
- **Fee-Run komplett clientseitig:** IBAN-Validierung, Ausschlusslogik,
  `pain.008`-XML-Erzeugung und SEPA-Mandat-PDF-Erzeugung wandern in den Browser des
  Kassierers. Der Server wird zum **blinden Blob-Store** für Bank-/SEPA-PII.
- **Server hört auf zu entschlüsseln:** `policy.CanDecryptBankData` und alle
  serverseitigen `crypto.Decrypt`-Lesepfade für Bank-/SEPA-Felder entfallen;
  `FIELD_ENCRYPTION_KEY` überlebt **nur** als einmalige Migrations-Brücke und wird
  danach entfernt.
- **Migration:** Da alles mindestens an den Group-Key gewrappt wird und der
  Kassierer diesen hält, migriert **der Browser des Kassierers** den gesamten
  `v1:`-Bestand in einem Lauf (entschlüsseln über die noch vorhandene Server-Brücke
  → an Group re-wrappen → hochladen → Server-Schlüssel löschen).

## Capabilities

### New Capabilities
- `client-side-bank-encryption`: Clientseitige Envelope-Verschlüsselung der
  Bank-/SEPA-PII mit Finance-Group-Key, Nutzer-Schlüsselpaaren, Recovery und
  Rollen-Re-Wrap. Server speichert ausschließlich Ciphertext + gewrappte Schlüssel.

### Modified Capabilities
- `bank-data-at-rest-encryption`: Die Anforderung „Zentrale Autorisierung der
  Entschlüsselung" (`policy.CanDecryptBankData`) und „Eigentümer-/Eltern-Lesen über
  Server-Endpoints" werden **entfernt** (Server entschlüsselt nicht mehr).
  „App-gehaltener Schlüssel" wird auf **Migrations-Brücke (zeitlich begrenzt)**
  reduziert. Format-/Decrypt-Semantik bleibt nur für die Migrationsdauer relevant.
- `sepa-beitragslauf`: Der `pain.008`-Export wird vom serverseitigen Builder auf
  einen **clientseitigen** Builder im Browser des Kassierers umgestellt.

## Impact

- **Code (Go):** `internal/crypto` (Bank-/SEPA-Pfade entfallen, nur Migrations-Brücke
  bleibt), `internal/policy/bankdata.go` (entfällt), `internal/members` (bank_crypto,
  drafts: kein Server-Decrypt mehr; speichern Blobs+Wraps), `internal/config`
  (Vereins-SEPA als Group-Blob + Tresor-Setup/Rotation-Endpoints), `internal/beitragslauf`
  (Server-XML-Builder entfällt → `export-data` liefert nur Blobs aus), `internal/upload`
  (Mandat-PDFs als Blobs). **Kein** Auth-Umbau (Krypto vom Login entkoppelt).
- **Code (Frontend):** Krypto-Core **portiert aus `origin/encryption:web/src/lib/crypto.ts`**
  (WebCrypto, bin. Blobs für PDF ergänzen); Vault-UI **neu** gegen heutige
  Komponenten-Standards (Passphrase-Dialog, Gate, Setup/Rotation); IBAN-Validierung
  (Port `sepa/iban.go` → TS); clientseitiger `pain.008`-Builder; Mandat-PDF;
  Fee-Run-Seite; Bankdaten-Eingabe.
- **Datenbank:** Neue Tabelle `member_sensitive(member_id, ciphertext, dek_enc_vorstand)`
  + `clubs.vorstand_kdf_salt`, `clubs.vorstand_key_check` (Schema-Muster aus dem Branch,
  ohne `dek_enc_member`/`member_salt`). Migration des `v1:`-Bestands.
- **Auth:** **Unverändert** — die Tresor-Passphrase ist vom Login getrennt.
- **Betrieb:** `FIELD_ENCRYPTION_KEY` wird nach erfolgreicher Migration entfernt;
  Backup enthält dann nur noch unlesbare Blobs. Recovery-Code des Group-Keys muss
  **physisch** sicher beim Vorstand liegen — **Verlust = Datenverlust aller
  Bank-/SEPA-Felder**.
- **Bedrohungsmodell:** schließt Backup-/Snapshot-Diebstahl, neugierigen Hoster und
  lesenden Admin; schützt **nicht** gegen aktiv bösartigen Server oder
  Endgeräte-/Phishing-Kompromittierung (dokumentierte Grenze).
- **Kompatibilität:** Kein Zero-Downtime — Migration ist eine koordinierte Aktion
  (alle Mitglieder brauchen ein Schlüsselpaar bzw. werden group-only migriert).

## Test-Anforderungen

Detaillierung erfolgt mit den Specs/Tasks. Mindestumfang (Route → Testname →
erwarteter Status, garantierte Invariante):

- `PUT /api/admin/encryption-config` (Tresor-Einrichtung): Happy-Path 200 speichert
  Salt + Key-Check; bei bereits vorhandener Konfiguration 409; ohne Berechtigung 403.
  *Invariante:* Server speichert nie die Passphrase, nur Salt + Key-Check.
- `PUT /api/members/{id}/bank-details` (Blob + Group-Wrap speichern): Happy-Path 200 mit
  Envelope-Format (Ciphertext + `dek_enc_vorstand`); nicht-berechtigter Schreiber 403;
  Klartext-Wert / fehlender Wrap 400. *Invariante:* Server sieht nie Klartext-IBAN.
- `GET …` Bank-Blob-Auslieferung: liefert nur Ciphertext + Group-Wrap, **nie** Klartext.
  *Invariante:* kein serverseitiger Decrypt-Pfad in regulären Routen.
- Rotation `PUT /api/admin/rotate-encryption`: re-wrappt alle DEKs mit neuem Salt +
  Key-Check; ohne Berechtigung 403. *Invariante:* nach Rotation sind alle DEKs nur mit
  der neuen Passphrase entschlüsselbar; Bestand bleibt vollständig lesbar.
- `POST /api/fee-run/export-data`: liefert nur Ciphertext + Group-Wraps + Beträge,
  **nie** Klartext-IBAN; ohne Berechtigung 403. *Invariante:* die `pain.008`-Datei
  entsteht ausschließlich clientseitig.
- Migration `v1:` → Envelope: nach Lauf trägt jeder Bestandswert das Envelope-Format
  und ist serverseitig **nicht** mehr entschlüsselbar (`FIELD_ENCRYPTION_KEY` entfernt).

## Context

Sensible Mitgliedsdaten (IBAN, Adresse, Geburtsdatum) liegen im Klartext in der SQLite-Datenbank. Der Hauptentwickler hat SSH-Zugang zum VPS und kann die DB-Datei direkt lesen. Das primäre Risiko ist kein adversarialer Angreifer, sondern ein übersehener Bug in der Anwendung (fehlende Auth-Middleware, zu breites SELECT, Logging-Artefakt), der sensitive Felder exponiert.

Ziel ist Envelope-Verschlüsselung mit Client-seitiger Entschlüsselung: Der Server speichert und überträgt ausschließlich Ciphertext. Selbst bei einem vollständigen DB-Dump oder einem Auth-losen API-Endpoint sind die Daten wertlos ohne den im Browser gehaltenen Schlüssel.

Betroffene Felder: `date_of_birth`, `street`, `zip`, `city`, `iban`, `account_holder`.

## Goals / Non-Goals

**Goals:**
- Server speichert und verarbeitet sensible Felder ausschließlich als Ciphertext
- Vorstand-Mitglieder können alle Mitgliedsdaten über eine geteilte Passphrase entschlüsseln
- Mitglieder mit verknüpftem User-Account können ihre eigenen Daten via Login-Passwort entschlüsseln
- CSV-Export läuft vollständig im Browser (kein Server-Decrypt)
- Passphrase-Rotation für den Vorstand ohne Datenverlust

**Non-Goals:**
- Schutz vor einem Angreifer mit aktivem Zugang zum laufenden Prozess (RAM-Dump)
- Schutz der unverschlüsselten Felder (Name, Status, Sportzeugs)
- Server-seitige Suche/Filterung über verschlüsselte Felder
- Ende-zu-Ende-Verschlüsselung von E-Mails oder Anhängen

## Decisions

### 1. Client-seitige Entschlüsselung (kein Server-Decrypt)

**Entscheidung:** Der Server gibt Ciphertext zurück. Entschlüsselung passiert ausschließlich im Browser via WebCrypto API.

**Alternativen:**
- Server entschlüsselt bei authentifizierten Anfragen → Key muss auf dem Server liegen → Admin/Provider kann Key aus env lesen; ein Bug in einer Route exponiert Klartext
- Hardware Security Module (HSM) → nicht verfügbar auf IONOS VPS Linux XS

**Rationale:** Nur Client-Decrypt schützt gegen die definierten Bedrohungen (Auth-Bug, zu breites SELECT). Der Mehraufwand im Frontend ist vertretbar.

---

### 2. WebCrypto API statt externer Bibliothek

**Entscheidung:** Ausschließlich `window.crypto.subtle` (Browser-nativ). Keine npm-Abhängigkeit.

**Alternativen:**
- libsodium.js (XSalsa20-Poly1305) → bessere Ergonomie, aber ~400 KB WASM
- tweetnacl → 7 KB, aber kein Argon2; müsste PBKDF2 selbst wrappen

**Rationale:** WebCrypto ist in allen Zielbrowsern verfügbar (Chrome 37+, Firefox 34+, Safari 7+). Kein WASM-Download, kein Build-Overhead, kein Supply-Chain-Risiko.

**Primitives:**
```
PBKDF2(SHA-256, 600 000 Iterationen, 32 Byte Output) → Key-Ableitung
AES-GCM 256 bit (random 12-Byte IV, prepended zum Ciphertext)  → Datenverschlüsselung
AES-KW 256 bit                                                   → DEK-Wrapping
```

---

### 3. Envelope Encryption (DEK pro Mitglied)

**Entscheidung:** Jedes Mitglied erhält einen zufälligen 256-bit Data Encryption Key (DEK). Der DEK wird separat für Vorstand (`DEK_enc_vorstand`) und optional für das Mitglied selbst (`DEK_enc_member`) gewrappt und in der DB gespeichert.

```
Schreiben:
  DEK  = crypto.getRandomValues(32 Byte)
  blob = AES-GCM(JSON{date_of_birth, street, zip, city, iban, account_holder}, DEK)
  DEK_enc_V = AES-KW(DEK, vorstand_key)
  DEK_enc_M = AES-KW(DEK, member_key)   // falls user_id vorhanden

Lesen (Vorstand):
  DEK = AES-KW-unwrap(DEK_enc_V, vorstand_key)
  payload = AES-GCM-decrypt(blob, DEK)

Lesen (Mitglied):
  DEK = AES-KW-unwrap(DEK_enc_M, member_key)
  payload = AES-GCM-decrypt(blob, DEK)
```

**Alternative:** Direkte Verschlüsselung mit vorstand_key (kein DEK) → kein Dual-Key-Zugriff möglich; Key-Rotation würde alle Daten re-encrypten statt nur DEKs.

**Rationale:** DEK-Schicht entkoppelt Key-Rotation vom Datenzugriff. Rotation = DEKs neu wrappen (kein Ciphertext-Neuencrypt nötig).

---

### 4. Geteilte Vorstand-Passphrase (nicht per-Person-Keypairs)

**Entscheidung:** Alle Vorstand-Mitglieder verwenden dieselbe Passphrase. Der abgeleitete `vorstand_key` ist für alle identisch.

**Alternative:** RSA/EC-Keypair pro Vorstand-Mitglied; DEK für jedes Mitglied N-fach gewrappt.

**Rationale:** Verein hat 2–3 Vorstand-Personen. Shared-Secret ist manageable. Per-Person-Keys würden N DEK-Wraps pro Mitglied erfordern, zusätzliche Key-Storage-Infrastruktur und einen komplexen Onboarding-Flow für neue Vorstand-Mitglieder. Der erhöhte Aufwand steht nicht im Verhältnis zum Threat-Modell.

**Rotation bei Vorstandswechsel:** Neues Passwort → Browser re-wrapped alle DEKs → neues Passwort out-of-band weitergegeben.

---

### 5. Vorstand-Salt ist per-Installation fix, Member-Salt ist per-Nutzer

**Entscheidung:**
- `vorstand_kdf_salt`: Ein zufälliger 32-Byte Salt, einmalig bei Setup generiert, in `clubs.vorstand_kdf_salt` gespeichert.
- `member_salt` in `member_sensitive`: individuell pro Mitglied generiert beim ersten Schreiben.

**Rationale:** Der Vorstand-Key ist global (alle teilen ihn); sein Salt muss nur gegen Rainbow-Tables schützen, nicht pro-Entry variieren. Member-Keys sind per-User; unterschiedliche Salts verhindern, dass gleiche Passwörter denselben Key ergeben.

---

### 6. sessionStorage für Key-Caching

**Entscheidung:** `vorstand_key` wird base64-kodiert in `sessionStorage['vk']` gecacht. Kein `localStorage`.

**Rationale:** `sessionStorage` wird beim Tab-Schließen geleert. `localStorage` würde den Key dauerhaft im Browser halten — schlechter Trade-off. Reines In-Memory-State (React) geht bei Navigation verloren.

Zusätzlich: 30-Minuten-Inaktivitäts-Timer löscht `sessionStorage['vk']` und zwingt zur erneuten Passphrase-Eingabe.

---

### 7. Datenlayout: Einzelner JSON-Blob pro Mitglied

**Entscheidung:** Alle sensiblen Felder werden als ein AES-GCM-Blob gespeichert (kein Feldweise-Encrypt).

```sql
CREATE TABLE member_sensitive (
  member_id        INTEGER PRIMARY KEY REFERENCES members(id) ON DELETE CASCADE,
  ciphertext       TEXT NOT NULL,   -- base64(iv || AES-GCM(payload))
  dek_enc_vorstand TEXT NOT NULL,   -- base64(AES-KW(DEK, vorstand_key))
  dek_enc_member   TEXT,            -- base64(AES-KW(DEK, member_key)), nullable
  member_salt      TEXT             -- base64, PBKDF2-salt für member_key
);

-- In clubs-Tabelle:
ALTER TABLE clubs ADD COLUMN vorstand_kdf_salt TEXT;
ALTER TABLE clubs ADD COLUMN vorstand_key_check TEXT; -- AES-GCM("ok", vorstand_key)
```

**Alternative:** Ein Column pro Feld (6 Ciphertext-Columns) → 6 DEK-Wraps, 6 IVs, komplexere Schema-Evolution.

**Rationale:** Felder werden immer zusammen gelesen/geschrieben. Ein Blob ist einfacher und erfordert nur einen DEK.

## Risks / Trade-offs

**[Vorstand vergisst Passphrase]** → alle sensiblen Daten aller Mitglieder unzugänglich.  
Mitigation: Zwei-Personen-Regel dokumentieren. Bei Anlegen der Passphrase explizite Warnung anzeigen. Keine serverseitige Recovery — das ist Absicht.

**[Passwort-Änderung eines Mitglieds ohne DEK-Re-Wrap]** → `DEK_enc_member` veraltet; Mitglied kann eigene Daten nicht mehr lesen.  
Mitigation: Change-Password-Flow (`PUT /api/auth/change-password`) schickt altes und neues Passwort; Browser re-derived beide Keys, re-wrapped DEK, postet neues `DEK_enc_member` mit.

**[XSS liest sessionStorage]** → vorstand_key exponiert.  
Mitigation: Content Security Policy (`script-src 'self'`), kein `eval`, alle Third-Party-Scripts vermeiden. Das Risiko ist real aber klein für eine interne Single-Origin-App.

**[Migrationsphase: Doppelte Datenhaltung]** → kurz existieren alte Klartext-Felder und neues Ciphertext parallel.  
Mitigation: Alte Felder werden erst gedroppt nachdem Vorstand alle Datensätze migriert hat (separater Migrations-Schritt als nächste DB-Migration nach Vollständigkeit).

**[WebCrypto-Fehler schwer zu debuggen]** → Crypto-Exceptions haben wenig Kontext.  
Mitigation: `lib/crypto.ts` wrrappt alle Calls mit sprechenden Error-Messages. Unit-Tests mit bekannten Testvektoren.

## Migration Plan

1. **DB-Migration deployen** (`migration_N_member_sensitive.up.sql`): Tabelle `member_sensitive` anlegen; `vorstand_kdf_salt` und `vorstand_key_check` zu `clubs` hinzufügen. Alte Felder in `members` bleiben vorerst nullable.

2. **Vorstand richtet Tresor ein** (`/admin/tresor-einrichten`): Passphrase eingeben → Salt + Key-Check-Value in DB schreiben → Vorstand-Key ab jetzt nutzbar.

3. **Einmalige Datenmigration** (`/admin/tresor-migration`): Vorstand lädt alle Mitgliedsdatensätze (Klartext aus alter Spalte), verschlüsselt im Browser, postet Ciphertext. Fortschrittsanzeige. Idempotent (bereits migrierte Datensätze werden übersprungen).

4. **Rollout abgeschlossen**: Nach erfolgreicher Migration werden die alten Klartext-Spalten in einer Folgemigration auf `NULL` gesetzt und schließlich gedroppt.

**Rollback:** Solange alte Spalten nicht gedroppt sind, kann der vorherige Stand durch Rücksetzen der Frontend-Version wiederhergestellt werden (alte Felder noch in DB).

## Open Questions

- Soll `vorstand_key_check` periodisch neu generiert werden (z.B. nach Rotation), oder reicht ein einmaliger Check-Value?
- Sollen Telefonnummern (`user_phones`) ebenfalls verschlüsselt werden? (Aktuell: Klartext in separater Tabelle, adressiert in dieser Change nicht.)
- Wie wird der Vorstand über den initialen Setup-Schritt informiert (E-Mail, UI-Banner)?

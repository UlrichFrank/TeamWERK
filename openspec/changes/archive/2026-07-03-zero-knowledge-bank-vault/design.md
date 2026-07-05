## Context

Ausgerollt (aus `encrypt-bank-sepa-at-rest`): Bank-/SEPA-PII liegt AES-256-GCM
verschlüsselt at-rest (`"v1:"`-Format), der **Server** hält `FIELD_ENCRYPTION_KEY`
und entschlüsselt zentral autorisiert über `policy.CanDecryptBankData`
(admin ∨ vorstand ∨ kassierer ∨ Eigentümer ∨ Elternteil). Serverseitige
Klartext-Konsumenten heute:

- `internal/beitragslauf` (`query.go` → `Decrypt` jeder Mitglieds-IBAN/Kontoinhaber;
  `handler.go` Validierung/Ausschluss; `xml.go` `pain.008`-Erzeugung) — Batch über
  **alle** aktiven Mitglieder.
- `internal/members/bank_crypto.go`, `drafts.go` (Einzel-Lesen/Schreiben, gegated).
- `internal/config/handler.go` (Vereins-SEPA-Stammdaten).
- `internal/upload/handler.go` (SEPA-Mandat-PDFs, `EncryptBytes`/`DecryptBytes`).

Constraints: einzelner IONOS VPS (1 GB RAM), Go 1.26, `modernc.org/sqlite`, React 18.
**Der Verein betreibt den VPS selbst.**

## Goals / Non-Goals

**Goals:**
- Bank-/SEPA-PII clientseitig verschlüsseln; Server speichert **nur** Ciphertext +
  gewrappte Schlüssel und besitzt **keinen** Entschlüsselungsschlüssel.
- Lese-Berechtigte: **Eigentümer** (verknüpftes Mitglied) ∨ **Vorstand** ∨ **Kassierer**
  über einen gemeinsamen Finance-Group-Key. **Niemand sonst — auch nicht `admin`,
  nicht der Server.**
- Backup-/Snapshot-Diebstahl, neugieriger Hoster und lesender Admin laufen ins Leere
  (nur unlesbare Blobs).

**Non-Goals:**
- Schutz gegen **aktiv bösartigen Server** (Trojaner-JS), Supply-Chain im
  Frontend-Build oder ein **kompromittiertes Endgerät / Phishing** des Kassierers
  (= „Path B", verworfen — siehe D6).
- Zero-Downtime-Migration (bewusst koordinierte Aktion).
- Änderung der fachlichen Beitragslauf-Logik (kein Pro-rata etc.) — nur Verlagerung
  des `pain.008`-Builders in den Client.

## Bedrohungsmodell (explizit)

| Szenario | Wahrscheinlichkeit (Verein) | Path A (dieses Modell) |
|---|---|---|
| Gestohlenes Backup / Disk-Snapshot | realistisch | ✅ nur Ciphertext |
| Neugieriger/kompellierter Hoster (at rest) | niedrig–mittel | ✅ nur Ciphertext |
| App-`admin` liest Bankdaten | mittel (Insider) | ✅ kein Schlüssel |
| App-RCE serviert Trojaner-JS, wartet auf Login | niedrig–mittel | ❌ at-rest sicher, Live-Tampering gewinnt |
| Gestohlene Deploy-Credentials → Trojaner-Bundle | niedrig | ❌ |
| Frontend-Supply-Chain (npm) | niedrig, steigend | ❌ |
| Kassierer-Endgerät / Phishing | **höchstes Realrisiko** | ❌ (Klartext landet ohnehin beim Bank-Upload) |

Kernaussage: Das **höchste Realrisiko** (Endgerät/Phishing) deckt **kein** Modell ab;
**Path A** schließt die gesamte passive/at-rest/Admin-Klasse, in der kleine Vereine
real verbrennen. Die Szenarien, in denen „Path B" (Native/Extension) zusätzlich
hülfe, sind allesamt niedrigwahrscheinlich **und** verlangen einen aktiven,
persistenten Angreifer, der auf einen Login wartet.

## Decisions

### D1 — Envelope-Encryption mit zwei Wrap-Pfaden (LOCKED)
Pro Mitglied: zufälliger `DEK_m`; `blob_m = AES-GCM(DEK_m, {iban, kontoinhaber, …})`.
`DEK_m` wird gewrappt an (a) `FinanceGroupPub` (immer) und (b) `OwnerPub` (nur wenn
das Mitglied einen Login hat). Der Server speichert `blob_m` + die Wraps; er kann
keinen davon öffnen.
**Warum:** Multi-Reader mit rotierenden Lesern (Vorstand/Kassierer) ist nur über
Key-Wrapping lösbar; der Group-Key vermeidet, bei jedem Boardwechsel jeden
Mitglieds-DEK neu wrappen zu müssen.

### D2 — Finance-Group-Key (LOCKED)
Ein Vereins-Schlüsselpaar `FinanceGroup{Pub,Priv}`. `FinanceGroupPriv` ist an jeden
**aktuellen** `vorstand`/`kassierer`-Pubkey gewrappt, zusätzlich an einen
**gedruckten Recovery-Code** (Vorstand-Safe). Mitglieds-DEKs werden an
`FinanceGroupPub` gewrappt.
**Warum:** „alle Vorstand+Kassierer lesen alles" ohne O(Mitglieder)-Re-Wrap pro
Rollenwechsel.

### D3 — Nutzer-Schlüsselpaare + Split-Key-Derivation (LOCKED)
Jeder Nutzer hat ein Schlüsselpaar; `UserPriv` wird mit
`KEK = Argon2id(pw, salt_enc)` gewrappt und als Ciphertext serverseitig abgelegt.
Authentifizierung nutzt eine **getrennte** Ableitung `Argon2id(pw, salt_auth)`, damit
der Server beim Login nie einen Wert erhält, mit dem sich `UserPriv` öffnen ließe
(Bitwarden-Prinzip).
**Warum:** Ohne Split sähe der Server beim Login das Passwort und könnte den
Wrapping-Key ableiten → Zero-Knowledge gebrochen.

### D4 — Kinder/Login-lose Mitglieder → nur Group-Wrap (LOCKED)
Mitglieder ohne eigenen Login erhalten **keinen** Owner-Wrap; ihr DEK wird nur an
`FinanceGroupPub` gewrappt. Wer die IBAN einträgt (Elternteil/Kassierer)
verschlüsselt im Browser an `FinanceGroupPub` und kann sie **nicht** zurücklesen.
**Folge:** Eltern verlieren das heutige Lese-Recht (`…∨ Elternteil` in
`CanDecryptBankData`). „Niemand sonst" wird strikt erfüllt.
**Offen (Tasks):** Korrektur-UX, wenn der Elternteil einen Tippfehler nicht
zurücklesen kann (erneut eingeben vs. Kassierer-assistiert).

### D5 — Fee-Run komplett clientseitig (LOCKED)
Der Server liefert für den Fee-Run nur `blob`+`group-wrap` aller aktiven Mitglieder
aus. Der Browser des Kassierers entwrappt mit `FinanceGroupPriv`, validiert IBANs
(Port `sepa/iban.go` → TS), wendet die Ausschlusslogik an, baut `pain.008` und die
Mandat-PDFs und lädt das Ergebnis herunter.
**Offen (Tasks):** Wandert das Klartext in das **append-only Saison-Protokoll**?
Falls es IBANs enthält, muss auch das Protokoll clientseitig erzeugt/gespeichert
werden; enthält es nur Namen/Beträge, bleibt es serverseitig.

### D6 — Path B (Server wirklich aus der Trust-Base) verworfen
Native-App oder Browser-Extension-aus-Store als Auslieferungskanal, den der Server
nicht still austauschen kann. **Verworfen:** unverhältnismäßiger Aufwand (Monate,
Store-Distribution, fremde UX für ehrenamtlichen Vorstand) für eine niedrig­
wahrscheinliche Angriffsklasse, die zudem das höchste Realrisiko (Endgerät) nicht
adressiert — und der Verein betreibt den VPS **selbst**, der Angreifer „bösartiger
Betreiber" ist also faktisch „bereits gerooteter eigener Host".
**Re-Evaluierung**, falls „der Hoster muss kryptografisch ausgeschlossen sein" zur
harten (z. B. vertraglichen) Anforderung wird.

### D7 — Migration über den Kassierer-Browser (LOCKED)
Da alles an `group_public_key` gewrappt wird und der Tresor-Inhaber die Passphrase (und
damit `GroupPriv`) hält, migriert dessen Browser den gesamten `v1:`-Bestand in einem Lauf:
über die noch vorhandene Server-Brücke (`FIELD_ENCRYPTION_KEY`) entschlüsseln → clientseitig
zu Envelope re-verschlüsseln (Wrap an `group_public_key`) → hochladen. Danach wird
`FIELD_ENCRYPTION_KEY` entfernt.

**D7.1 — Zwei-Deploy-Strategie für ein minimales Brücken-Fenster (LOCKED 2026-06-25).**
Das sicherheitskritische, irreversible Fenster ist die Zeit, in der der Server gleichzeitig
den Brücken-Schlüssel **und** einen `v1:`-Klartext über TLS ausliefernden Endpoint hält.
Statt es per zweitem Build+Deploy zu schließen (Fenster = ganzer Deploy-Zyklus *nach* der
Migration), wird die **nicht-destruktive Startup-Toleranz vorgezogen**: Branch A macht den
Server lauffähig **mit und ohne** Schlüssel. Der kritische Moment ist dann eine
sekundenschnelle, skriptbare Ops-Aktion (`make zk-finalize-remote`: Schlüssel aus `env`
entfernen + Restart) — **kein Code-Deploy**. Der eigentliche Code-/Spalten-Abbau (Branch B)
ist reine Hygiene und folgt zeitlich entkoppelt.

```
Branch A (feat/zk-migrate-bestand)  →  deploy  [reversibel; FIELD_ENCRYPTION_KEY bleibt]
   ├─ Startup tolerant ggü. fehlendem Key (vorgezogen aus dem alten 6.3)
   ├─ gegateter Brücken-Endpoint  /api/admin/migrate-legacy/{status,data,upload}
   └─ Frontend  /admin/migration  (entsperrter Tresor erforderlich)
        ↓ Tresor-Inhaber migriert im Browser (Minuten)
make zk-finalize-remote  →  status==complete? → Key aus env entfernen → restart  [kritisch, ~Sek.]
        ↓
Branch B (feat/zk-remove-bridge)  →  deploy  [Hygiene, jederzeit]
   ├─ internal/migration + Brücke (crypto.Decrypt/…) entfernen, InitFromEnv-Aufruf streichen
   └─ Migration 009: Legacy-Spalten droppen
```

**Idempotenz & Self-Disable:** `migrate-legacy/upload` schreibt pro Datensatz in **einer
Transaktion** den Envelope **und** nullt die Legacy-`v1:`-Spalte (Mandat: Datei auf
Client-Magic umschreiben). `migrate-legacy/data` liefert nur noch nicht-migrierte Datensätze;
`status.complete` wird true, sobald kein `v1:`-Bestand mehr existiert — der Endpoint
deaktiviert sich damit faktisch selbst, noch bevor Branch B ihn entfernt. Der `data`/`upload`-
Pfad ist zusätzlich an `crypto.HasKey()` gebunden (404 „nur wenn Bridge").

**Reihenfolge (gesamt):** (1) Tresor einrichten + alle Bank-Flows im Browser prüfen, (2)
Branch A deployen, (3) DB-Backup ziehen, (4) Browser-Migrationslauf, (5)
`make zk-finalize-remote`, (6) Branch B deployen.

## Offene Design-Fragen (für Spec-Phase)

- **Krypto-Primitive:** WebCrypto (ECDH P-256 universell; X25519 neuer) vs.
  `libsodium-wasm` (sealed boxes / `crypto_box`). Wrapping-Schema (ECDH+AES-KW vs.
  sealed box). Go-Seite braucht kaum noch Krypto (nur Blob-Store + Migrations-Brücke).
- **Schema:** Tabellen für `user_public_keys`, `member_bank_blobs` (blob + wraps),
  `finance_group_key` (Pub + Wraps pro Rolleninhaber + Recovery-Wrap),
  `recovery_blobs`. Verhältnis zu bestehenden Spalten `members.iban/account_holder`.
- **Auth-Flow:** konkrete Umstellung Login/Refresh auf Split-Derivation; Auswirkung
  auf bestehende Passwörter (Re-Hash beim nächsten Login?).
- **Rollen-Zeremonie:** UX für „neuer Kassierer bekommt Group-Wrap" (welcher
  bestehende Halter macht das, wann) und „entzogener Rolleninhaber" (Group-Key
  rotieren = alle DEKs neu wrappen, oder Restrisiko akzeptieren?).
- **Recovery:** Format/Stärke des gedruckten Group-Recovery-Codes; Eigentümer-
  Recovery (oder bewusst „bei Passwort-Reset eigene IBAN neu eingeben").
- **Drafts:** `member_change_drafts` mit `field_name='bankdaten'` müssen ebenfalls als
  Group-Blob laufen (heute server-entschlüsselt in `drafts.go`).
- **Verhältnis zu `member-encryption`/`vorstand-vault`-Drafts** (laut Memory
  Spec/Code-Drift auf einem `encryption`-Branch): prüfen, ob dort verwertbare Vorarbeit
  liegt, sonst als veraltet markieren.

## Wiederverwendung & Reframing aus `origin/encryption` (Fund 2026-06-24)

Der Branch `origin/encryption` (Tip 2026-05-24, nie nach `main` gemerged) enthält eine
**vollständige, lauffähige WebCrypto-Implementierung** desselben Vorhabens — breiter im
Feld-Scope (auch Adresse/Geburtsdatum), aber für Bank-/SEPA-PII direkt verwertbar:

- `web/src/lib/crypto.ts` — kompletter Envelope-Core: **PBKDF2(SHA-256, 600 000 Iter.)**
  → AES-KW-Wrapping-Key, **AES-GCM-256** (12-Byte-IV prepended) für Daten, `wrapKey`/
  `unwrapKey`, Key-Check-Verifikation, Salt-Generierung. Reine WebCrypto, **kein WASM/
  keine npm-Abhängigkeit**.
- `internal/db/migrations/011_member_sensitive.{up,down}.sql` — Schema-Vorlage:
  `member_sensitive(member_id PK, ciphertext, dek_enc_vorstand, dek_enc_member NULL,
  member_salt)` + `clubs.vorstand_kdf_salt`, `clubs.vorstand_key_check`.
- UI-Gerüst: `VaultContext` (sessionStorage `vk`, 30-min-Inaktivität, Key-Caching),
  `VaultGate`, `VaultPassphraseDialog`, `AdminTresorEinrichtenPage`,
  `AdminTresorVerwaltungPage`, `MemberSensitivTab`.
- Specs `vorstand-vault` (Passphrase-Entry, Session-Expiry, Setup-once, Rotation) und
  `member-encryption` (Requirement-Phrasing direkt übernehmbar).

**Wichtiges architektonisches Reframing:** Der Branch nutzt eine **geteilte
Vorstand-Passphrase** statt Pro-Person-Schlüsselpaaren — eine **separate** Passphrase
(nicht das Login-Passwort), die nie an den Server geht; Server speichert nur Salt +
Key-Check. `vorstand_key = PBKDF2(passphrase, salt)` wrappt jeden Mitglieds-DEK
(`dek_enc_vorstand`); der Eigentümer-Pfad (`dek_enc_member`) leitet aus dem
**Login-Passwort** ab. Das verändert zwei meiner ursprünglichen Annahmen:

- **D2/D3 → Alternative „geteilte Passphrase":** Für 2–3 Vorstands-/Kassierer-Personen
  ist die geteilte Passphrase deutlich einfacher, **bereits geschrieben** und
  proportional. Preis: Secret-Verteilung out-of-band, Rotation bei jedem Wechsel
  (re-wrap aller DEKs — Endpoint existiert), keine Pro-Person-Revocation. Pro-Person-
  Keypairs (mein ursprüngliches D2/D3) bieten Onboarding/Revoke ohne Secret-Kanal,
  kosten aber viel mehr (Keypairs, Group-Key, Login-Flow-Umbau) — nichts davon
  geschrieben. **Fork → offene Frage 1.**
- **Split-Key-Derivation (D3) ist für Path A unnötig:** Der Gruppenpfad nutzt ohnehin
  eine **separate**, nie übertragene Passphrase (Server sieht das Gruppen-Secret nie).
  Der Eigentümer-Pfad nutzt das Login-Passwort — der Server sieht es transient nur beim
  Login, was ausschließlich gegen einen **aktiv bösartigen Server** (Path B,
  ausgeschlossen) relevant wäre. Für Path A kann D3 entfallen.

**Mapping unserer Anforderungen auf das Branch-Modell:**
- „Kinder ohne Login → nur Group-Wrap" ✅ bereits abgebildet (`dek_enc_member` NULL).
- „admin ausgeschlossen" ✅ admin hat ohne Passphrase keinen Zugriff.
- **Anzupassen:** Scope auf Bank/SEPA verengen; `kassierer` neben `vorstand` als
  Group-Halter; **Vereins-SEPA-Stammdaten** (`clubs.*`) als Group-Blob ergänzen
  (Branch verschlüsselte sie nicht); **binäre Blobs** für SEPA-Mandat-PDFs (Lib kann
  heute nur Objekt/String); **`pain.008` clientseitig** (Branch hatte nur CSV-Export);
  `member_change_drafts(bankdaten)` als Group-Blob.

**Branch-Code ist veraltet** (Pre-SEPA-Beitragslauf, ~Mai 2026) — der **Krypto-Core
`lib/crypto.ts` und das Schema-/Spec-Muster sind wiederverwendbar**, das umgebende
Frontend/Backend muss an den heutigen `main`-Stand portiert werden.

## Gelockte Entscheidungen (Antworten 2026-06-24/25)

- **G1 — Asymmetrisches Finance-Gruppen-Keypair (Modell B, gewählt 2026-06-25).** Ein
  Vereins-RSA-OAEP-Keypair. Der **öffentliche** Schlüssel (`group_public_key`) ist nicht
  geheim, wird an jeden Client ausgeliefert und erlaubt **jedem** (Mitglied, Elternteil,
  öffentliches Beitritts-Formular) das Verschlüsseln eines DEK an die Gruppe
  (`dek_enc_vorstand = RSA-OAEP(DEK, group_public_key)`). Der **private** Schlüssel
  (`GroupPriv`, PKCS8) liegt serverseitig als `group_private_key_enc = AES-GCM(GroupPriv,
  PBKDF2(passphrase, salt))` — entschlüsselbar nur mit der **einen geteilten Passphrase**,
  die `vorstand`/`kassierer` kennen und die den Browser nie verlässt. **Genau ein
  Menschen-Secret** (die Passphrase), identisch zur symmetrischen Variante; das Keypair ist
  passphrase-geschützte Maschinerie.
  **Warum B statt symmetrisch (A):** B erhält die **Self-Service-Schreibpfade**
  (Beitrittsantrag mit IBAN, Profil-/Eltern-Eingabe, Bankdaten-Draft), weil Schreiben nur
  den öffentlichen Schlüssel braucht. Lesen bleibt streng passphrase-gated (nur
  Vorstand/Kassierer). Verworfen A: dort kann nur ein Passphrase-Inhaber schreiben →
  Bankdaten würden 100 % Backoffice, bestehende Flows entfielen.
- **G2 — Kein Eigentümer-Selbstlesen (bleibt).** Es **liest** ausschließlich die
  Finance-Gruppe. Mitglieder/Eltern **schreiben** (per öffentlichem Schlüssel), können aber
  **nicht zurücklesen** (kein `GroupPriv`). Kein `dek_enc_member`. Krypto vollständig vom
  Login/Passwort-Reset entkoppelt → **kein Auth-Umbau**.
- **G3 — Kein serverseitiges Recovery.** Zwei-Personen-Regel + Warnung bei Einrichtung;
  Passphrase-Verlust = `GroupPriv` unentschlüsselbar = Datenverlust (akzeptiert).
- **G4 — Krypto-Core portieren, UI neu.** WebCrypto-Basis (AES-GCM/PBKDF2/Salt/Key-Check)
  aus `origin/encryption` übernommen; AES-KW-Gruppen-Wrapping **ersetzt durch RSA-OAEP**
  (Keypair-Modell); Vault-UI frisch.
- **G5 — CSV-Bankimport clientseitig.** Der Browser parst die CSV, validiert + verschlüsselt
  IBANs lokal (per öffentlichem Schlüssel), lädt Envelopes hoch. Serverseitiger CSV-
  Bankimport entfällt.

Damit entfallen: D3 (Split-Key-Derivation), Pro-Person-Keypairs, Recovery-Blobs,
`dek_enc_member`/`member_salt`. **Schema:** `member_sensitive(member_id, ciphertext,
dek_enc_vorstand)` + `clubs.group_public_key`, `clubs.group_private_key_enc`,
`clubs.vorstand_kdf_salt`, `clubs.vorstand_key_check`.

**Krypto-Primitive (Modell B):**
- Keypair: RSA-OAEP (SHA-256), 2048 bit. `wrapKey('raw', DEK, GroupPub, 'RSA-OAEP')` /
  `unwrapKey(..., GroupPriv, 'RSA-OAEP', 'AES-GCM')`.
- DEK: AES-GCM-256, Daten = AES-GCM(payload, DEK), IV prepended (unverändert).
- `GroupPriv`-Schutz: `KEK = PBKDF2(passphrase, salt, 600k, SHA-256)`; `group_private_key_enc
  = AES-GCM(PKCS8(GroupPriv), KEK)`. `vorstand_key_check = AES-GCM("ok", KEK)`.

## Ablauf Key-Wechsel (Modell B — zwei Stufen)

Modell B trennt zwei Fälle, die im symmetrischen Modell verschmolzen wären:

**(a) Passphrase-Rotation — Normalfall (Ausscheiden / Leak-Verdacht der Passphrase) — O(1):**
```
GroupPriv mit ALTER Passphrase entschlüsseln (KEK_ALT = PBKDF2(pw_ALT, salt_ALT))
   → mit KEK_NEU = PBKDF2(pw_NEU, salt_NEU) neu verschlüsseln
   → group_private_key_enc, vorstand_kdf_salt, vorstand_key_check ersetzen
Die Mitglieds-DEKs/Blobs UND group_public_key bleiben UNANGETASTET.
```
Ablauf (Browser eines aktuellen Halters): Tresor mit alter Passphrase entsperren →
`GroupPriv` im RAM → neue Passphrase wählen → `GroupPriv` unter `KEK_NEU` neu verschlüsseln
→ `PUT /api/admin/rotate-encryption {vorstand_kdf_salt, vorstand_key_check,
group_private_key_enc}`. **Kein** DEK-Batch nötig — daher O(1), unabhängig von der
Mitgliederzahl (günstiger als das symmetrische Modell, wo jeder DEK neu gewrappt werden
müsste).

**(b) Keypair-Rotation — Ausnahmefall (`GroupPriv` selbst kompromittiert) — O(n):**
```
neues Keypair erzeugen → alle DEKs mit altem GroupPriv entschlüsseln
   → mit neuem GroupPub neu wrappen (dek_enc_vorstand) → Batch + neuer pub/priv schreiben
```
Nur nötig, wenn der private Schlüssel selbst geleakt ist (er liegt nur nach Entsperren im
Browser-RAM). `rotate-encryption` akzeptiert dafür optional einen `wraps`-Batch +
`group_public_key`.

Der Server sieht in **keinem** Fall die Passphrase oder `GroupPriv` im Klartext.

Eigenschaften & Fallstricke:
- **Atomar:** Server schreibt clubs-Felder (+ ggf. DEK-Batch) in **einer** Transaktion.
- **Forward-Secrecy-Grenze (ehrlich):** Passphrase-Rotation (a) sperrt eine Person aus, die
  **nur** die Passphrase kannte. Hat sie `GroupPriv` aktiv aus dem Browser-RAM kopiert, hilft
  nur Keypair-Rotation (b). Für den Zielangreifer (passiv: Backup/Hoster/Admin) ist (a)
  ausreichend.
- **Abgrenzung:** Dies ist **nicht** das serverseitige `rotate-key`/`"v2:"` aus dem
  verworfenen `harden-field-encryption-key` (server-seitig). Hier rein clientseitig.

## Risks / Trade-offs

- **Recovery-Verlust = Datenverlust:** Geht `FinanceGroupPriv` (alle Halter + Code)
  verloren, sind **alle** Bank-/SEPA-Felder unwiederbringlich. Mitigation: Wrap an
  mehrere Rolleninhaber + physischer Recovery-Code; DB-Backup vor Migration.
- **Großer Frontend-Umbau:** `pain.008`/IBAN/Mandat-PDF in TS neu — fachliche
  Parität mit `internal/beitragslauf` muss getestet werden (gleiche XML-Ausgabe).
- **Migration ist koordiniert, kein Zero-Downtime:** Fenster nötig; Reihenfolge
  strikt (erst Pubkeys, dann Migration, dann Server-Schlüssel löschen).
- **Modellgrenze bleibt:** aktiv bösartiger Server / Endgerät nicht abgedeckt —
  ehrlich dokumentieren, nicht überversprechen.
- **`admin` verliert Bank-Lesen:** beabsichtigt; ggf. Break-Glass-Überlegung
  (bewusst keine — würde Zero-Knowledge aufweichen).

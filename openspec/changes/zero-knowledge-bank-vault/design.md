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

### D7 — Migration über den Kassierer-Browser (LOCKED, Detail offen)
Da alles mindestens an `FinanceGroupPub` gewrappt wird und der Kassierer
`FinanceGroupPriv` hält, migriert dessen Browser den gesamten `v1:`-Bestand in einem
Lauf: über die noch vorhandene Server-Brücke (`FIELD_ENCRYPTION_KEY`) entschlüsseln
→ an Group (und vorhandene Owner-Pubkeys) re-wrappen → hochladen. Danach wird
`FIELD_ENCRYPTION_KEY` entfernt; Owner-Wraps für Mitglieder ohne Pubkey werden
lazily ergänzt, sobald sie sich anmelden.
**Reihenfolge:** (1) Schlüsselpaare/Group-Key ausrollen, (2) Mitglieder erzeugen
Pubkeys (oder bleiben group-only), (3) Kassierer-Migrationslauf, (4) Server-Schlüssel
löschen + serverseitige Decrypt-Pfade entfernen.

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

## Gelockte Entscheidungen (Antworten 2026-06-24)

- **G1 — Geteilte Finance-Gruppen-Passphrase** (statt Pro-Person-Keypairs). Eine
  gemeinsame Tresor-Passphrase für `vorstand`/`kassierer`, separat vom Login, nie an den
  Server; `vorstand_key = PBKDF2(passphrase, salt)` als AES-KW-Wrapping-Key. Branch-Modell,
  proportional für 2–3 Personen.
- **G2 — Kein Eigentümer-Selbstlesen.** Es liest ausschließlich die Finance-Gruppe; kein
  `dek_enc_member`, kein `member_salt`. Krypto vollständig vom Login/Passwort-Reset
  entkoppelt → **kein Auth-Umbau**, kein Change-Password-Re-Wrap. (Bewusste Aufweichung
  des ursprünglichen „Eigentümer liest eigenes".)
- **G3 — Kein serverseitiges Recovery.** Zwei-Personen-Regel + Warnung bei Einrichtung;
  Passphrase-Verlust = Datenverlust (akzeptiert).
- **G4 — Krypto-Core portieren, UI neu.** `web/src/lib/crypto.ts` aus `origin/encryption`
  übernehmen (um binäre Blobs für PDFs erweitern); Vault-UI/Flows frisch gegen heutige
  Komponenten-Standards bauen.

Damit entfallen aus dem ursprünglichen Entwurf: D3 (Split-Key-Derivation), Pro-Person-
Keypairs, Group-Keypair-Re-Wrap-Zeremonie, Recovery-Blobs, `dek_enc_member`/`member_salt`.
Schema reduziert sich auf `member_sensitive(member_id, ciphertext, dek_enc_vorstand)` +
`clubs.vorstand_kdf_salt`, `clubs.vorstand_key_check`.

## Ablauf Passphrase-Rotation (Detail)

Rotation erfolgt beim **Ausscheiden** eines `vorstand`/`kassierer` (oder bei Leak-Verdacht)
— **nicht** bei Aufnahme (dort wird die bestehende Passphrase nur out-of-band
weitergegeben). Der Hebel ist die **DEK-Schicht**: nur die kleinen Data-Key-Wraps werden
neu gewickelt, die Bank-Blobs (`AES-GCM(bankdaten, DEK_m)`) bleiben unangetastet.

Ausgangslage pro Mitglied:
```
ciphertext       = AES-GCM(bankdaten, DEK_m)        ← bleibt unverändert
dek_enc_vorstand = AES-KW(DEK_m, vorstand_key_ALT)  ← wird ausgetauscht
vorstand_key_ALT = PBKDF2(passphrase_ALT, salt_ALT)
```

Ablauf (vollständig im Browser eines aktuellen Halters):
1. Tresor mit **alter** Passphrase entsperren → `vorstand_key_ALT` im RAM.
2. Neue Passphrase wählen → neuer Zufalls-Salt → `vorstand_key_NEU = PBKDF2(pw_NEU, salt_NEU)`.
3. Alle `{member_id, dek_enc_vorstand}` laden.
4. Pro Mitglied (reine AES-KW-Ops): `DEK_m = unwrap(dek_enc_vorstand, key_ALT)` →
   `dek_enc_vorstand_NEU = wrap(DEK_m, key_NEU)`.
5. `key_check_NEU = AES-GCM("ok", key_NEU)`.
6. Batch **atomar** an `PUT /api/admin/rotate-encryption`: `{salt_NEU, key_check_NEU,
   [{member_id, dek_enc_vorstand_NEU}, …]}`.
7. `sessionStorage['vk']` auf `key_NEU` aktualisieren (Session bleibt aktiv).
8. Neue Passphrase out-of-band an verbleibende Halter.

Der Server sieht **weder** alte **noch** neue Passphrase — nur Salt, `"ok"`-Prüfwert und
neu gewickelte DEKs.

Eigenschaften & Fallstricke:
- **Atomar (alles-oder-nichts):** Server schreibt die Batch in **einer** Transaktion; sonst
  entsteht ein Mischbestand aus ALT-/NEU-Wraps → unlesbar mit beiden Passphrasen.
- **Beide Schlüssel nur gleichzeitig im Browser:** `key_ALT` bleibt im RAM, bis die NEU-Batch
  bestätigt ist; bricht der Vorgang ab, wurde nichts geschrieben → wiederholbar.
- **Abbruch statt Teilschreiben:** Lässt sich ein einzelner DEK nicht entwrappen
  (Korruption/falsche Passphrase), bricht die Rotation ab, ohne zu schreiben.
- **Forward-Secrecy-Grenze (ehrlich):** Rotation sperrt die ausgeschiedene Person aus
  *künftigem* Zugriff aus; sie macht **nicht** ungeschehen, was diese Person während ihrer
  aktiven Zeit bereits gesehen/kopiert hat. Inhärent, kein Krypto-Schema löst das.
- **Abgrenzung:** Dies ist **nicht** das serverseitige `rotate-key`/`"v2:"` aus dem
  verworfenen `harden-field-encryption-key` (das server-seitig entschlüsselt). Hier rein
  clientseitig, Server hält keinen Schlüssel.

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

## Why

Das Vereins-Gruppen-Keypair, das alle Bank-/SEPA-DEKs wrappt, ist RSA-OAEP-SHA-256 mit **2048 Bit** (`web/src/lib/crypto.ts`, `RSA_MODULUS=2048`). 2048 Bit ist aktuell akzeptabel (~112 Bit Sicherheit, NIST-konform), aber dieser **eine** Schlüssel schützt den gesamten, auf Jahre angelegten Datenbestand ohne automatische Rotation und ohne Recovery. NIST SP 800-57 empfiehlt ≥ 3072 Bit für langfristig zu schützende Daten (Sicherheitsaudit 2026-06-26, **B-8 Info**). **Kein konkreter Exploit** — präventive Härtung. Die übrigen Primitive (AES-GCM-256, PBKDF2-SHA256 600k, frische IVs, per-Record-DEK, `crypto.getRandomValues`) sind einwandfrei.

## What Changes

- **Mindest-Schlüsselstärke** für das bei der Tresor-Einrichtung erzeugte Gruppen-Keypair: RSA-3072 (≈128-Bit) **oder** Migration auf X25519/ECDH-ES + HKDF zum DEK-Wrapping (schneller, kleinere Wraps) — Verfahren im Design/Apply zu wählen.
- **Bestand:** über den bereits spezifizierten Keypair-Rotations-Pfad (neues Keypair, alle DEKs neu wrappen) können RSA-2048-Installationen auf die Mindeststärke angehoben werden; kein Datenverlust, Bestand bleibt lesbar.
- Betrifft nur **neue** Setups bzw. eine bewusste Rotation; entschlüsselbar bleibt alles über die Tresor-Passphrase.

**Diese Proposal wird vorerst NICHT umgesetzt** (nur angelegt). Nebenbefund aus dem Audit (konstant-zeitiger Vergleich des `METRICS_TOKEN` in `internal/health/health.go`) ist NICHT Teil dieser Proposal — separater Einzeiler.

## Capabilities

### New Capabilities
<!-- keine -->

### Modified Capabilities
- `client-side-bank-encryption`: Neue Anforderung an die Mindest-Schlüsselstärke des Gruppen-Keypairs (RSA-3072 oder X25519-äquivalent); Anhebung des Bestands über den vorhandenen Rotations-Pfad.

## Impact

- **Code:** `web/src/lib/crypto.ts` (`RSA_MODULUS` 2048→3072 bzw. Umstellung des Wrapping-Verfahrens), ggf. `bankCrypto.ts`/`VaultContext`.
- **Kompatibilität:** Bestehende 2048-Envelopes bleiben entschlüsselbar; erst eine Rotation hebt die Stärke. Frontend muss beim Lesen beide Stärken/Verfahren tolerieren, falls schrittweise migriert wird.
- **Performance:** RSA-3072-Keygen/Wrap ist langsamer als 2048 (einmalig pro Setup/Rotation, unkritisch); X25519 wäre schneller.
- **Daten/Migration:** keine DB-Schemaänderung; reine clientseitige Re-Wrapping-Operation bei Rotation.

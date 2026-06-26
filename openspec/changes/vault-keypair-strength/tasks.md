## 1. Verfahren wählen

- [ ] 1.1 Entscheidung RSA-3072 vs. X25519/ECDH-ES + HKDF (Design); Abwägung Performance/Wrap-Größe/Browser-Support dokumentieren

## 2. Schlüsselerzeugung

- [ ] 2.1 `web/src/lib/crypto.ts`: `RSA_MODULUS` 2048→3072 bzw. Umstellung des Keygen-/Wrapping-Verfahrens für NEUE Setups
- [ ] 2.2 `bankCrypto.ts`/`VaultContext` an das ggf. neue Wrapping-Verfahren anpassen

## 3. Abwärtskompatibilität & Rotation

- [ ] 3.1 Lesen: Client toleriert sowohl alte (2048) als auch neue (3072/X25519) Envelopes
- [ ] 3.2 Keypair-Rotation hebt 2048 auf die Mindeststärke (alle DEKs neu wrappen), Bestand bleibt lesbar

## 4. Tests & Verifikation

- [ ] 4.1 Crypto-Unit-Tests (`web/src/lib/crypto.test.ts`): Roundtrip mit neuer Stärke; Lesen eines 2048-Envelopes bleibt möglich
- [ ] 4.2 `pnpm -C web test/build` + `openspec validate vault-keypair-strength --strict`

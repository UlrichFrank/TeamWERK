// Clientseitige Envelope-Verschlüsselung der Bank-/SEPA-PII (Zero-Knowledge "at rest",
// Modell B — asymmetrisches Gruppen-Keypair). Alle Schlüssel werden im Browser
// abgeleitet/verwendet; der Server sieht nur Ciphertext.
//
// Modell:
//   - Gruppen-Keypair (RSA-OAEP, SHA-256, 2048 bit): der ÖFFENTLICHE Schlüssel ist nicht
//     geheim und erlaubt JEDEM das Verschlüsseln (Schreiben); der PRIVATE Schlüssel liegt
//     mit der geteilten Tresor-Passphrase verschlüsselt vor und erlaubt nur Vorstand/
//     Kassierer das Lesen.
//   - Pro Mitglied ein zufälliger DEK (AES-GCM-256); Daten = AES-GCM(payload, DEK), IV
//     prepended. Der DEK wird per RSA-OAEP an den öffentlichen Gruppen-Schlüssel gewrappt.
//   - Schutz des privaten Schlüssels: KEK = PBKDF2(passphrase, salt, 600k, SHA-256);
//     group_private_key_enc = AES-GCM(PKCS8(GroupPriv), KEK); Key-Check = AES-GCM("ok", KEK).

const PBKDF2_ITERATIONS = 600_000
const SALT_BYTES = 32
const IV_BYTES = 12
const RSA_MODULUS = 2048

// --- Base64-Helfer ---

export function bufToB64(buf: ArrayBuffer): string {
  // In Blöcken kodieren: String.fromCharCode(...) mit einem großen Spread (z. B. ein
  // mehrere MB großes SEPA-Mandat-PDF) sprengt sonst den Call-Stack.
  const bytes = new Uint8Array(buf)
  const chunk = 0x8000 // 32 KiB
  let binary = ''
  for (let i = 0; i < bytes.length; i += chunk) {
    binary += String.fromCharCode(...bytes.subarray(i, i + chunk))
  }
  return btoa(binary)
}

export function b64ToBuf(b64: string): ArrayBuffer {
  const binary = atob(b64)
  const buf = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i++) buf[i] = binary.charCodeAt(i)
  return buf.buffer
}

// --- KEK-Ableitung aus der Tresor-Passphrase (AES-GCM, für Privatschlüssel + Key-Check) ---

export async function deriveKEK(passphrase: string, saltB64: string): Promise<CryptoKey> {
  const salt = b64ToBuf(saltB64)
  const raw = new TextEncoder().encode(passphrase)
  const base = await crypto.subtle.importKey('raw', raw, 'PBKDF2', false, ['deriveKey'])
  return crypto.subtle.deriveKey(
    { name: 'PBKDF2', hash: 'SHA-256', salt, iterations: PBKDF2_ITERATIONS },
    base,
    { name: 'AES-GCM', length: 256 },
    true,
    ['encrypt', 'decrypt'],
  )
}

// --- Gruppen-Keypair (RSA-OAEP) ---

export async function generateGroupKeypair(): Promise<CryptoKeyPair> {
  return crypto.subtle.generateKey(
    {
      name: 'RSA-OAEP',
      modulusLength: RSA_MODULUS,
      publicExponent: new Uint8Array([1, 0, 1]),
      hash: 'SHA-256',
    },
    true,
    ['wrapKey', 'unwrapKey'],
  )
}

export async function exportPublicKey(pub: CryptoKey): Promise<string> {
  return bufToB64(await crypto.subtle.exportKey('spki', pub))
}

export async function importPublicKey(spkiB64: string): Promise<CryptoKey> {
  return crypto.subtle.importKey(
    'spki',
    b64ToBuf(spkiB64),
    { name: 'RSA-OAEP', hash: 'SHA-256' },
    true,
    ['wrapKey'],
  )
}

// Privatschlüssel mit der KEK verschlüsseln (PKCS8 ‖ IV prepended) bzw. wieder importieren.
export async function encryptPrivateKey(priv: CryptoKey, kek: CryptoKey): Promise<string> {
  const pkcs8 = await crypto.subtle.exportKey('pkcs8', priv)
  const iv = crypto.getRandomValues(new Uint8Array(IV_BYTES))
  const ct = await crypto.subtle.encrypt({ name: 'AES-GCM', iv }, kek, pkcs8)
  const out = new Uint8Array(IV_BYTES + ct.byteLength)
  out.set(iv, 0)
  out.set(new Uint8Array(ct), IV_BYTES)
  return bufToB64(out.buffer)
}

export async function decryptPrivateKey(encB64: string, kek: CryptoKey): Promise<CryptoKey> {
  const buf = new Uint8Array(b64ToBuf(encB64))
  const iv = buf.slice(0, IV_BYTES) as BufferSource
  const data = buf.slice(IV_BYTES) as BufferSource
  const pkcs8 = await crypto.subtle.decrypt({ name: 'AES-GCM', iv }, kek, data)
  return crypto.subtle.importKey('pkcs8', pkcs8, { name: 'RSA-OAEP', hash: 'SHA-256' }, true, [
    'unwrapKey',
  ])
}

// --- DEK-Erzeugung + Wrapping an das Gruppen-Keypair ---

export async function generateDEK(): Promise<CryptoKey> {
  return crypto.subtle.generateKey({ name: 'AES-GCM', length: 256 }, true, ['encrypt', 'decrypt'])
}

// Wrappt einen DEK an den öffentlichen Gruppen-Schlüssel (Schreiben — kein Secret nötig).
export async function wrapDEK(dek: CryptoKey, groupPub: CryptoKey): Promise<string> {
  return bufToB64(await crypto.subtle.wrapKey('raw', dek, groupPub, { name: 'RSA-OAEP' }))
}

// Entwrappt einen DEK mit dem privaten Gruppen-Schlüssel (Lesen — nur Passphrase-Inhaber).
export async function unwrapDEK(wrappedB64: string, groupPriv: CryptoKey): Promise<CryptoKey> {
  return crypto.subtle.unwrapKey(
    'raw',
    b64ToBuf(wrappedB64),
    groupPriv,
    { name: 'RSA-OAEP' },
    { name: 'AES-GCM', length: 256 },
    true,
    ['encrypt', 'decrypt'],
  )
}

// --- Daten verschlüsseln/entschlüsseln (AES-GCM, IV prepended) ---

export async function encrypt(payload: object, dek: CryptoKey): Promise<string> {
  const iv = crypto.getRandomValues(new Uint8Array(IV_BYTES))
  const data = new TextEncoder().encode(JSON.stringify(payload))
  const ciphertext = await crypto.subtle.encrypt({ name: 'AES-GCM', iv }, dek, data)
  const out = new Uint8Array(IV_BYTES + ciphertext.byteLength)
  out.set(iv, 0)
  out.set(new Uint8Array(ciphertext), IV_BYTES)
  return bufToB64(out.buffer)
}

export async function decrypt(ciphertextB64: string, dek: CryptoKey): Promise<object> {
  const buf = new Uint8Array(b64ToBuf(ciphertextB64))
  const iv = buf.slice(0, IV_BYTES) as BufferSource
  const data = buf.slice(IV_BYTES) as BufferSource
  const plain = await crypto.subtle.decrypt({ name: 'AES-GCM', iv }, dek, data)
  return JSON.parse(new TextDecoder().decode(plain))
}

// --- String verschlüsseln/entschlüsseln (für den Key-Check-Wert) ---

async function encryptString(plain: string, key: CryptoKey): Promise<string> {
  const iv = crypto.getRandomValues(new Uint8Array(IV_BYTES))
  const data = new TextEncoder().encode(plain)
  const ciphertext = await crypto.subtle.encrypt({ name: 'AES-GCM', iv }, key, data)
  const out = new Uint8Array(IV_BYTES + ciphertext.byteLength)
  out.set(iv, 0)
  out.set(new Uint8Array(ciphertext), IV_BYTES)
  return bufToB64(out.buffer)
}

async function decryptString(ciphertextB64: string, key: CryptoKey): Promise<string> {
  const buf = new Uint8Array(b64ToBuf(ciphertextB64))
  const iv = buf.slice(0, IV_BYTES) as BufferSource
  const data = buf.slice(IV_BYTES) as BufferSource
  const plain = await crypto.subtle.decrypt({ name: 'AES-GCM', iv }, key, data)
  return new TextDecoder().decode(plain)
}

// --- Tresor-Passphrase verifizieren (Key-Check) ---

export async function verifyVaultPassphrase(
  passphrase: string,
  saltB64: string,
  keyCheckB64: string,
): Promise<boolean> {
  try {
    const kek = await deriveKEK(passphrase, saltB64)
    return (await decryptString(keyCheckB64, kek)) === 'ok'
  } catch {
    return false
  }
}

// --- Einrichtung: Keypair erzeugen, privaten Schlüssel + Key-Check unter der Passphrase ---

export interface VaultSetup {
  groupPublicKey: string // SPKI base64 — nicht geheim
  groupPrivateKeyEnc: string // AES-GCM(PKCS8, KEK) base64
  vorstandKdfSalt: string
  vorstandKeyCheck: string
}

export async function generateVaultSetup(passphrase: string): Promise<VaultSetup> {
  const saltB64 = generateSalt()
  const kek = await deriveKEK(passphrase, saltB64)
  const keypair = await generateGroupKeypair()
  return {
    groupPublicKey: await exportPublicKey(keypair.publicKey),
    groupPrivateKeyEnc: await encryptPrivateKey(keypair.privateKey, kek),
    vorstandKdfSalt: saltB64,
    vorstandKeyCheck: await encryptString('ok', kek),
  }
}

// Passphrase-Rotation (O(1)): denselben privaten Schlüssel unter einer neuen Passphrase
// neu verschlüsseln. Keypair (und damit alle DEKs) bleiben unverändert.
export async function rewrapPrivateKeyForRotation(
  priv: CryptoKey,
  newPassphrase: string,
): Promise<{ groupPrivateKeyEnc: string; vorstandKdfSalt: string; vorstandKeyCheck: string }> {
  const saltB64 = generateSalt()
  const kek = await deriveKEK(newPassphrase, saltB64)
  return {
    groupPrivateKeyEnc: await encryptPrivateKey(priv, kek),
    vorstandKdfSalt: saltB64,
    vorstandKeyCheck: await encryptString('ok', kek),
  }
}

// --- Salt-Erzeugung ---

export function generateSalt(): string {
  const salt = crypto.getRandomValues(new Uint8Array(SALT_BYTES))
  return bufToB64(salt.buffer)
}

// --- Binäre Blobs (SEPA-Mandat-PDFs) ---
//
// Format: MAGIC ‖ IV(12) ‖ AES-GCM(content, DEK). Der Magic-Header erlaubt es Lese-/
// Migrationspfaden, einen bereits verschlüsselten Blob von Klartext (z. B. "%PDF") zu
// unterscheiden — analog zum Datei-Header der serverseitigen Go-Implementierung.

const BLOB_MAGIC = new TextEncoder().encode('TWENC1\n') // 7 Byte

function startsWithMagic(blob: Uint8Array): boolean {
  if (blob.length < BLOB_MAGIC.length) return false
  for (let i = 0; i < BLOB_MAGIC.length; i++) if (blob[i] !== BLOB_MAGIC[i]) return false
  return true
}

export function isEncryptedBytes(blob: Uint8Array): boolean {
  return startsWithMagic(blob)
}

export async function encryptBytes(content: Uint8Array, dek: CryptoKey): Promise<Uint8Array> {
  const iv = crypto.getRandomValues(new Uint8Array(IV_BYTES))
  const ciphertext = await crypto.subtle.encrypt({ name: 'AES-GCM', iv }, dek, content as BufferSource)
  const out = new Uint8Array(BLOB_MAGIC.length + IV_BYTES + ciphertext.byteLength)
  out.set(BLOB_MAGIC, 0)
  out.set(iv, BLOB_MAGIC.length)
  out.set(new Uint8Array(ciphertext), BLOB_MAGIC.length + IV_BYTES)
  return out
}

export async function decryptBytes(blob: Uint8Array, dek: CryptoKey): Promise<Uint8Array> {
  if (!startsWithMagic(blob)) {
    throw new Error('decryptBytes: kein verschlüsselter Blob (Magic-Header fehlt)')
  }
  const iv = blob.slice(BLOB_MAGIC.length, BLOB_MAGIC.length + IV_BYTES) as BufferSource
  const data = blob.slice(BLOB_MAGIC.length + IV_BYTES) as BufferSource
  const plain = await crypto.subtle.decrypt({ name: 'AES-GCM', iv }, dek, data)
  return new Uint8Array(plain)
}

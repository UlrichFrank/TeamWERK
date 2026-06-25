// Clientseitige Envelope-Verschlüsselung der Bank-/SEPA-PII (Zero-Knowledge "at rest").
// Alle Schlüssel werden im Browser abgeleitet/verwendet; der Server sieht nur Ciphertext.
//
// Primitive (reine WebCrypto, kein WASM / keine npm-Abhängigkeit):
//   PBKDF2(SHA-256, 600 000 Iter.)  → Ableitung des Gruppen-Schlüssels aus der Passphrase
//   AES-KW 256                       → Wrapping der Data-Keys (DEK)
//   AES-GCM 256 (12-Byte-IV)         → Verschlüsselung der Daten (IV prepended)
//
// Modell: pro Mitglied ein zufälliger DEK; Daten = AES-GCM(payload, DEK); der DEK wird mit
// dem Gruppen-Schlüssel gewrappt (dek_enc_vorstand). Rotation = DEKs neu wrappen, ohne die
// Daten-Blobs anzufassen (siehe design.md). Es gibt bewusst keinen Eigentümer-Wrap.

const PBKDF2_ITERATIONS = 600_000
const SALT_BYTES = 32
const IV_BYTES = 12

// --- Base64-Helfer ---

export function bufToB64(buf: ArrayBuffer): string {
  return btoa(String.fromCharCode(...new Uint8Array(buf)))
}

export function b64ToBuf(b64: string): ArrayBuffer {
  const binary = atob(b64)
  const buf = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i++) buf[i] = binary.charCodeAt(i)
  return buf.buffer
}

// --- Schlüsselableitung aus der Tresor-Passphrase ---

// Leitet den Gruppen-Schlüssel als AES-KW-Wrapping-Key ab (zum Wrappen/Unwrappen der DEKs).
export async function deriveKey(passphrase: string, saltB64: string): Promise<CryptoKey> {
  const salt = b64ToBuf(saltB64)
  const raw = new TextEncoder().encode(passphrase)
  const base = await crypto.subtle.importKey('raw', raw, 'PBKDF2', false, ['deriveKey'])
  return crypto.subtle.deriveKey(
    { name: 'PBKDF2', hash: 'SHA-256', salt, iterations: PBKDF2_ITERATIONS },
    base,
    { name: 'AES-KW', length: 256 },
    true,
    ['wrapKey', 'unwrapKey'],
  )
}

// Leitet aus derselben Passphrase einen AES-GCM-Schlüssel ab — nur für den Key-Check-Wert.
export async function deriveKeyAsGCM(passphrase: string, saltB64: string): Promise<CryptoKey> {
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

// --- DEK-Erzeugung ---

export async function generateDEK(): Promise<CryptoKey> {
  return crypto.subtle.generateKey({ name: 'AES-GCM', length: 256 }, true, ['encrypt', 'decrypt'])
}

// --- DEK wrappen/entwrappen (AES-KW) ---

export async function wrapKey(dek: CryptoKey, wrappingKey: CryptoKey): Promise<string> {
  const wrapped = await crypto.subtle.wrapKey('raw', dek, wrappingKey, 'AES-KW')
  return bufToB64(wrapped)
}

export async function unwrapKey(wrappedB64: string, wrappingKey: CryptoKey): Promise<CryptoKey> {
  const wrapped = b64ToBuf(wrappedB64)
  return crypto.subtle.unwrapKey(
    'raw',
    wrapped,
    wrappingKey,
    'AES-KW',
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
  const iv = buf.slice(0, IV_BYTES)
  const data = buf.slice(IV_BYTES)
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
  const iv = buf.slice(0, IV_BYTES)
  const data = buf.slice(IV_BYTES)
  const plain = await crypto.subtle.decrypt({ name: 'AES-GCM', iv }, key, data)
  return new TextDecoder().decode(plain)
}

// --- Tresor-Passphrase verifizieren (Key-Check) ---

// Prüft clientseitig, ob die eingegebene Passphrase korrekt ist, ohne sie zu speichern oder
// einen Server-Request auszulösen: der abgeleitete Schlüssel muss den Key-Check zu "ok"
// entschlüsseln.
export async function verifyVaultPassphrase(
  passphrase: string,
  saltB64: string,
  keyCheckB64: string,
): Promise<boolean> {
  try {
    const gcmKey = await deriveKeyAsGCM(passphrase, saltB64)
    return (await decryptString(keyCheckB64, gcmKey)) === 'ok'
  } catch {
    return false
  }
}

// --- Einrichtung: Salt + Key-Check erzeugen ---

// Erzeugt die serverseitig zu speichernden, nicht-zurückrechenbaren Hilfswerte. Die
// Passphrase selbst verlässt den Browser nie.
export async function generateVaultSetup(
  passphrase: string,
): Promise<{ saltB64: string; keyCheckB64: string }> {
  const salt = crypto.getRandomValues(new Uint8Array(SALT_BYTES))
  const saltB64 = bufToB64(salt.buffer)
  const gcmKey = await deriveKeyAsGCM(passphrase, saltB64)
  const keyCheckB64 = await encryptString('ok', gcmKey)
  return { saltB64, keyCheckB64 }
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

// Meldet, ob ein Blob bereits verschlüsselt ist (trägt den Magic-Header).
export function isEncryptedBytes(blob: Uint8Array): boolean {
  return startsWithMagic(blob)
}

export async function encryptBytes(content: Uint8Array, dek: CryptoKey): Promise<Uint8Array> {
  const iv = crypto.getRandomValues(new Uint8Array(IV_BYTES))
  const ciphertext = await crypto.subtle.encrypt({ name: 'AES-GCM', iv }, dek, content)
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
  const iv = blob.slice(BLOB_MAGIC.length, BLOB_MAGIC.length + IV_BYTES)
  const data = blob.slice(BLOB_MAGIC.length + IV_BYTES)
  const plain = await crypto.subtle.decrypt({ name: 'AES-GCM', iv }, dek, data)
  return new Uint8Array(plain)
}

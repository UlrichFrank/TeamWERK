// WebCrypto-based envelope encryption for member sensitive data.
// All keys are derived/used in the browser; the server only sees ciphertext.

const PBKDF2_ITERATIONS = 600_000;
const SALT_BYTES = 32;
const IV_BYTES = 12;

// --- Base64 helpers ---

export function bufToB64(buf: ArrayBuffer): string {
  return btoa(String.fromCharCode(...new Uint8Array(buf)));
}

export function b64ToBuf(b64: string): ArrayBuffer {
  const binary = atob(b64);
  const buf = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) buf[i] = binary.charCodeAt(i);
  return buf.buffer;
}

// --- Key derivation ---

export async function deriveKey(passphrase: string, saltB64: string): Promise<CryptoKey> {
  const salt = b64ToBuf(saltB64);
  const raw = new TextEncoder().encode(passphrase);
  const base = await crypto.subtle.importKey("raw", raw, "PBKDF2", false, ["deriveKey"]);
  return crypto.subtle.deriveKey(
    { name: "PBKDF2", hash: "SHA-256", salt, iterations: PBKDF2_ITERATIONS },
    base,
    { name: "AES-KW", length: 256 },
    true,
    ["wrapKey", "unwrapKey"],
  );
}

export async function deriveKeyFromPassword(password: string, saltB64: string): Promise<CryptoKey> {
  return deriveKey(password, saltB64);
}

// --- DEK generation ---

export async function generateDEK(): Promise<CryptoKey> {
  return crypto.subtle.generateKey({ name: "AES-GCM", length: 256 }, true, ["encrypt", "decrypt"]);
}

// --- DEK wrapping/unwrapping (AES-KW) ---

export async function wrapKey(dek: CryptoKey, wrappingKey: CryptoKey): Promise<string> {
  const wrapped = await crypto.subtle.wrapKey("raw", dek, wrappingKey, "AES-KW");
  return bufToB64(wrapped);
}

export async function unwrapKey(wrappedB64: string, wrappingKey: CryptoKey): Promise<CryptoKey> {
  const wrapped = b64ToBuf(wrappedB64);
  return crypto.subtle.unwrapKey(
    "raw",
    wrapped,
    wrappingKey,
    "AES-KW",
    { name: "AES-GCM", length: 256 },
    true,
    ["encrypt", "decrypt"],
  );
}

// --- Data encryption/decryption (AES-GCM, IV prepended) ---

export async function encrypt(payload: object, dek: CryptoKey): Promise<string> {
  const iv = crypto.getRandomValues(new Uint8Array(IV_BYTES));
  const data = new TextEncoder().encode(JSON.stringify(payload));
  const ciphertext = await crypto.subtle.encrypt({ name: "AES-GCM", iv }, dek, data);
  const out = new Uint8Array(IV_BYTES + ciphertext.byteLength);
  out.set(iv, 0);
  out.set(new Uint8Array(ciphertext), IV_BYTES);
  return bufToB64(out.buffer);
}

export async function decrypt(ciphertextB64: string, dek: CryptoKey): Promise<object> {
  const buf = new Uint8Array(b64ToBuf(ciphertextB64));
  const iv = buf.slice(0, IV_BYTES);
  const data = buf.slice(IV_BYTES);
  const plain = await crypto.subtle.decrypt({ name: "AES-GCM", iv }, dek, data);
  return JSON.parse(new TextDecoder().decode(plain));
}

// --- Vault passphrase verification ---

export async function verifyVaultPassphrase(
  passphrase: string,
  saltB64: string,
  keyCheckB64: string,
): Promise<boolean> {
  try {
    const gcmKey = await deriveKeyAsGCM(passphrase, saltB64);
    const result = await decryptString(keyCheckB64, gcmKey);
    return result === "ok";
  } catch {
    return false;
  }
}

// Derive a key for AES-GCM (for the key-check value verification and sessionStorage key)
export async function deriveKeyAsGCM(passphrase: string, saltB64: string): Promise<CryptoKey> {
  const salt = b64ToBuf(saltB64);
  const raw = new TextEncoder().encode(passphrase);
  const base = await crypto.subtle.importKey("raw", raw, "PBKDF2", false, ["deriveKey"]);
  return crypto.subtle.deriveKey(
    { name: "PBKDF2", hash: "SHA-256", salt, iterations: PBKDF2_ITERATIONS },
    base,
    { name: "AES-GCM", length: 256 },
    true,
    ["encrypt", "decrypt"],
  );
}

// --- Setup: generate salt + key-check for initial vault setup ---

export async function generateVaultSetup(passphrase: string): Promise<{
  saltB64: string;
  keyCheckB64: string;
}> {
  const salt = crypto.getRandomValues(new Uint8Array(SALT_BYTES));
  const saltB64 = bufToB64(salt.buffer);
  const gcmKey = await deriveKeyAsGCM(passphrase, saltB64);
  const keyCheckB64 = await encryptString("ok", gcmKey);
  return { saltB64, keyCheckB64 };
}

// --- String encrypt/decrypt (used for key-check) ---

async function encryptString(plain: string, key: CryptoKey): Promise<string> {
  const iv = crypto.getRandomValues(new Uint8Array(IV_BYTES));
  const data = new TextEncoder().encode(plain);
  const ciphertext = await crypto.subtle.encrypt({ name: "AES-GCM", iv }, key, data);
  const out = new Uint8Array(IV_BYTES + ciphertext.byteLength);
  out.set(iv, 0);
  out.set(new Uint8Array(ciphertext), IV_BYTES);
  return bufToB64(out.buffer);
}

async function decryptString(ciphertextB64: string, key: CryptoKey): Promise<string> {
  const buf = new Uint8Array(b64ToBuf(ciphertextB64));
  const iv = buf.slice(0, IV_BYTES);
  const data = buf.slice(IV_BYTES);
  const plain = await crypto.subtle.decrypt({ name: "AES-GCM", iv }, key, data);
  return new TextDecoder().decode(plain);
}

// --- Member salt generation ---

export function generateSalt(): string {
  const salt = crypto.getRandomValues(new Uint8Array(SALT_BYTES));
  return bufToB64(salt.buffer);
}

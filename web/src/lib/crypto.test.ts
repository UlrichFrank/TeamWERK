import { describe, it, expect } from 'vitest'
import {
  bufToB64,
  b64ToBuf,
  deriveKey,
  generateDEK,
  wrapKey,
  unwrapKey,
  encrypt,
  decrypt,
  generateVaultSetup,
  verifyVaultPassphrase,
  generateSalt,
  encryptBytes,
  decryptBytes,
  isEncryptedBytes,
} from './crypto'

const SALT = generateSalt()

describe('Base64-Helfer', () => {
  it('roundtrip beliebiger Bytes', () => {
    const bytes = new Uint8Array([0, 1, 2, 254, 255, 128, 64])
    expect(new Uint8Array(b64ToBuf(bufToB64(bytes.buffer)))).toEqual(bytes)
  })
})

describe('Envelope-Encryption (DEK + AES-GCM)', () => {
  it('verschlüsselt und entschlüsselt ein Objekt verlustfrei', async () => {
    const dek = await generateDEK()
    const payload = { iban: 'DE89370400440532013000', kontoinhaber: 'Max Mustermann' }
    const ct = await encrypt(payload, dek)
    expect(ct).not.toContain('DE89370400440532013000')
    expect(await decrypt(ct, dek)).toEqual(payload)
  })

  it('erzeugt pro Aufruf einen anderen Ciphertext (zufälliger IV)', async () => {
    const dek = await generateDEK()
    const a = await encrypt({ iban: 'DE89' }, dek)
    const b = await encrypt({ iban: 'DE89' }, dek)
    expect(a).not.toEqual(b)
  })

  it('lehnt manipulierten Ciphertext ab (GCM-Authentifizierung)', async () => {
    const dek = await generateDEK()
    const ct = await encrypt({ iban: 'DE89' }, dek)
    const raw = new Uint8Array(b64ToBuf(ct))
    raw[raw.length - 1] ^= 0xff // letztes Byte (GCM-Tag) kippen
    await expect(decrypt(bufToB64(raw.buffer), dek)).rejects.toBeDefined()
  })

  it('entschlüsselt nicht mit einem anderen DEK', async () => {
    const dekA = await generateDEK()
    const dekB = await generateDEK()
    const ct = await encrypt({ iban: 'DE89' }, dekA)
    await expect(decrypt(ct, dekB)).rejects.toBeDefined()
  })
})

describe('DEK-Wrapping (AES-KW) und Passphrase-Ableitung', () => {
  it('wrappt und entwrappt einen DEK über den abgeleiteten Gruppen-Schlüssel', async () => {
    const groupKey = await deriveKey('geheime tresor passphrase', SALT)
    const dek = await generateDEK()
    const payload = { iban: 'DE89370400440532013000' }
    const ct = await encrypt(payload, dek)

    const wrapped = await wrapKey(dek, groupKey)

    // Frischer Schlüssel aus derselben Passphrase + Salt (PBKDF2 ist deterministisch).
    const groupKeyAgain = await deriveKey('geheime tresor passphrase', SALT)
    const dekAgain = await unwrapKey(wrapped, groupKeyAgain)
    expect(await decrypt(ct, dekAgain)).toEqual(payload)
  })

  it('entwrappt nicht mit falscher Passphrase', async () => {
    const groupKey = await deriveKey('richtig', SALT)
    const wrapped = await wrapKey(await generateDEK(), groupKey)
    const wrongKey = await deriveKey('falsch', SALT)
    await expect(unwrapKey(wrapped, wrongKey)).rejects.toBeDefined()
  })
})

describe('Binäre Blobs (Mandat-PDFs)', () => {
  it('verschlüsselt und entschlüsselt einen PDF-Blob verlustfrei', async () => {
    const dek = await generateDEK()
    const pdf = new TextEncoder().encode('%PDF-1.4\n… binär …\x00\x01\x02')
    const enc = await encryptBytes(pdf, dek)
    expect(isEncryptedBytes(enc)).toBe(true)
    expect(new TextDecoder().decode(enc.slice(0, 4))).not.toBe('%PDF')
    expect(Array.from(await decryptBytes(enc, dek))).toEqual(Array.from(pdf))
  })

  it('erkennt Klartext (kein Magic-Header) und lehnt ihn ab', async () => {
    const dek = await generateDEK()
    const plain = new TextEncoder().encode('%PDF unverschlüsselt')
    expect(isEncryptedBytes(plain)).toBe(false)
    await expect(decryptBytes(plain, dek)).rejects.toBeDefined()
  })

  it('lehnt einen manipulierten Blob ab', async () => {
    const dek = await generateDEK()
    const enc = await encryptBytes(new TextEncoder().encode('mandat'), dek)
    enc[enc.length - 1] ^= 0xff
    await expect(decryptBytes(enc, dek)).rejects.toBeDefined()
  })
})

describe('Tresor-Einrichtung & Key-Check', () => {
  it('verifiziert die korrekte Passphrase und weist die falsche ab', async () => {
    const { saltB64, keyCheckB64 } = await generateVaultSetup('correct horse battery staple')
    expect(await verifyVaultPassphrase('correct horse battery staple', saltB64, keyCheckB64)).toBe(
      true,
    )
    expect(await verifyVaultPassphrase('falsch', saltB64, keyCheckB64)).toBe(false)
  })
})

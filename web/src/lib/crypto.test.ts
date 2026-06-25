import { describe, it, expect } from 'vitest'
import {
  bufToB64,
  b64ToBuf,
  generateDEK,
  generateGroupKeypair,
  exportPublicKey,
  importPublicKey,
  wrapDEK,
  unwrapDEK,
  encrypt,
  decrypt,
  deriveKEK,
  encryptPrivateKey,
  decryptPrivateKey,
  generateVaultSetup,
  verifyVaultPassphrase,
  generateSalt,
  encryptBytes,
  decryptBytes,
  isEncryptedBytes,
} from './crypto'

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
    raw[raw.length - 1] ^= 0xff
    await expect(decrypt(bufToB64(raw.buffer), dek)).rejects.toBeDefined()
  })
})

describe('Gruppen-Keypair: Schreiben (public) / Lesen (private)', () => {
  it('wrappt einen DEK an den öffentlichen Schlüssel und entwrappt mit dem privaten', async () => {
    const kp = await generateGroupKeypair()
    const dek = await generateDEK()
    const payload = { iban: 'DE89370400440532013000' }
    const ct = await encrypt(payload, dek)

    // Schreiben: nur der (re-importierte, öffentliche) Schlüssel nötig.
    const pubB64 = await exportPublicKey(kp.publicKey)
    const pub = await importPublicKey(pubB64)
    const wrapped = await wrapDEK(dek, pub)

    // Lesen: privater Schlüssel.
    const dek2 = await unwrapDEK(wrapped, kp.privateKey)
    expect(await decrypt(ct, dek2)).toEqual(payload)
  })
})

describe('Privatschlüssel-Schutz unter der Passphrase', () => {
  it('verschlüsselt den privaten Schlüssel und stellt ihn mit korrekter Passphrase wieder her', async () => {
    const salt = generateSalt()
    const kek = await deriveKEK('correct horse battery staple', salt)
    const kp = await generateGroupKeypair()
    const encPriv = await encryptPrivateKey(kp.privateKey, kek)

    // Mit korrekter Passphrase: Privatschlüssel zurückgewinnen und einen Wrap entwrappen.
    const kekAgain = await deriveKEK('correct horse battery staple', salt)
    const priv = await decryptPrivateKey(encPriv, kekAgain)
    const dek = await generateDEK()
    const wrapped = await wrapDEK(dek, kp.publicKey)
    const dek2 = await unwrapDEK(wrapped, priv)
    expect(await decrypt(await encrypt({ x: 1 }, dek), dek2)).toEqual({ x: 1 })
  })

  it('scheitert mit falscher Passphrase', async () => {
    const salt = generateSalt()
    const kek = await deriveKEK('richtig', salt)
    const kp = await generateGroupKeypair()
    const encPriv = await encryptPrivateKey(kp.privateKey, kek)
    const wrongKek = await deriveKEK('falsch', salt)
    await expect(decryptPrivateKey(encPriv, wrongKek)).rejects.toBeDefined()
  })
})

describe('Tresor-Einrichtung & Key-Check (End-to-End)', () => {
  it('Setup → Schreiben (public) → Entsperren (passphrase) → Lesen', async () => {
    const setup = await generateVaultSetup('eine starke passphrase')

    // Schreiben mit dem ausgelieferten öffentlichen Schlüssel.
    const pub = await importPublicKey(setup.groupPublicKey)
    const dek = await generateDEK()
    const ciphertext = await encrypt({ iban: 'DE89' }, dek)
    const dekEnc = await wrapDEK(dek, pub)

    // Entsperren + Lesen.
    expect(
      await verifyVaultPassphrase('eine starke passphrase', setup.vorstandKdfSalt, setup.vorstandKeyCheck),
    ).toBe(true)
    const kek = await deriveKEK('eine starke passphrase', setup.vorstandKdfSalt)
    const priv = await decryptPrivateKey(setup.groupPrivateKeyEnc, kek)
    const dekBack = await unwrapDEK(dekEnc, priv)
    expect(await decrypt(ciphertext, dekBack)).toEqual({ iban: 'DE89' })
  })

  it('weist die falsche Passphrase ab', async () => {
    const setup = await generateVaultSetup('correct')
    expect(await verifyVaultPassphrase('falsch', setup.vorstandKdfSalt, setup.vorstandKeyCheck)).toBe(false)
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
})

import { describe, it, expect } from 'vitest'
import { generateGroupKeypair, generateDEK, encrypt, wrapDEK } from './crypto'
import { decryptBankData } from './bankCrypto'

describe('bankCrypto', () => {
  it('decryptBankData entschlüsselt einen an den öffentlichen Schlüssel erstellten Envelope', async () => {
    const kp = await generateGroupKeypair()
    const data = { iban: 'DE89370400440532013000', account_holder: 'Max Mustermann' }

    // Schreibseite (wie encryptBankData, aber ohne API): an den öffentlichen Schlüssel.
    const dek = await generateDEK()
    const env = {
      bank_ciphertext: await encrypt(data, dek),
      bank_dek_enc: await wrapDEK(dek, kp.publicKey),
    }

    // Leseseite: mit dem privaten Schlüssel.
    expect(await decryptBankData(env, kp.privateKey)).toEqual(data)
  })
})

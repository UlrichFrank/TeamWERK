import { api } from './api'
import { generateDEK, encrypt, decrypt, wrapDEK, unwrapDEK, importPublicKey } from './crypto'

// Bankdaten-Envelope (Modell B): clientseitige Ver-/Entschlüsselung der Mitglieds-Bankdaten.
// Schreiben braucht nur den öffentlichen Gruppen-Schlüssel (kein Tresor-Entsperren); Lesen
// braucht den entschlüsselten privaten Schlüssel aus dem VaultContext.

export interface BankData {
  iban: string
  account_holder: string
}

export interface BankEnvelope {
  bank_ciphertext: string
  bank_dek_enc: string
}

// Verschlüsselt Bankdaten an den öffentlichen Gruppen-Schlüssel. Wirft, wenn der Tresor
// noch nicht eingerichtet ist.
export async function encryptBankData(data: BankData): Promise<BankEnvelope> {
  const { data: cfg } = await api.get<{ configured: boolean; group_public_key: string }>(
    '/encryption-pubkey',
  )
  if (!cfg.configured || !cfg.group_public_key) {
    throw new Error('Bankdaten-Tresor ist noch nicht eingerichtet.')
  }
  const pub = await importPublicKey(cfg.group_public_key)
  const dek = await generateDEK()
  return {
    bank_ciphertext: await encrypt(data, dek),
    bank_dek_enc: await wrapDEK(dek, pub),
  }
}

// Entschlüsselt einen Bankdaten-Envelope mit dem privaten Gruppen-Schlüssel (Tresor entsperrt).
export async function decryptBankData(env: BankEnvelope, privateKey: CryptoKey): Promise<BankData> {
  const dek = await unwrapDEK(env.bank_dek_enc, privateKey)
  return (await decrypt(env.bank_ciphertext, dek)) as BankData
}

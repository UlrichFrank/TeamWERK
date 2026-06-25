import { api } from './api'
import {
  generateDEK,
  encrypt,
  decrypt,
  encryptBytes,
  decryptBytes,
  wrapDEK,
  unwrapDEK,
  importPublicKey,
} from './crypto'

// Envelope-Krypto (Modell B): clientseitige Ver-/Entschlüsselung von Bank-/SEPA-PII.
// Schreiben braucht nur den öffentlichen Gruppen-Schlüssel (kein Tresor-Entsperren); Lesen
// braucht den entschlüsselten privaten Schlüssel aus dem VaultContext.

// --- Generischer Kern ---

// Lädt den öffentlichen Gruppen-Schlüssel; wirft, wenn der Tresor nicht eingerichtet ist.
async function fetchGroupPublicKey(): Promise<CryptoKey> {
  const { data: cfg } = await api.get<{ configured: boolean; group_public_key: string }>(
    '/encryption-pubkey',
  )
  if (!cfg.configured || !cfg.group_public_key) {
    throw new Error('Bankdaten-Tresor ist noch nicht eingerichtet.')
  }
  return importPublicKey(cfg.group_public_key)
}

// Verschlüsselt ein Objekt an den öffentlichen Gruppen-Schlüssel → {ciphertext, dekEnc}.
async function encryptToGroup(obj: object): Promise<{ ciphertext: string; dekEnc: string }> {
  const pub = await fetchGroupPublicKey()
  const dek = await generateDEK()
  return { ciphertext: await encrypt(obj, dek), dekEnc: await wrapDEK(dek, pub) }
}

async function decryptFromGroup<T>(ciphertext: string, dekEnc: string, privateKey: CryptoKey): Promise<T> {
  const dek = await unwrapDEK(dekEnc, privateKey)
  return (await decrypt(ciphertext, dek)) as T
}

// --- Datei-Blobs (SEPA-Mandat-PDFs) ---

// Verschlüsselt Datei-Bytes an den Gruppen-Schlüssel → {blob (Ciphertext mit Magic), dekEnc}.
export async function encryptFile(bytes: Uint8Array): Promise<{ blob: Uint8Array; dekEnc: string }> {
  const pub = await fetchGroupPublicKey()
  const dek = await generateDEK()
  return { blob: await encryptBytes(bytes, dek), dekEnc: await wrapDEK(dek, pub) }
}

// Entschlüsselt einen Datei-Blob mit dem privaten Gruppen-Schlüssel (Tresor entsperrt).
export async function decryptFile(blob: Uint8Array, dekEnc: string, privateKey: CryptoKey): Promise<Uint8Array> {
  const dek = await unwrapDEK(dekEnc, privateKey)
  return decryptBytes(blob, dek)
}

// --- Mitglieds-Bankdaten ---

export interface BankData {
  iban: string
  account_holder: string
}

export interface BankEnvelope {
  bank_ciphertext: string
  bank_dek_enc: string
}

export async function encryptBankData(data: BankData): Promise<BankEnvelope> {
  const { ciphertext, dekEnc } = await encryptToGroup(data)
  return { bank_ciphertext: ciphertext, bank_dek_enc: dekEnc }
}

export async function decryptBankData(env: BankEnvelope, privateKey: CryptoKey): Promise<BankData> {
  return decryptFromGroup<BankData>(env.bank_ciphertext, env.bank_dek_enc, privateKey)
}

// --- Vereins-SEPA-Stammdaten ---

export interface ClubSepaData {
  glaeubiger_id: string
  iban: string
  bic: string
  kontoinhaber: string
}

export interface ClubSepaEnvelope {
  sepa_ciphertext: string
  sepa_dek_enc: string
}

export async function encryptClubSepa(data: ClubSepaData): Promise<ClubSepaEnvelope> {
  const { ciphertext, dekEnc } = await encryptToGroup(data)
  return { sepa_ciphertext: ciphertext, sepa_dek_enc: dekEnc }
}

export async function decryptClubSepa(env: ClubSepaEnvelope, privateKey: CryptoKey): Promise<ClubSepaData> {
  return decryptFromGroup<ClubSepaData>(env.sepa_ciphertext, env.sepa_dek_enc, privateKey)
}

import { createContext, useCallback, useContext, useEffect, useRef, useState } from 'react'
import { api } from '../lib/api'
import { b64ToBuf, bufToB64, deriveKEK, decryptPrivateKey, verifyVaultPassphrase } from '../lib/crypto'

// Zero-Knowledge-Tresor (Modell B): hält den entschlüsselten privaten Gruppen-Schlüssel
// (RSA-OAEP) flüchtig im Browser, um Mitglieds-DEKs zu entwrappen (Lesen). Das Schreiben
// braucht den Tresor nicht — dafür genügt der öffentliche Schlüssel (siehe groupPublicKey()).
// Die Passphrase verlässt den Browser nie; nach 30 Minuten Inaktivität (und beim
// Tab-Schließen) wird der Schlüssel verworfen.

const SESSION_KEY = 'vk' // pkcs8(GroupPriv) base64 — nur Session, flüchtig
const INACTIVITY_MS = 30 * 60 * 1000

interface EncryptionConfig {
  configured: boolean
  group_public_key: string
  group_private_key_enc: string
  vorstand_kdf_salt: string
  vorstand_key_check: string
}

interface VaultContextValue {
  isUnlocked: boolean
  privateKey: CryptoKey | null
  unlock: (passphrase: string) => Promise<boolean>
  lock: () => void
}

const VaultContext = createContext<VaultContextValue>({
  isUnlocked: false,
  privateKey: null,
  unlock: async () => false,
  lock: () => {},
})

export function useVault() {
  return useContext(VaultContext)
}

async function importPriv(pkcs8B64: string): Promise<CryptoKey> {
  return crypto.subtle.importKey('pkcs8', b64ToBuf(pkcs8B64), { name: 'RSA-OAEP', hash: 'SHA-256' }, true, [
    'unwrapKey',
  ])
}

export function VaultProvider({ children }: { children: React.ReactNode }) {
  const [privateKey, setPrivateKey] = useState<CryptoKey | null>(null)
  const inactivityTimer = useRef<ReturnType<typeof setTimeout> | null>(null)

  const lock = useCallback(() => {
    sessionStorage.removeItem(SESSION_KEY)
    setPrivateKey(null)
    if (inactivityTimer.current) clearTimeout(inactivityTimer.current)
  }, [])

  const resetInactivityTimer = useCallback(() => {
    if (inactivityTimer.current) clearTimeout(inactivityTimer.current)
    inactivityTimer.current = setTimeout(lock, INACTIVITY_MS)
  }, [lock])

  // Schlüssel beim Mount aus sessionStorage wiederherstellen (Navigation/Reload).
  useEffect(() => {
    const stored = sessionStorage.getItem(SESSION_KEY)
    if (!stored) return
    importPriv(stored)
      .then(key => {
        setPrivateKey(key)
        resetInactivityTimer()
      })
      .catch(() => lock())
  }, [lock, resetInactivityTimer])

  // Inaktivitäts-Timer bei Interaktion zurücksetzen, solange der Tresor offen ist.
  useEffect(() => {
    if (!privateKey) return
    const events = ['mousemove', 'keydown', 'click', 'touchstart']
    events.forEach(e => window.addEventListener(e, resetInactivityTimer, { passive: true }))
    return () => events.forEach(e => window.removeEventListener(e, resetInactivityTimer))
  }, [privateKey, resetInactivityTimer])

  const unlock = useCallback(
    async (passphrase: string): Promise<boolean> => {
      try {
        const { data } = await api.get<EncryptionConfig>('/admin/encryption-config')
        if (!data.configured) return false
        const ok = await verifyVaultPassphrase(passphrase, data.vorstand_kdf_salt, data.vorstand_key_check)
        if (!ok) return false
        const kek = await deriveKEK(passphrase, data.vorstand_kdf_salt)
        const priv = await decryptPrivateKey(data.group_private_key_enc, kek)
        sessionStorage.setItem(SESSION_KEY, bufToB64(await crypto.subtle.exportKey('pkcs8', priv)))
        setPrivateKey(priv)
        resetInactivityTimer()
        return true
      } catch {
        return false
      }
    },
    [resetInactivityTimer],
  )

  return (
    <VaultContext.Provider value={{ isUnlocked: !!privateKey, privateKey, unlock, lock }}>
      {children}
    </VaultContext.Provider>
  )
}

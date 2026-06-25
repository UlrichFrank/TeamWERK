import { createContext, useCallback, useContext, useEffect, useRef, useState } from 'react'
import { api } from '../lib/api'
import { b64ToBuf, bufToB64, deriveKey, verifyVaultPassphrase } from '../lib/crypto'

// Zero-Knowledge-Tresor: hält den aus der geteilten Finance-Gruppen-Passphrase
// abgeleiteten AES-KW-Wrapping-Key flüchtig im Browser (sessionStorage), um die
// Mitglieds-DEKs zu wrappen/entwrappen. Die Passphrase selbst wird nie gespeichert
// oder an den Server gesendet; nach 30 Minuten Inaktivität (und beim Tab-Schließen)
// wird der Schlüssel verworfen.

const SESSION_KEY = 'vk'
const INACTIVITY_MS = 30 * 60 * 1000

interface EncryptionConfig {
  configured: boolean
  vorstand_kdf_salt: string
  vorstand_key_check: string
}

interface VaultContextValue {
  isUnlocked: boolean
  wrappingKey: CryptoKey | null
  unlock: (passphrase: string) => Promise<boolean>
  lock: () => void
}

const VaultContext = createContext<VaultContextValue>({
  isUnlocked: false,
  wrappingKey: null,
  unlock: async () => false,
  lock: () => {},
})

export function useVault() {
  return useContext(VaultContext)
}

async function importKwKey(b64: string): Promise<CryptoKey> {
  return crypto.subtle.importKey('raw', b64ToBuf(b64), { name: 'AES-KW', length: 256 }, true, [
    'wrapKey',
    'unwrapKey',
  ])
}

async function exportKey(key: CryptoKey): Promise<string> {
  return bufToB64(await crypto.subtle.exportKey('raw', key))
}

export function VaultProvider({ children }: { children: React.ReactNode }) {
  const [wrappingKey, setWrappingKey] = useState<CryptoKey | null>(null)
  const inactivityTimer = useRef<ReturnType<typeof setTimeout> | null>(null)

  const lock = useCallback(() => {
    sessionStorage.removeItem(SESSION_KEY)
    setWrappingKey(null)
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
    importKwKey(stored)
      .then(key => {
        setWrappingKey(key)
        resetInactivityTimer()
      })
      .catch(() => lock())
  }, [lock, resetInactivityTimer])

  // Inaktivitäts-Timer bei Interaktion zurücksetzen, solange der Tresor offen ist.
  useEffect(() => {
    if (!wrappingKey) return
    const events = ['mousemove', 'keydown', 'click', 'touchstart']
    events.forEach(e => window.addEventListener(e, resetInactivityTimer, { passive: true }))
    return () => events.forEach(e => window.removeEventListener(e, resetInactivityTimer))
  }, [wrappingKey, resetInactivityTimer])

  const unlock = useCallback(
    async (passphrase: string): Promise<boolean> => {
      try {
        const { data } = await api.get<EncryptionConfig>('/admin/encryption-config')
        if (!data.configured) return false
        const ok = await verifyVaultPassphrase(passphrase, data.vorstand_kdf_salt, data.vorstand_key_check)
        if (!ok) return false
        const key = await deriveKey(passphrase, data.vorstand_kdf_salt)
        sessionStorage.setItem(SESSION_KEY, await exportKey(key))
        setWrappingKey(key)
        resetInactivityTimer()
        return true
      } catch {
        return false
      }
    },
    [resetInactivityTimer],
  )

  return (
    <VaultContext.Provider value={{ isUnlocked: !!wrappingKey, wrappingKey, unlock, lock }}>
      {children}
    </VaultContext.Provider>
  )
}

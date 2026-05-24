import { createContext, useCallback, useContext, useEffect, useRef, useState } from 'react'
import { api } from '../lib/api'
import { b64ToBuf, bufToB64, deriveKey, deriveKeyAsGCM, verifyVaultPassphrase } from '../lib/crypto'

const SESSION_KEY = 'vk'
const SESSION_KW_KEY = 'vk_kw'
const INACTIVITY_MS = 30 * 60 * 1000

interface VaultContextValue {
  isUnlocked: boolean
  vaultKey: CryptoKey | null
  vaultKwKey: CryptoKey | null
  unlockVault: (passphrase: string) => Promise<boolean>
  lockVault: () => void
}

const VaultContext = createContext<VaultContextValue>({
  isUnlocked: false,
  vaultKey: null,
  vaultKwKey: null,
  unlockVault: async () => false,
  lockVault: () => {},
})

export function useVault() {
  return useContext(VaultContext)
}

async function importGcmKey(b64: string): Promise<CryptoKey> {
  const raw = b64ToBuf(b64)
  return crypto.subtle.importKey('raw', raw, { name: 'AES-GCM', length: 256 }, true, ['encrypt', 'decrypt'])
}

async function importKwKey(b64: string): Promise<CryptoKey> {
  const raw = b64ToBuf(b64)
  return crypto.subtle.importKey('raw', raw, { name: 'AES-KW', length: 256 }, true, ['wrapKey', 'unwrapKey'])
}

async function exportKey(key: CryptoKey): Promise<string> {
  const raw = await crypto.subtle.exportKey('raw', key)
  return bufToB64(raw)
}

export function VaultProvider({ children }: { children: React.ReactNode }) {
  const [vaultKey, setVaultKey] = useState<CryptoKey | null>(null)
  const [vaultKwKey, setVaultKwKey] = useState<CryptoKey | null>(null)
  const inactivityTimer = useRef<ReturnType<typeof setTimeout> | null>(null)

  const lockVault = useCallback(() => {
    sessionStorage.removeItem(SESSION_KEY)
    sessionStorage.removeItem(SESSION_KW_KEY)
    setVaultKey(null)
    setVaultKwKey(null)
    if (inactivityTimer.current) clearTimeout(inactivityTimer.current)
  }, [])

  const resetInactivityTimer = useCallback(() => {
    if (inactivityTimer.current) clearTimeout(inactivityTimer.current)
    inactivityTimer.current = setTimeout(lockVault, INACTIVITY_MS)
  }, [lockVault])

  // Restore keys from sessionStorage on mount
  useEffect(() => {
    const storedGcm = sessionStorage.getItem(SESSION_KEY)
    const storedKw = sessionStorage.getItem(SESSION_KW_KEY)
    if (storedGcm && storedKw) {
      Promise.all([importGcmKey(storedGcm), importKwKey(storedKw)])
        .then(([gcm, kw]) => {
          setVaultKey(gcm)
          setVaultKwKey(kw)
          resetInactivityTimer()
        })
        .catch(() => lockVault())
    }
  }, [lockVault, resetInactivityTimer])

  // Reset inactivity timer on user interaction when vault is open
  useEffect(() => {
    if (!vaultKey) return
    const events = ['mousemove', 'keydown', 'click', 'touchstart']
    events.forEach(e => window.addEventListener(e, resetInactivityTimer, { passive: true }))
    return () => events.forEach(e => window.removeEventListener(e, resetInactivityTimer))
  }, [vaultKey, resetInactivityTimer])

  const unlockVault = useCallback(async (passphrase: string): Promise<boolean> => {
    try {
      const { data } = await api.get<{ vorstand_kdf_salt: string; vorstand_key_check: string; configured: boolean }>(
        '/admin/encryption-config',
      )
      if (!data.configured) return false

      const ok = await verifyVaultPassphrase(passphrase, data.vorstand_kdf_salt, data.vorstand_key_check)
      if (!ok) return false

      const [gcmKey, kwKey] = await Promise.all([
        deriveKeyAsGCM(passphrase, data.vorstand_kdf_salt),
        deriveKey(passphrase, data.vorstand_kdf_salt),
      ])

      const [gcmB64, kwB64] = await Promise.all([exportKey(gcmKey), exportKey(kwKey)])
      sessionStorage.setItem(SESSION_KEY, gcmB64)
      sessionStorage.setItem(SESSION_KW_KEY, kwB64)

      setVaultKey(gcmKey)
      setVaultKwKey(kwKey)
      resetInactivityTimer()
      return true
    } catch {
      return false
    }
  }, [resetInactivityTimer])

  return (
    <VaultContext.Provider value={{ isUnlocked: !!vaultKey, vaultKey, vaultKwKey, unlockVault, lockVault }}>
      {children}
    </VaultContext.Provider>
  )
}

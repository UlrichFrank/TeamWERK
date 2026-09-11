import { createContext, useCallback, useContext, useEffect, useRef, useState } from 'react'
import { api } from '../lib/api'
import {
  deriveKEK,
  decryptPrivateKey,
  rewrapPrivateKeyForRotation,
  verifyVaultPassphrase,
} from '../lib/crypto'
import * as vaultKeyStore from '../lib/vaultKeyStore'

// Zero-Knowledge-Tresor (Modell B): hält den entschlüsselten privaten Gruppen-Schlüssel
// (RSA-OAEP) flüchtig im Browser, um Mitglieds-DEKs zu entwrappen (Lesen). Das Schreiben
// braucht den Tresor nicht — dafür genügt der öffentliche Schlüssel (siehe groupPublicKey()).
// Die Passphrase verlässt den Browser nie; nach 30 Minuten Inaktivität (und beim
// Tab-Schließen) wird der Schlüssel verworfen.
//
// Der Schlüssel liegt als NICHT-EXPORTIERBARES CryptoKey-Objekt in IndexedDB
// (`lib/vaultKeyStore`), nicht mehr als PKCS8-Base64 in `sessionStorage`: ein XSS kann ihn
// so zwar benutzen, solange die Seite offen ist, aber nicht exfiltrieren. `sessionStorage`
// trägt nur noch den Ablauf-Zeitstempel — und genau der hält die alte Semantik aufrecht:
// IndexedDB überlebt anders als `sessionStorage` das Schließen des Tabs, ein frischer Tab
// findet aber keinen gültigen Zeitstempel und verwirft den Schlüssel (fail-closed).

const LEGACY_SESSION_KEY = 'vk' // Altformat: pkcs8(GroupPriv) base64 — wird beim Mount entsorgt
const DEADLINE_KEY = 'vk_until' // Ablauf-Zeitstempel (ms) — kein Schlüsselmaterial
const INACTIVITY_MS = 30 * 60 * 1000
// Der Zeitstempel wird bei jeder Interaktion neu gesetzt (mousemove feuert ~60×/s); ein
// Schreibvorgang lohnt erst, wenn er den gespeicherten Wert spürbar verschiebt.
const DEADLINE_SLACK_MS = 10_000

interface EncryptionConfig {
  configured: boolean
  group_public_key: string
  group_private_key_enc: string
  vorstand_kdf_salt: string
  vorstand_key_check: string
}

export interface RotationPayload {
  groupPrivateKeyEnc: string
  vorstandKdfSalt: string
  vorstandKeyCheck: string
}

interface VaultContextValue {
  isUnlocked: boolean
  privateKey: CryptoKey | null
  unlock: (passphrase: string) => Promise<boolean>
  // Rotation braucht die PKCS8-Bytes und damit einen exportierbaren Schlüssel — den
  // gecachten gibt es bewusst nicht. Deshalb die aktuelle Passphrase erneut eingeben;
  // sie entschlüsselt den Privatschlüssel einmalig und exportierbar, nur für diesen Aufruf.
  rotate: (currentPassphrase: string, newPassphrase: string) => Promise<RotationPayload | null>
  lock: () => void
}

const VaultContext = createContext<VaultContextValue>({
  isUnlocked: false,
  privateKey: null,
  unlock: async () => false,
  rotate: async () => null,
  lock: () => {},
})

// Context-Datei exportiert bewusst Provider + Hook zusammen (idiomatisches
// Muster); der Hook aus einer eigenen Datei würde alle Importeure umhängen.
// eslint-disable-next-line react-refresh/only-export-components
export function useVault() {
  return useContext(VaultContext)
}

function readDeadline(): number | null {
  const raw = sessionStorage.getItem(DEADLINE_KEY)
  if (!raw) return null
  const value = Number(raw)
  return Number.isFinite(value) ? value : null
}

export function VaultProvider({ children }: { children: React.ReactNode }) {
  const [privateKey, setPrivateKey] = useState<CryptoKey | null>(null)
  const inactivityTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  // Spiegel des Schlüssel-Zustands für Event-Handler: der Klick auf „Sperren" läuft
  // nach lock() weiter zum window-Listener (resetInactivityTimer, Bubbling) — der darf
  // die gerade gelöschte Frist nicht neu setzen. React-State wäre in diesem Tick noch
  // der alte Wert, der Ref ist es nicht.
  const hasKeyRef = useRef(false)

  const lock = useCallback(() => {
    hasKeyRef.current = false
    sessionStorage.removeItem(DEADLINE_KEY)
    void vaultKeyStore.clear()
    setPrivateKey(null)
    if (inactivityTimer.current) clearTimeout(inactivityTimer.current)
  }, [])

  const resetInactivityTimer = useCallback(() => {
    if (!hasKeyRef.current) return
    if (inactivityTimer.current) clearTimeout(inactivityTimer.current)
    const deadline = Date.now() + INACTIVITY_MS
    const stored = readDeadline()
    if (stored === null || deadline - stored > DEADLINE_SLACK_MS) {
      sessionStorage.setItem(DEADLINE_KEY, String(deadline))
    }
    inactivityTimer.current = setTimeout(lock, INACTIVITY_MS)
  }, [lock])

  // Schlüssel beim Mount aus IndexedDB wiederherstellen (Navigation/Reload) — aber nur,
  // solange der Zeitstempel dieser Sitzung noch gilt. Fehlt er (neuer Tab) oder ist er
  // abgelaufen, wird der Schlüssel verworfen statt wiederbelebt.
  useEffect(() => {
    let cancelled = false
    sessionStorage.removeItem(LEGACY_SESSION_KEY) // Altformat (exportierbares PKCS8) entsorgen
    const deadline = readDeadline()
    if (deadline === null || Date.now() >= deadline) {
      sessionStorage.removeItem(DEADLINE_KEY)
      void vaultKeyStore.clear()
      return
    }
    vaultKeyStore
      .get()
      .then(key => {
        if (cancelled || !key) return
        hasKeyRef.current = true
        setPrivateKey(key)
        resetInactivityTimer()
      })
      .catch(() => lock())
    return () => {
      cancelled = true
    }
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
        // extractable=false (Default): der gecachte Schlüssel ist nicht auslesbar.
        const priv = await decryptPrivateKey(data.group_private_key_enc, kek)
        // Ohne IndexedDB (Private Mode) ein No-op: der Schlüssel lebt dann nur im
        // React-State, ein Reload sperrt den Tresor. Kein Sicherheitsverlust.
        await vaultKeyStore.put(priv)
        hasKeyRef.current = true
        setPrivateKey(priv)
        resetInactivityTimer()
        return true
      } catch {
        return false
      }
    },
    [resetInactivityTimer],
  )

  const rotate = useCallback(
    async (currentPassphrase: string, newPassphrase: string): Promise<RotationPayload | null> => {
      try {
        const { data } = await api.get<EncryptionConfig>('/admin/encryption-config')
        if (!data.configured) return null
        const ok = await verifyVaultPassphrase(
          currentPassphrase,
          data.vorstand_kdf_salt,
          data.vorstand_key_check,
        )
        if (!ok) return null
        const kek = await deriveKEK(currentPassphrase, data.vorstand_kdf_salt)
        // Nur hier exportierbar: `rewrapPrivateKeyForRotation` schreibt die PKCS8-Bytes
        // unter der neuen Passphrase neu heraus. Der Schlüssel bleibt lokal und wird nach
        // dem Aufruf verworfen — gecacht wird weiterhin nur die nicht-exportierbare Variante.
        const priv = await decryptPrivateKey(data.group_private_key_enc, kek, { extractable: true })
        return await rewrapPrivateKeyForRotation(priv, newPassphrase)
      } catch {
        return null
      }
    },
    [],
  )

  return (
    <VaultContext.Provider value={{ isUnlocked: !!privateKey, privateKey, unlock, rotate, lock }}>
      {children}
    </VaultContext.Provider>
  )
}

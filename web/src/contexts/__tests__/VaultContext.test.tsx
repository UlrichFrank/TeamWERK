import 'fake-indexeddb/auto'
import { IDBFactory } from 'fake-indexeddb'
import { useState } from 'react'
import { describe, test, expect, vi, beforeAll, beforeEach } from 'vitest'
import { render, screen, waitFor, cleanup, act } from '@testing-library/react'
import { VaultProvider, useVault } from '../VaultContext'
import { generateVaultSetup, type VaultSetup } from '../../lib/crypto'
import * as vaultKeyStore from '../../lib/vaultKeyStore'

const mockGet = vi.fn()
vi.mock('../../lib/api', () => ({ api: { get: (...args: unknown[]) => mockGet(...args) } }))

const PASSPHRASE = 'eine starke passphrase'
const DEADLINE_KEY = 'vk_until'

let setup: VaultSetup
let captured: CryptoKey | null = null

function Probe() {
  const { isUnlocked, privateKey, unlock } = useVault()
  captured = privateKey
  return (
    <div>
      <span data-testid="state">{isUnlocked ? 'unlocked' : 'locked'}</span>
      <button onClick={() => void unlock(PASSPHRASE)}>entsperren</button>
    </div>
  )
}

function mount() {
  return render(
    <VaultProvider>
      <Probe />
    </VaultProvider>,
  )
}

async function unlockVault() {
  await act(async () => {
    screen.getByText('entsperren').click()
  })
  await waitFor(() => expect(screen.getByTestId('state').textContent).toBe('unlocked'))
}

// Alle Werte, die die App in Web-Storage ablegt — dort darf kein Schlüsselmaterial liegen.
function webStorageValues(): string[] {
  const out: string[] = []
  for (const storage of [sessionStorage, localStorage]) {
    for (let i = 0; i < storage.length; i++) {
      const k = storage.key(i)
      if (k) out.push(`${k}=${storage.getItem(k) ?? ''}`)
    }
  }
  return out
}

describe('VaultContext: Schlüssel-Caching ohne exportierbares Material', () => {
  beforeAll(async () => {
    setup = await generateVaultSetup(PASSPHRASE)
  }, 60_000)

  beforeEach(() => {
    // Frischer IndexedDB-Zustand je Test; sessionStorage leert das globale Setup.
    vi.stubGlobal('indexedDB', new IDBFactory())
    mockGet.mockReset()
    mockGet.mockResolvedValue({
      data: {
        configured: true,
        group_public_key: setup.groupPublicKey,
        group_private_key_enc: setup.groupPrivateKeyEnc,
        vorstand_kdf_salt: setup.vorstandKdfSalt,
        vorstand_key_check: setup.vorstandKeyCheck,
      },
    })
    captured = null
  })

  test('entsperrter Schlüssel ist nicht exportierbar und liegt in keinem Web-Storage', async () => {
    mount()
    await unlockVault()

    expect(captured).not.toBeNull()
    expect(captured!.extractable).toBe(false)
    await expect(crypto.subtle.exportKey('pkcs8', captured!)).rejects.toBeDefined()

    // Kein PKCS8-Base64 mehr in sessionStorage/localStorage — nur der Ablauf-Zeitstempel.
    expect(sessionStorage.getItem('vk')).toBeNull()
    const values = webStorageValues()
    expect(values).toEqual([`${DEADLINE_KEY}=${sessionStorage.getItem(DEADLINE_KEY)}`])
    expect(Number(sessionStorage.getItem(DEADLINE_KEY))).toBeGreaterThan(Date.now())

    // Der gecachte Schlüssel selbst ist ebenfalls nicht exportierbar.
    const stored = await vaultKeyStore.get()
    expect(stored).not.toBeNull()
    await expect(crypto.subtle.exportKey('pkcs8', stored!)).rejects.toBeDefined()
  }, 60_000)

  test('Re-Mount stellt den entsperrten Tresor aus IndexedDB wieder her', async () => {
    mount()
    await unlockVault()
    cleanup()

    mount()
    await waitFor(() => expect(screen.getByTestId('state').textContent).toBe('unlocked'))
    // Ohne erneutes Entsperren: der zweite Mount hat die Konfiguration gar nicht geladen.
    expect(mockGet).toHaveBeenCalledTimes(1)
    expect(captured?.extractable).toBe(false)
  }, 60_000)

  test('abgelaufener Zeitstempel sperrt und räumt den Schlüssel weg', async () => {
    mount()
    await unlockVault()
    cleanup()

    sessionStorage.setItem(DEADLINE_KEY, String(Date.now() - 1))
    mount()
    await waitFor(() => expect(vaultKeyStore.get()).resolves.toBeNull())
    expect(screen.getByTestId('state').textContent).toBe('locked')
    expect(sessionStorage.getItem(DEADLINE_KEY)).toBeNull()
  }, 60_000)

  test('fehlender Zeitstempel (neuer Tab) sperrt, obwohl IndexedDB den Schlüssel hätte', async () => {
    mount()
    await unlockVault()
    cleanup()

    sessionStorage.removeItem(DEADLINE_KEY) // frische Session, IndexedDB überlebt sie
    mount()
    await waitFor(() => expect(vaultKeyStore.get()).resolves.toBeNull())
    expect(screen.getByTestId('state').textContent).toBe('locked')
  }, 60_000)

  test('Alt-Eintrag sessionStorage["vk"] wird beim Mount entfernt', async () => {
    sessionStorage.setItem('vk', 'MIIEvAIBADANBgkq…altes-pkcs8-base64')
    mount()
    await waitFor(() => expect(sessionStorage.getItem('vk')).toBeNull())
  })

  test('lock() verwirft Schlüssel, Zeitstempel und IndexedDB-Eintrag', async () => {
    function LockProbe() {
      const { isUnlocked, unlock, lock } = useVault()
      return (
        <div>
          <span data-testid="state">{isUnlocked ? 'unlocked' : 'locked'}</span>
          <button onClick={() => void unlock(PASSPHRASE)}>entsperren</button>
          <button onClick={lock}>sperren</button>
        </div>
      )
    }
    render(
      <VaultProvider>
        <LockProbe />
      </VaultProvider>,
    )
    await unlockVault()

    await act(async () => {
      screen.getByText('sperren').click()
    })
    expect(screen.getByTestId('state').textContent).toBe('locked')
    expect(sessionStorage.getItem(DEADLINE_KEY)).toBeNull()
    await waitFor(() => expect(vaultKeyStore.get()).resolves.toBeNull())
  }, 60_000)

  test('falsche Passphrase entsperrt nicht und schreibt nichts', async () => {
    function WrongProbe() {
      const { isUnlocked, unlock } = useVault()
      const [result, setResult] = useState('')
      return (
        <div>
          <span data-testid="state">{isUnlocked ? 'unlocked' : 'locked'}</span>
          <span data-testid="result">{result}</span>
          <button onClick={() => void unlock('falsch').then(ok => setResult(ok ? 'ok' : 'fail'))}>
            entsperren
          </button>
        </div>
      )
    }
    render(
      <VaultProvider>
        <WrongProbe />
      </VaultProvider>,
    )
    await act(async () => {
      screen.getByText('entsperren').click()
    })
    await waitFor(() => expect(screen.getByTestId('result').textContent).toBe('fail'))
    expect(screen.getByTestId('state').textContent).toBe('locked')
    expect(await vaultKeyStore.get()).toBeNull()
    expect(sessionStorage.getItem(DEADLINE_KEY)).toBeNull()
  }, 60_000)
})
